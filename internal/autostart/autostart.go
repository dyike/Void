package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.dyike.void</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.ExecutablePath}}</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>ProcessType</key>
	<string>Background</string>
</dict>
</plist>`

func getPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.dyike.void.plist"), nil
}

func getExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	// Note: Wails builds a .app bundle on macOS. os.Executable() returns the path to the internal binary
	// e.g. /Applications/Void.app/Contents/MacOS/Void
	return execPath, nil
}

// Enable creates a LaunchAgent to start the app at login
func Enable() error {
	plistPath, err := getPlistPath()
	if err != nil {
		return fmt.Errorf("failed to get plist path: %w", err)
	}

	execPath, err := getExecutablePath()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Ensure LaunchAgents directory exists
	if err := os.MkdirAll(filepath.Dir(plistPath), 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents dir: %w", err)
	}

	f, err := os.Create(plistPath)
	if err != nil {
		return fmt.Errorf("failed to create plist file: %w", err)
	}
	defer f.Close()

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse plist template: %w", err)
	}

	data := struct{ ExecutablePath string }{ExecutablePath: execPath}
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	return nil
}

// Disable removes the LaunchAgent
func Disable() error {
	plistPath, err := getPlistPath()
	if err != nil {
		return err
	}

	// Ignore not-exist errors
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return nil
	}

	return os.Remove(plistPath)
}

// IsEnabled checks if the LaunchAgent exists
func IsEnabled() bool {
	plistPath, err := getPlistPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(plistPath)
	return err == nil
}
