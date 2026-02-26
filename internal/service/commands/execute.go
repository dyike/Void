package commands

import (
	"os/exec"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// LaunchApp opens an application by its path.
func (s *Service) LaunchApp(appPath string) error {
	cmd := exec.Command("open", appPath)
	return cmd.Run()
}

// OpenURL opens a URL in the default browser.
func (s *Service) OpenURL(url string) error {
	return exec.Command("open", url).Run()
}

// HideWindow hides the main application window.
func (s *Service) HideWindow() {
	if app := application.Get(); app != nil {
		app.Hide()
	}
}

// OpenSystemSettings opens a macOS Settings pane by bundle identifier.
func (s *Service) OpenSystemSettings(bundleID string) error {
	cmd := exec.Command("open", "x-apple.systempreferences:"+bundleID)
	if err := cmd.Run(); err != nil {
		return exec.Command("open", "-b", "com.apple.systempreferences").Run()
	}
	return nil
}

// RevealInFinder reveals a file in Finder (selects it).
func (s *Service) RevealInFinder(path string) error {
	return exec.Command("open", "-R", path).Run()
}

// QuickLook opens Quick Look preview for a file.
func (s *Service) QuickLook(path string) error {
	return exec.Command("qlmanage", "-p", path).Start()
}

// CopyToClipboard copies text to the macOS clipboard.
func (s *Service) CopyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// ExecuteSystemCommand runs a built-in macOS system command by ID.
func (s *Service) ExecuteSystemCommand(id string) error {
	switch id {
	case "system-lock-screen":
		return exec.Command("pmset", "displaysleepnow").Run()
	case "system-sleep":
		return exec.Command("pmset", "sleepnow").Run()
	case "system-empty-trash":
		script := `tell application "Finder" to empty the trash`
		return exec.Command("osascript", "-e", script).Run()
	case "system-restart":
		script := `tell application "System Events" to restart`
		return exec.Command("osascript", "-e", script).Run()
	case "system-shutdown":
		script := `tell application "System Events" to shut down`
		return exec.Command("osascript", "-e", script).Run()
	case "system-open-finder":
		return exec.Command("open", "-a", "Finder").Run()
	case "system-open-terminal":
		appName := "Terminal"
		if s.settings != nil {
			if cfg, err := s.settings.GetSettings(); err == nil && cfg != nil && cfg.TerminalApp != "" {
				appName = cfg.TerminalApp
			}
		}
		return exec.Command("open", "-a", appName).Run()
	case "system-show-desktop":
		script := `tell application "System Events" to key code 103 using {command down, mission control key down}`
		return exec.Command("osascript", "-e", script).Run()
	// ── Window management ───────────────────────────────────
	case "window-left-half":
		return moveWindow(0, 0, 0.5, 1)
	case "window-right-half":
		return moveWindow(0.5, 0, 0.5, 1)
	case "window-top-half":
		return moveWindow(0, 0, 1, 0.5)
	case "window-bottom-half":
		return moveWindow(0, 0.5, 1, 0.5)
	case "window-top-left":
		return moveWindow(0, 0, 0.5, 0.5)
	case "window-top-right":
		return moveWindow(0.5, 0, 0.5, 0.5)
	case "window-bottom-left":
		return moveWindow(0, 0.5, 0.5, 0.5)
	case "window-bottom-right":
		return moveWindow(0.5, 0.5, 0.5, 0.5)
	case "window-maximize":
		return moveWindow(0, 0, 1, 1)
	case "window-center":
		return centerWindow()
	}
	return nil
}
