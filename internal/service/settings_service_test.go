package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dyike/Void/internal/repo"
)

func TestFailedSettingsSavePreservesCacheAndSkipsSystemChanges(t *testing.T) {
	dir := t.TempDir()
	// A directory at the file path makes persistence fail even when tests run as root.
	if err := os.Mkdir(filepath.Join(dir, "settings.json"), 0755); err != nil {
		t.Fatal(err)
	}
	svc := NewSettingsService(repo.NewSettingsRepo(dir))
	current, err := svc.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	original := *current
	called := false
	svc.SetLaunchAtLoginFunc(func(bool) { called = true })
	svc.SetToggleSystemTrayFunc(func(bool) { called = true })
	current.GlobalShortcut = "option+shift+space"
	current.LaunchAtLogin = true
	if _, err := svc.SaveSettings(*current); err == nil {
		t.Fatal("expected persistence failure")
	}
	cached, _ := svc.GetSettings()
	if called {
		t.Fatal("failed save changed system integrations")
	}
	if cached.GlobalShortcut != original.GlobalShortcut || cached.LaunchAtLogin != original.LaunchAtLogin {
		t.Fatal("failed save changed cached settings")
	}
}
