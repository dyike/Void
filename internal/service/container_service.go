package service

import (
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

func (s *ContainerService) PullImage(reference string) (model.ContainerCommandResult, error) {
	return s.repo.PullImage(reference)
}

func (s *ContainerService) RemoveImage(reference string) (model.ContainerCommandResult, error) {
	return s.repo.RemoveImage(reference)
}

func (s *ContainerService) InstallContainer() error {
	return s.repo.InstallContainer(s.terminalApp())
}

func (s *ContainerService) StartSystem() error {
	return s.repo.StartSystem(s.terminalApp())
}

func (s *ContainerService) RunContainer(opts model.RunContainerOptions) (model.ContainerCommandResult, error) {
	if !opts.Detach {
		opts.Detach = true
	}
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
	args := []string{"container", "run"}
	if opts.Detach {
		args = append(args, "--detach")
	}
	if opts.Remove {
		args = append(args, "--rm")
	}
	if strings.TrimSpace(opts.Name) != "" {
		args = append(args, "--name", shellQuoteLocal(opts.Name))
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
		args = append(args, opts.Command)
	}
	return strings.Join(args, " ")
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
	return status.Available && status.Error == ""
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
