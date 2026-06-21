package service

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/repo"
)

// ContainerService exposes Apple Container workflows to the Wails frontend.
type ContainerService struct {
	repo     *repo.ContainerRepo
	settings *SettingsService
}

func NewContainerService(containerRepo *repo.ContainerRepo, settingsSvc *SettingsService) *ContainerService {
	return &ContainerService{repo: containerRepo, settings: settingsSvc}
}

func (s *ContainerService) GetStatus() model.ContainerCLIStatus {
	return s.repo.Status()
}

func (s *ContainerService) GetHostResources() model.HostResourceInfo {
	memoryBytes := hostMemoryBytes()
	return model.HostResourceInfo{
		CPUCores:    runtime.NumCPU(),
		MemoryBytes: memoryBytes,
		MemoryMiB:   int(memoryBytes / 1024 / 1024),
	}
}

func (s *ContainerService) ListContainers() ([]model.ContainerInfo, error) {
	if !s.isReady() {
		return []model.ContainerInfo{}, nil
	}
	return s.repo.ListContainers()
}

func (s *ContainerService) ListImages() ([]model.ImageInfo, error) {
	if !s.isReady() {
		return []model.ImageInfo{}, nil
	}
	return s.repo.ListImages()
}

func (s *ContainerService) ListNetworks() ([]model.NetworkInfo, error) {
	if !s.isReady() {
		return []model.NetworkInfo{}, nil
	}
	return s.repo.ListNetworks()
}

func (s *ContainerService) CreateNetwork(opts model.CreateNetworkOptions) (model.ContainerCommandResult, error) {
	return s.repo.CreateNetwork(opts)
}

func (s *ContainerService) RemoveNetwork(name string) (model.ContainerCommandResult, error) {
	return s.repo.RemoveNetwork(name)
}

func (s *ContainerService) PullImage(reference string) (model.ContainerCommandResult, error) {
	return s.repo.PullImage(reference)
}

func (s *ContainerService) RemoveImage(reference string) (model.ContainerCommandResult, error) {
	return s.repo.RemoveImage(reference)
}

func (s *ContainerService) PruneImages(all bool) (model.ContainerCommandResult, error) {
	return s.repo.PruneImages(all)
}

func (s *ContainerService) InstallContainer() error {
	return s.repo.InstallContainer(s.terminalApp())
}

func (s *ContainerService) StartSystem() error {
	return s.repo.StartSystem(s.terminalApp())
}

func (s *ContainerService) RunContainer(opts model.RunContainerOptions) (model.ContainerCommandResult, error) {
	return s.repo.RunContainer(opts)
}

func (s *ContainerService) StartContainer(id string) (model.ContainerCommandResult, error) {
	return s.repo.StartContainer(id)
}

func (s *ContainerService) StopContainer(id string) (model.ContainerCommandResult, error) {
	return s.repo.StopContainer(id)
}

func (s *ContainerService) RestartContainer(id string) (model.ContainerCommandResult, error) {
	return s.repo.RestartContainer(id)
}

func (s *ContainerService) RemoveContainer(id string) (model.ContainerCommandResult, error) {
	return s.repo.RemoveContainer(id)
}

func (s *ContainerService) GetLogs(id string, tail int) (string, error) {
	if tail <= 0 {
		tail = 200
	}
	return s.repo.Logs(id, tail)
}

func (s *ContainerService) InspectContainer(id string) (string, error) {
	return s.repo.Inspect(id)
}

func (s *ContainerService) OpenShell(id string) error {
	return s.repo.OpenShell(id, s.terminalApp())
}

func (s *ContainerService) BuildRunCommand(opts model.RunContainerOptions) string {
	opts = normalizeRunOptionsLocal(opts)
	args := []string{"container", "run"}
	if opts.Detach {
		args = append(args, "--detach")
	}
	if opts.Remove && !opts.Detach {
		args = append(args, "--rm")
	}
	if strings.TrimSpace(opts.Name) != "" {
		args = append(args, "--name", shellQuoteLocal(opts.Name))
	}
	if strings.TrimSpace(opts.Network) != "" {
		args = append(args, "--network", shellQuoteLocal(opts.Network))
	}
	for _, dns := range opts.DNS {
		if strings.TrimSpace(dns) != "" {
			args = append(args, "--dns", shellQuoteLocal(dns))
		}
	}
	if opts.NoDNS {
		args = append(args, "--no-dns")
	}
	if strings.TrimSpace(opts.Platform) != "" {
		args = append(args, "--platform", shellQuoteLocal(opts.Platform))
	}
	if strings.TrimSpace(opts.CPUs) != "" {
		args = append(args, "--cpus", shellQuoteLocal(opts.CPUs))
	}
	if strings.TrimSpace(opts.Memory) != "" {
		args = append(args, "--memory", shellQuoteLocal(opts.Memory))
	}
	if strings.TrimSpace(opts.Workdir) != "" {
		args = append(args, "--workdir", shellQuoteLocal(opts.Workdir))
	}
	if strings.TrimSpace(opts.User) != "" {
		args = append(args, "--user", shellQuoteLocal(opts.User))
	}
	if strings.TrimSpace(opts.Entrypoint) != "" {
		args = append(args, "--entrypoint", shellQuoteLocal(opts.Entrypoint))
	}
	for _, label := range opts.Labels {
		if strings.TrimSpace(label) != "" {
			args = append(args, "--label", shellQuoteLocal(label))
		}
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
	for _, port := range opts.Ports {
		if strings.TrimSpace(port) != "" {
			args = append(args, "--publish", shellQuoteLocal(port))
		}
	}
	for _, env := range opts.Envs {
		if strings.TrimSpace(env) != "" {
			args = append(args, "--env", shellQuoteLocal(env))
		}
	}
	for _, volume := range opts.Volumes {
		if strings.TrimSpace(volume) != "" {
			args = append(args, "--volume", shellQuoteLocal(volume))
		}
	}
	if strings.TrimSpace(opts.Image) != "" {
		args = append(args, shellQuoteLocal(opts.Image))
	}
	if strings.TrimSpace(opts.Command) != "" {
		args = append(args, "/bin/sh", "-lc", shellQuoteLocal(opts.Command))
	}
	return strings.Join(args, " ")
}

func normalizeRunOptionsLocal(opts model.RunContainerOptions) model.RunContainerOptions {
	if opts.Detach {
		opts.Remove = false
	}
	opts.CPUs = normalizeRunResourceLocal(opts.CPUs)
	opts.Memory = normalizeRunResourceLocal(opts.Memory)
	if strings.TrimSpace(opts.Command) == "" && opts.Detach {
		opts.Command = defaultDetachedCommandLocal(opts.Image)
	}
	return opts
}

func normalizeRunResourceLocal(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || isZeroResourceValueLocal(trimmed) {
		return ""
	}
	return trimmed
}

func isZeroResourceValueLocal(value string) bool {
	cleaned := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
	for _, suffix := range []string{"bytes", "byte", "mib", "gib", "tib", "pib", "kib", "mb", "gb", "tb", "pb", "kb", "mi", "gi", "ti", "pi", "ki", "b", "m", "g", "t", "p", "k"} {
		cleaned = strings.TrimSuffix(cleaned, suffix)
	}
	number, err := strconv.ParseFloat(cleaned, 64)
	return err == nil && number == 0
}

func hostMemoryBytes() uint64 {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	value, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func defaultDetachedCommandLocal(image string) string {
	name := normalizedImageNameLocal(image)
	switch name {
	case "alpine", "ubuntu", "debian", "busybox", "fedora", "centos", "rockylinux", "archlinux", "oraclelinux":
		return "sleep infinity"
	default:
		return ""
	}
}

func normalizedImageNameLocal(image string) string {
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

func shellQuoteLocal(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "''"
	}
	if strings.ContainsAny(value, " \t\n'\"\\$`!*?[]{}()&;|<>") {
		return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
	}
	return value
}

func (s *ContainerService) isReady() bool {
	status := s.repo.Status()
	return status.Supported && status.Available && status.Error == ""
}

func (s *ContainerService) terminalApp() string {
	if s.settings == nil {
		return "Terminal"
	}
	settings, err := s.settings.GetSettings()
	if err != nil || settings == nil || strings.TrimSpace(settings.TerminalApp) == "" {
		return "Terminal"
	}
	return settings.TerminalApp
}
