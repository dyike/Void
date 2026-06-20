package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
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

func (r *ContainerRepo) Status() model.ContainerCLIStatus {
	path, err := exec.LookPath(r.binary)
	if err != nil {
		return model.ContainerCLIStatus{Available: false, Error: "`container` CLI was not found in PATH"}
	}

	version, err := r.run(5*time.Second, "--version")
	if err != nil {
		return model.ContainerCLIStatus{Available: true, Path: path, Error: strings.TrimSpace(err.Error())}
	}
	status, err := r.run(5*time.Second, "system", "status")
	trimmedStatus := strings.TrimSpace(status)
	lowerStatus := strings.ToLower(trimmedStatus)
	if err != nil {
		if strings.Contains(lowerStatus, "not running") || strings.Contains(lowerStatus, "not registered") {
			return model.ContainerCLIStatus{
				Available: true,
				Path:      path,
				Version:   strings.TrimSpace(version),
				System:    trimmedStatus,
				Error:     trimmedStatus + ". Run `container system start`.",
			}
		}
		return model.ContainerCLIStatus{
			Available: true,
			Path:      path,
			Version:   strings.TrimSpace(version),
			System:    trimmedStatus,
			Error:     strings.TrimSpace(err.Error()),
		}
	}
	if strings.Contains(lowerStatus, "not running") || strings.Contains(lowerStatus, "not registered") {
		return model.ContainerCLIStatus{
			Available: true,
			Path:      path,
			Version:   strings.TrimSpace(version),
			System:    trimmedStatus,
			Error:     trimmedStatus + ". Run `container system start`.",
		}
	}
	return model.ContainerCLIStatus{Available: true, Path: path, Version: strings.TrimSpace(version), System: trimmedStatus}
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

	containers := make([]model.ContainerInfo, 0, len(rows))
	for _, row := range rows {
		id := firstString(row, "id", "containerID", "containerId", "name")
		name := firstString(row, "name", "names", "id")
		image := firstString(row, "image", "imageName", "imageReference", "reference")
		status := firstString(row, "status", "state", "runtimeStatus")
		created := firstString(row, "createdAt", "created", "createdTime")
		ports := firstStringSlice(row, "ports", "publishedPorts", "portMappings", "publish")

		if id == "" && name == "" && image == "" {
			continue
		}
		if name == "" {
			name = id
		}
		if id == "" {
			id = name
		}
		containers = append(containers, model.ContainerInfo{
			ID:        id,
			Name:      name,
			Image:     image,
			Status:    normalizeStatus(status),
			Ports:     ports,
			CreatedAt: created,
			Age:       firstString(row, "age"),
		})
	}

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

	images := make([]model.ImageInfo, 0, len(rows))
	for _, row := range rows {
		name := firstString(row, "name", "reference", "image", "repository")
		tag := firstString(row, "tag")
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
			Digest:    firstString(row, "digest"),
			Size:      firstString(row, "size", "fullSize"),
			CreatedAt: firstString(row, "createdAt", "created"),
		})
	}
	return images, nil
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
	args := []string{"image", "delete", reference}
	out, err := r.run(2*time.Minute, args...)
	return model.ContainerCommandResult{Command: shellCommand(r.binary, args), Output: strings.TrimSpace(out)}, err
}

func (r *ContainerRepo) InstallContainer(terminalApp string) error {
	if brew, err := exec.LookPath("brew"); err == nil && brew != "" {
		return openTerminalCommand(terminalApp, "brew install container")
	}
	return exec.Command("open", "https://github.com/apple/container/releases/latest").Run()
}

func (r *ContainerRepo) StartSystem(terminalApp string) error {
	return openTerminalCommand(terminalApp, "container system start")
}

func (r *ContainerRepo) RunContainer(opts model.RunContainerOptions) (model.ContainerCommandResult, error) {
	if strings.TrimSpace(opts.Image) == "" {
		return model.ContainerCommandResult{}, errors.New("image is required")
	}

	args := []string{"run"}
	if opts.Detach {
		args = append(args, "--detach")
	}
	if opts.Remove {
		args = append(args, "--rm")
	}
	if strings.TrimSpace(opts.Name) != "" {
		args = append(args, "--name", strings.TrimSpace(opts.Name))
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
		args = append(args, strings.Fields(cmd)...)
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
	return r.containerCommand(2*time.Minute, "delete", id)
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
	return openTerminalCommand(terminalApp, fmt.Sprintf("container exec -it %s /bin/sh", shellQuote(id)))
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

	cmd := exec.CommandContext(ctx, r.binary, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
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
