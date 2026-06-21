package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dyike/Void/internal/model"
)

// ContainerRepo wraps Apple's `container` CLI.
type ContainerRepo struct {
	binary string
}

func NewContainerRepo() *ContainerRepo {
	return &ContainerRepo{binary: "container"}
}

func (r *ContainerRepo) executable() (string, error) {
	return resolveExecutable(r.binary, []string{
		"/usr/local/bin",
		"/opt/homebrew/bin",
		"/usr/bin",
		"/bin",
	})
}

func resolveExecutable(name string, extraDirs []string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("executable name is empty")
	}
	if strings.ContainsRune(name, os.PathSeparator) {
		if info, err := os.Stat(name); err == nil && isExecutableFile(info) {
			return name, nil
		}
		return "", fmt.Errorf("%s was not found", name)
	}
	if path, err := exec.LookPath(name); err == nil && path != "" {
		return path, nil
	}
	for _, dir := range extraDirs {
		path := dir + string(os.PathSeparator) + name
		if info, err := os.Stat(path); err == nil && isExecutableFile(info) {
			return path, nil
		}
	}
	return "", fmt.Errorf("%s was not found in PATH or standard install locations", name)
}

func isExecutableFile(info os.FileInfo) bool {
	return !info.IsDir() && info.Mode()&0111 != 0
}

func containerPlatformSupport() (bool, string, string) {
	if runtime.GOOS != "darwin" {
		return false, runtime.GOOS, "Apple Container requires macOS 26 or later."
	}
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	version := strings.TrimSpace(string(out))
	if err != nil || version == "" {
		return false, version, "Unable to detect macOS version. Apple Container requires macOS 26 or later."
	}
	major, err := macOSMajorVersion(version)
	if err != nil {
		return false, version, "Unable to parse macOS version " + version + ". Apple Container requires macOS 26 or later."
	}
	if major < 26 {
		return false, version, "Apple Container requires macOS 26 or later. Current macOS: " + version + "."
	}
	return true, version, ""
}

func macOSMajorVersion(version string) (int, error) {
	major, _, ok := strings.Cut(strings.TrimSpace(version), ".")
	if !ok {
		major = strings.TrimSpace(version)
	}
	return strconv.Atoi(major)
}

func (r *ContainerRepo) Status() model.ContainerCLIStatus {
	supported, osVersion, platformError := containerPlatformSupport()
	if !supported {
		return model.ContainerCLIStatus{
			Available: false,
			Supported: false,
			OSVersion: osVersion,
			Error:     platformError,
		}
	}

	path, err := r.executable()
	if err != nil {
		return model.ContainerCLIStatus{
			Available: false,
			Supported: true,
			OSVersion: osVersion,
			Error:     "`container` CLI was not found in PATH, /usr/local/bin, or /opt/homebrew/bin. Install Apple Container first.",
		}
	}

	version, err := r.run(5*time.Second, "--version")
	if err != nil {
		return model.ContainerCLIStatus{Available: true, Supported: true, Path: path, OSVersion: osVersion, Error: strings.TrimSpace(err.Error())}
	}
	status, err := r.run(5*time.Second, "system", "status")
	trimmedStatus := strings.TrimSpace(status)
	lowerStatus := strings.ToLower(trimmedStatus)
	if err != nil {
		if strings.Contains(lowerStatus, "not running") || strings.Contains(lowerStatus, "not registered") {
			return model.ContainerCLIStatus{
				Available: true,
				Supported: true,
				Path:      path,
				Version:   strings.TrimSpace(version),
				System:    trimmedStatus,
				OSVersion: osVersion,
				Error:     trimmedStatus + ". Run `container system start`.",
			}
		}
		return model.ContainerCLIStatus{
			Available: true,
			Supported: true,
			Path:      path,
			Version:   strings.TrimSpace(version),
			System:    trimmedStatus,
			OSVersion: osVersion,
			Error:     strings.TrimSpace(err.Error()),
		}
	}
	if strings.Contains(lowerStatus, "not running") || strings.Contains(lowerStatus, "not registered") {
		return model.ContainerCLIStatus{
			Available: true,
			Supported: true,
			Path:      path,
			Version:   strings.TrimSpace(version),
			System:    trimmedStatus,
			OSVersion: osVersion,
			Error:     trimmedStatus + ". Run `container system start`.",
		}
	}
	return model.ContainerCLIStatus{Available: true, Supported: true, Path: path, Version: strings.TrimSpace(version), System: trimmedStatus, OSVersion: osVersion}
}

func (r *ContainerRepo) ListContainers() ([]model.ContainerInfo, error) {
	out, err := r.run(20*time.Second, "list", "--all", "--format", "json")
	if err != nil {
		return nil, err
	}
	rows, err := decodeObjectList([]byte(out), "containers", "items", "data")
	if err != nil {
		return nil, err
	}

	containers := parseContainerRows(rows)

	sort.SliceStable(containers, func(i, j int) bool {
		if containers[i].Status == containers[j].Status {
			return containers[i].Name < containers[j].Name
		}
		return statusWeight(containers[i].Status) < statusWeight(containers[j].Status)
	})
	return containers, nil
}

func (r *ContainerRepo) ListImages() ([]model.ImageInfo, error) {
	out, err := r.run(20*time.Second, "image", "list", "--format", "json")
	if err != nil {
		return nil, err
	}
	rows, err := decodeObjectList([]byte(out), "images", "items", "data")
	if err != nil {
		return nil, err
	}

	return parseImageRows(rows), nil
}

func (r *ContainerRepo) ListNetworks() ([]model.NetworkInfo, error) {
	out, err := r.run(20*time.Second, "network", "list", "--format", "json")
	if err != nil {
		return nil, err
	}
	rows, err := decodeObjectList([]byte(out), "networks", "items", "data")
	if err != nil {
		return nil, err
	}
	return parseNetworkRows(rows), nil
}

func (r *ContainerRepo) CreateNetwork(opts model.CreateNetworkOptions) (model.ContainerCommandResult, error) {
	args, err := createNetworkArgs(opts)
	if err != nil {
		return model.ContainerCommandResult{}, err
	}
	out, err := r.run(2*time.Minute, args...)
	return model.ContainerCommandResult{Command: shellCommand(r.binary, args), Output: strings.TrimSpace(out)}, err
}

func (r *ContainerRepo) RemoveNetwork(name string) (model.ContainerCommandResult, error) {
	args, err := deleteNetworkArgs(name)
	if err != nil {
		return model.ContainerCommandResult{}, err
	}
	out, err := r.run(2*time.Minute, args...)
	return model.ContainerCommandResult{Command: shellCommand(r.binary, args), Output: strings.TrimSpace(out)}, err
}

func (r *ContainerRepo) PullImage(reference string) (model.ContainerCommandResult, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return model.ContainerCommandResult{}, errors.New("image reference is required")
	}
	args := []string{"image", "pull", "--progress", "plain", reference}
	out, err := r.run(10*time.Minute, args...)
	return model.ContainerCommandResult{Command: shellCommand(r.binary, args), Output: strings.TrimSpace(out)}, err
}

func (r *ContainerRepo) RemoveImage(reference string) (model.ContainerCommandResult, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return model.ContainerCommandResult{}, errors.New("image reference is required")
	}
	args := []string{"image", "delete", "--force", reference}
	out, err := r.run(2*time.Minute, args...)
	return model.ContainerCommandResult{Command: shellCommand(r.binary, args), Output: strings.TrimSpace(out)}, err
}

func (r *ContainerRepo) PruneImages(all bool) (model.ContainerCommandResult, error) {
	args := []string{"image", "prune"}
	if all {
		args = append(args, "--all")
	}
	out, err := r.run(2*time.Minute, args...)
	return model.ContainerCommandResult{Command: shellCommand(r.binary, args), Output: strings.TrimSpace(out)}, err
}

func (r *ContainerRepo) InstallContainer(terminalApp string) error {
	if brew, err := resolveExecutable("brew", []string{"/opt/homebrew/bin", "/usr/local/bin"}); err == nil && brew != "" {
		return openTerminalCommand(terminalApp, "brew install container")
	}
	return exec.Command("open", "https://github.com/apple/container/releases/latest").Run()
}

func (r *ContainerRepo) StartSystem(terminalApp string) error {
	binary, err := r.executable()
	if err != nil {
		binary = r.binary
	}
	return openTerminalCommand(terminalApp, fmt.Sprintf("%s system start", shellQuote(binary)))
}

func (r *ContainerRepo) RunContainer(opts model.RunContainerOptions) (model.ContainerCommandResult, error) {
	if strings.TrimSpace(opts.Image) == "" {
		return model.ContainerCommandResult{}, errors.New("image is required")
	}
	opts = normalizeRunOptions(opts)

	args := []string{"run"}
	if opts.Detach {
		args = append(args, "--detach")
	}
	if opts.Remove && !opts.Detach {
		args = append(args, "--rm")
	}
	if strings.TrimSpace(opts.Name) != "" {
		args = append(args, "--name", strings.TrimSpace(opts.Name))
	}
	if strings.TrimSpace(opts.Network) != "" {
		args = append(args, "--network", strings.TrimSpace(opts.Network))
	}
	for _, dns := range compactStrings(opts.DNS) {
		args = append(args, "--dns", dns)
	}
	if opts.NoDNS {
		args = append(args, "--no-dns")
	}
	if strings.TrimSpace(opts.Platform) != "" {
		args = append(args, "--platform", strings.TrimSpace(opts.Platform))
	}
	if strings.TrimSpace(opts.CPUs) != "" {
		args = append(args, "--cpus", strings.TrimSpace(opts.CPUs))
	}
	if strings.TrimSpace(opts.Memory) != "" {
		args = append(args, "--memory", strings.TrimSpace(opts.Memory))
	}
	if strings.TrimSpace(opts.Workdir) != "" {
		args = append(args, "--workdir", strings.TrimSpace(opts.Workdir))
	}
	if strings.TrimSpace(opts.User) != "" {
		args = append(args, "--user", strings.TrimSpace(opts.User))
	}
	if strings.TrimSpace(opts.Entrypoint) != "" {
		args = append(args, "--entrypoint", strings.TrimSpace(opts.Entrypoint))
	}
	for _, label := range compactStrings(opts.Labels) {
		args = append(args, "--label", label)
	}
	if opts.Init {
		args = append(args, "--init")
	}
	if opts.ReadOnly {
		args = append(args, "--read-only")
	}
	if opts.SSH {
		args = append(args, "--ssh")
	}
	for _, port := range compactStrings(opts.Ports) {
		args = append(args, "--publish", port)
	}
	for _, env := range compactStrings(opts.Envs) {
		args = append(args, "--env", env)
	}
	for _, volume := range compactStrings(opts.Volumes) {
		args = append(args, "--volume", volume)
	}
	args = append(args, strings.TrimSpace(opts.Image))
	if cmd := strings.TrimSpace(opts.Command); cmd != "" {
		args = append(args, "/bin/sh", "-lc", cmd)
	}

	out, err := r.run(10*time.Minute, args...)
	return model.ContainerCommandResult{Command: shellCommand(r.binary, args), Output: strings.TrimSpace(out)}, err
}

func (r *ContainerRepo) StartContainer(id string) (model.ContainerCommandResult, error) {
	return r.containerCommand(2*time.Minute, "start", id)
}

func (r *ContainerRepo) StopContainer(id string) (model.ContainerCommandResult, error) {
	return r.containerCommand(2*time.Minute, "stop", id)
}

func (r *ContainerRepo) RestartContainer(id string) (model.ContainerCommandResult, error) {
	if _, err := r.StopContainer(id); err != nil {
		return model.ContainerCommandResult{}, err
	}
	return r.StartContainer(id)
}

func (r *ContainerRepo) RemoveContainer(id string) (model.ContainerCommandResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.ContainerCommandResult{}, errors.New("container id is required")
	}
	args := []string{"delete", "--force", id}
	out, err := r.run(2*time.Minute, args...)
	return model.ContainerCommandResult{Command: shellCommand(r.binary, args), Output: strings.TrimSpace(out)}, err
}

func (r *ContainerRepo) Logs(id string, tail int) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("container id is required")
	}
	args := []string{"logs"}
	if tail > 0 {
		args = append(args, "-n", strconv.Itoa(tail))
	}
	args = append(args, id)
	return r.run(20*time.Second, args...)
}

func (r *ContainerRepo) Inspect(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", errors.New("container id is required")
	}
	return r.run(20*time.Second, "inspect", id)
}

func (r *ContainerRepo) OpenShell(id string, terminalApp string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("container id is required")
	}
	binary, err := r.executable()
	if err != nil {
		binary = r.binary
	}
	return openTerminalCommand(terminalApp, fmt.Sprintf("%s exec -it %s /bin/sh", shellQuote(binary), shellQuote(id)))
}

func (r *ContainerRepo) containerCommand(timeout time.Duration, action string, id string) (model.ContainerCommandResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.ContainerCommandResult{}, errors.New("container id is required")
	}
	args := []string{action, id}
	out, err := r.run(timeout, args...)
	return model.ContainerCommandResult{Command: shellCommand(r.binary, args), Output: strings.TrimSpace(out)}, err
}

func (r *ContainerRepo) run(timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	binary, err := r.executable()
	if err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err = cmd.Run()
	text := out.String()
	if ctx.Err() == context.DeadlineExceeded {
		return text, fmt.Errorf("container command timed out: %s", shellCommand(r.binary, args))
	}
	if err != nil {
		return text, fmt.Errorf("%s\n%s", err, strings.TrimSpace(text))
	}
	return text, nil
}

func decodeObjectList(data []byte, collectionKeys ...string) ([]map[string]any, error) {
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("failed to parse container JSON output: %w", err)
	}
	return objectList(decoded, collectionKeys...), nil
}

func parseImageRows(rows []map[string]any) []model.ImageInfo {
	images := make([]model.ImageInfo, 0, len(rows))
	for _, row := range rows {
		name := firstString(row, "name", "reference", "image", "repository")
		if name == "" {
			name = nestedString(row, "configuration", "name")
		}
		tag := firstString(row, "tag")
		if tag == "" {
			name, tag = splitImageTag(name)
		}
		if name == "" && tag != "" {
			name = tag
			tag = ""
		}
		if name == "" {
			continue
		}
		images = append(images, model.ImageInfo{
			ID:        firstString(row, "id", "imageID", "imageId"),
			Name:      name,
			Tag:       tag,
			Digest:    firstNonEmpty(firstString(row, "digest"), nestedString(row, "configuration", "descriptor", "digest")),
			Size:      imageSize(row),
			CreatedAt: firstNonEmpty(firstString(row, "createdAt", "created"), nestedString(row, "configuration", "creationDate")),
		})
	}
	return images
}

func parseContainerRows(rows []map[string]any) []model.ContainerInfo {
	containers := make([]model.ContainerInfo, 0, len(rows))
	for _, row := range rows {
		id := firstNonEmpty(firstString(row, "id", "containerID", "containerId"), nestedString(row, "configuration", "id"))
		name := firstNonEmpty(firstString(row, "name", "names"), nestedString(row, "configuration", "name"), id)
		image := firstNonEmpty(
			firstString(row, "image", "imageName", "imageReference", "reference"),
			nestedString(row, "configuration", "image", "reference"),
		)
		status := firstNonEmpty(firstString(row, "status", "state", "runtimeStatus"), nestedString(row, "status", "state"))
		created := firstNonEmpty(
			firstString(row, "createdAt", "created", "createdTime", "startedDate"),
			nestedString(row, "status", "startedDate"),
			nestedString(row, "configuration", "creationDate"),
		)

		if id == "" && name == "" && image == "" {
			continue
		}
		if id == "" {
			id = name
		}
		containers = append(containers, model.ContainerInfo{
			ID:        id,
			Name:      name,
			Image:     image,
			Status:    normalizeStatus(status),
			IP:        containerIP(row),
			CPUs:      nestedString(row, "configuration", "resources", "cpus"),
			Memory:    containerMemory(row),
			OS:        nestedString(row, "configuration", "platform", "os"),
			Arch:      nestedString(row, "configuration", "platform", "architecture"),
			Ports:     containerPorts(row),
			CreatedAt: created,
			Age:       firstString(row, "age"),
		})
	}
	return containers
}

func parseNetworkRows(rows []map[string]any) []model.NetworkInfo {
	networks := make([]model.NetworkInfo, 0, len(rows))
	for _, row := range rows {
		id := firstNonEmpty(firstString(row, "id", "networkID", "networkId"), nestedString(row, "configuration", "id"), nestedString(row, "configuration", "name"))
		name := firstNonEmpty(firstString(row, "name"), nestedString(row, "configuration", "name"), id)
		if id == "" && name == "" {
			continue
		}
		if id == "" {
			id = name
		}
		if name == "" {
			name = id
		}
		networks = append(networks, model.NetworkInfo{
			ID:          id,
			Name:        name,
			Mode:        firstNonEmpty(firstString(row, "mode"), nestedString(row, "configuration", "mode")),
			Plugin:      firstNonEmpty(firstString(row, "plugin"), nestedString(row, "configuration", "plugin")),
			IPv4Subnet:  firstNonEmpty(firstString(row, "ipv4Subnet"), nestedString(row, "status", "ipv4Subnet")),
			IPv4Gateway: firstNonEmpty(firstString(row, "ipv4Gateway"), nestedString(row, "status", "ipv4Gateway")),
			IPv6Subnet:  firstNonEmpty(firstString(row, "ipv6Subnet"), nestedString(row, "status", "ipv6Subnet")),
			Labels:      keyValueStrings(nestedAny(row, "configuration", "labels")),
			Options:     keyValueStrings(nestedAny(row, "configuration", "options")),
			CreatedAt:   firstNonEmpty(firstString(row, "createdAt", "created"), nestedString(row, "configuration", "creationDate")),
		})
	}
	sort.SliceStable(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})
	return networks
}

func createNetworkArgs(opts model.CreateNetworkOptions) ([]string, error) {
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return nil, errors.New("network name is required")
	}

	args := []string{"network", "create"}
	if opts.Internal {
		args = append(args, "--internal")
	}
	for _, label := range compactStrings(opts.Labels) {
		args = append(args, "--label", label)
	}
	for _, option := range compactStrings(opts.Options) {
		args = append(args, "--option", option)
	}
	if strings.TrimSpace(opts.Plugin) != "" {
		args = append(args, "--plugin", strings.TrimSpace(opts.Plugin))
	}
	if strings.TrimSpace(opts.Subnet) != "" {
		args = append(args, "--subnet", strings.TrimSpace(opts.Subnet))
	}
	if strings.TrimSpace(opts.SubnetV6) != "" {
		args = append(args, "--subnet-v6", strings.TrimSpace(opts.SubnetV6))
	}
	args = append(args, name)
	return args, nil
}

func deleteNetworkArgs(name string) ([]string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("network name is required")
	}
	return []string{"network", "delete", name}, nil
}

func objectList(value any, collectionKeys ...string) []map[string]any {
	switch typed := value.(type) {
	case []any:
		result := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			if obj, ok := item.(map[string]any); ok {
				result = append(result, obj)
			}
		}
		return result
	case map[string]any:
		for _, key := range collectionKeys {
			if nested, ok := typed[key]; ok {
				if rows := objectList(nested, collectionKeys...); len(rows) > 0 {
					return rows
				}
			}
		}
		return []map[string]any{typed}
	default:
		return nil
	}
}

func firstString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := lookup(row, key)
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(typed)
		case []any:
			parts := make([]string, 0, len(typed))
			for _, item := range typed {
				if s := fmt.Sprint(item); strings.TrimSpace(s) != "" {
					parts = append(parts, strings.TrimSpace(s))
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, ", ")
			}
		}
	}
	return ""
}

func firstStringSlice(row map[string]any, keys ...string) []string {
	for _, key := range keys {
		value, ok := lookup(row, key)
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case []any:
			parts := make([]string, 0, len(typed))
			for _, item := range typed {
				if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
					parts = append(parts, s)
				}
			}
			return parts
		case []string:
			return compactStrings(typed)
		case string:
			if typed == "" {
				return nil
			}
			if strings.Contains(typed, ",") {
				return compactStrings(strings.Split(typed, ","))
			}
			return []string{typed}
		}
	}
	return nil
}

func lookup(row map[string]any, key string) (any, bool) {
	for k, v := range row {
		if strings.EqualFold(k, key) {
			return v, true
		}
	}
	return nil, false
}

func nestedString(row map[string]any, path ...string) string {
	var current any = row
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		value, ok := lookup(obj, key)
		if !ok {
			return ""
		}
		current = value
	}
	switch typed := current.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return ""
	}
}

func splitImageTag(reference string) (string, string) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", ""
	}
	lastSlash := strings.LastIndex(reference, "/")
	lastColon := strings.LastIndex(reference, ":")
	if lastColon > lastSlash {
		return reference[:lastColon], reference[lastColon+1:]
	}
	return reference, ""
}

func imageSize(row map[string]any) string {
	if size := firstString(row, "size", "fullSize"); size != "" {
		return size
	}
	var total float64
	if variants, ok := row["variants"].([]any); ok {
		for _, variant := range variants {
			obj, ok := variant.(map[string]any)
			if !ok {
				continue
			}
			value, ok := lookup(obj, "size")
			if !ok {
				continue
			}
			switch typed := value.(type) {
			case float64:
				total += typed
			case int64:
				total += float64(typed)
			case int:
				total += float64(typed)
			}
		}
	}
	if total <= 0 {
		return nestedString(row, "configuration", "descriptor", "size")
	}
	return formatBytes(total)
}

func containerIP(row map[string]any) string {
	networks, ok := nestedAny(row, "status", "networks").([]any)
	if !ok {
		return firstString(row, "ip", "ipAddress")
	}
	for _, network := range networks {
		obj, ok := network.(map[string]any)
		if !ok {
			continue
		}
		if ip := firstString(obj, "ipv4Address", "ip", "ipAddress"); ip != "" {
			return ip
		}
	}
	return ""
}

func containerMemory(row map[string]any) string {
	value := nestedAny(row, "configuration", "resources", "memoryInBytes")
	switch typed := value.(type) {
	case float64:
		return formatBytes(typed)
	case int64:
		return formatBytes(float64(typed))
	case int:
		return formatBytes(float64(typed))
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return formatBytes(parsed)
		}
		return strings.TrimSpace(typed)
	default:
		return firstString(row, "memory", "mem")
	}
}

func containerPorts(row map[string]any) []string {
	if ports := firstStringSlice(row, "ports", "portMappings", "publish"); len(ports) > 0 {
		return ports
	}
	published, ok := nestedAny(row, "configuration", "publishedPorts").([]any)
	if !ok {
		return nil
	}
	ports := make([]string, 0, len(published))
	for _, item := range published {
		obj, ok := item.(map[string]any)
		if !ok {
			if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
				ports = append(ports, text)
			}
			continue
		}
		host := firstString(obj, "hostPort", "host")
		container := firstString(obj, "containerPort", "port")
		protocol := firstString(obj, "protocol")
		if host == "" && container == "" {
			continue
		}
		mapping := container
		if host != "" {
			mapping = host + ":" + container
		}
		if protocol != "" {
			mapping += "/" + protocol
		}
		ports = append(ports, mapping)
	}
	return ports
}

func normalizeRunOptions(opts model.RunContainerOptions) model.RunContainerOptions {
	if opts.Detach {
		opts.Remove = false
	}
	opts.CPUs = normalizeRunResource(opts.CPUs)
	opts.Memory = normalizeRunResource(opts.Memory)
	if strings.TrimSpace(opts.Command) == "" && opts.Detach {
		opts.Command = defaultDetachedCommand(opts.Image)
	}
	return opts
}

func normalizeRunResource(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || isZeroResourceValue(trimmed) {
		return ""
	}
	return trimmed
}

func isZeroResourceValue(value string) bool {
	cleaned := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
	for _, suffix := range []string{"bytes", "byte", "mib", "gib", "tib", "pib", "kib", "mb", "gb", "tb", "pb", "kb", "mi", "gi", "ti", "pi", "ki", "b", "m", "g", "t", "p", "k"} {
		cleaned = strings.TrimSuffix(cleaned, suffix)
	}
	number, err := strconv.ParseFloat(cleaned, 64)
	return err == nil && number == 0
}

func defaultDetachedCommand(image string) string {
	name := normalizedImageName(image)
	switch name {
	case "alpine", "ubuntu", "debian", "busybox", "fedora", "centos", "rockylinux", "archlinux", "oraclelinux":
		return "sleep infinity"
	default:
		return ""
	}
}

func normalizedImageName(image string) string {
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}
	if at := strings.Index(image, "@"); at >= 0 {
		image = image[:at]
	}
	lastSlash := strings.LastIndex(image, "/")
	lastColon := strings.LastIndex(image, ":")
	if lastColon > lastSlash {
		image = image[:lastColon]
	}
	if lastSlash >= 0 {
		image = image[lastSlash+1:]
	}
	return strings.ToLower(image)
}

func nestedAny(row map[string]any, path ...string) any {
	var current any = row
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		value, ok := lookup(obj, key)
		if !ok {
			return nil
		}
		current = value
	}
	return current
}

func keyValueStrings(value any) []string {
	obj, ok := value.(map[string]any)
	if !ok || len(obj) == 0 {
		return nil
	}
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		text := strings.TrimSpace(fmt.Sprint(obj[key]))
		if text == "" {
			continue
		}
		out = append(out, key+"="+text)
	}
	return out
}

func formatBytes(size float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	for _, unit := range units {
		if size < 1024 || unit == units[len(units)-1] {
			if unit == "B" {
				return fmt.Sprintf("%.0f %s", size, unit)
			}
			return fmt.Sprintf("%.1f %s", size, unit)
		}
		size /= 1024
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch {
	case status == "":
		return "unknown"
	case strings.Contains(status, "running"):
		return "running"
	case strings.Contains(status, "stopped"), strings.Contains(status, "exited"):
		return "stopped"
	case strings.Contains(status, "created"):
		return "created"
	default:
		return status
	}
}

func statusWeight(status string) int {
	switch normalizeStatus(status) {
	case "running":
		return 0
	case "created":
		return 1
	case "stopped":
		return 2
	default:
		return 3
	}
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func shellCommand(binary string, args []string) string {
	parts := append([]string{binary}, args...)
	for i, part := range parts {
		parts[i] = shellQuote(part)
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if strings.ContainsAny(value, " \t\n'\"\\$`!*?[]{}()&;|<>") {
		return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
	}
	return value
}

func openTerminalCommand(terminalApp string, command string) error {
	app := strings.TrimSpace(terminalApp)
	if app == "" {
		app = "Terminal"
	}

	switch strings.ToLower(app) {
	case "terminal", "terminal.app":
		script := fmt.Sprintf(`tell application "Terminal"
	activate
	do script %q
end tell`, command)
		return exec.Command("osascript", "-e", script).Run()
	case "iterm", "iterm2", "iterm.app", "iterm2.app":
		script := fmt.Sprintf(`tell application "iTerm"
	activate
	create window with default profile command %q
end tell`, command)
		return exec.Command("osascript", "-e", script).Run()
	case "alacritty":
		if err := exec.Command("open", "-na", "Alacritty", "--args", "-e", "/bin/zsh", "-lc", command).Run(); err == nil {
			return nil
		}
		return exec.Command("/bin/zsh", "-lc", fmt.Sprintf("alacritty -e /bin/zsh -lc %s >/dev/null 2>&1 &", shellQuote(command))).Run()
	case "wezterm":
		return exec.Command("open", "-na", "WezTerm", "--args", "start", "--", "/bin/zsh", "-lc", command).Run()
	case "ghostty":
		return exec.Command("open", "-na", "Ghostty", "--args", "-e", "/bin/zsh", "-lc", command).Run()
	case "warp":
		if err := copyText(command); err != nil {
			return err
		}
		return exec.Command("open", "-a", "Warp").Run()
	default:
		return exec.Command("open", "-a", app).Run()
	}
}

func copyText(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
