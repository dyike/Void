package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstalledAppsDeduplicateAndExcludeBackups(t *testing.T) {
	dir := t.TempDir()
	roots := []string{filepath.Join(dir, "Applications"), filepath.Join(dir, "UserApps")}
	app := func(path, id, extra string) string {
		t.Helper()
		contents := filepath.Join(path, "Contents")
		if err := os.MkdirAll(contents, 0755); err != nil {
			t.Fatal(err)
		}
		plist := fmt.Sprintf(`<plist><dict><key>CFBundleIdentifier</key><string>%s</string>%s</dict></plist>`, id, extra)
		if err := os.WriteFile(filepath.Join(contents, "Info.plist"), []byte(plist), 0600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	shared := app(filepath.Join(roots[0], "Nix Apps", "Alacritty.app"), "org.alacritty", "")
	copy := app(filepath.Join(roots[1], "Alacritty.app"), "org.alacritty", "")
	backup := app(filepath.Join(roots[1], "Home Manager Apps.backup-2026-09-27", "Alacritty.app"), "old.alacritty", "")
	helper := app(filepath.Join(shared, "Contents", "Helper.app"), "org.helper", "")
	background := app(filepath.Join(roots[0], "Agent.app"), "org.agent", "<key>LSBackgroundOnly</key><true/>")
	outside := app(filepath.Join(dir, "Downloads", "Build.app"), "org.build", "")
	alias := filepath.Join(roots[0], "Nix Apps", "Long Alacritty Alias.app")
	if err := os.Symlink(shared, alias); err != nil {
		t.Fatal(err)
	}
	r := NewCommandRepo(t.TempDir())
	paths := []string{backup, copy, helper, background, outside, shared, alias, filepath.Join(roots[0], "Missing.app")}
	for range 2 {
		apps := r.appsFromPaths(paths, roots)
		if len(apps) != 1 || apps[0].Path != shared || apps[0].ID != "app-alacritty" {
			t.Fatalf("apps = %+v", apps)
		}
		for i, j := 0, len(paths)-1; i < j; i, j = i+1, j-1 {
			paths[i], paths[j] = paths[j], paths[i]
		}
	}
}

func TestDirectoryDiscoveryFollowsPackageManagerLinks(t *testing.T) {
	dir := t.TempDir()
	root, store := filepath.Join(dir, "Applications"), filepath.Join(dir, "store")
	app := filepath.Join(store, "Alacritty.app")
	if err := os.MkdirAll(filepath.Join(app, "Contents"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(`<plist><dict><key>CFBundleIdentifier</key><string>org.alacritty</string></dict></plist>`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(store, filepath.Join(root, "Home Manager Apps")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(store, "loop")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(store, filepath.Join(root, "Home Manager Apps.backup-2026-09-27")); err != nil {
		t.Fatal(err)
	}
	roots := []string{root}
	apps := NewCommandRepo(t.TempDir()).appsFromPaths(scanApplicationPaths(roots), roots)
	if len(apps) != 1 || apps[0].Path != filepath.Join(root, "Home Manager Apps", "Alacritty.app") {
		t.Fatalf("apps = %+v", apps)
	}
}

func TestBinaryAppMetadata(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS plist conversion")
	}
	dir := filepath.Join(t.TempDir(), "Example.app", "Contents")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "Info.plist")
	data := `<plist><dict><key>CFBundleIdentifier</key><string>org.example.app</string><key>CFBundleURLTypes</key><array><dict><key>CFBundleIdentifier</key><string>nested.value</string></dict></array><key>LSBackgroundOnly</key><false/></dict></plist>`
	if err := os.WriteFile(file, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("/usr/bin/plutil", "-convert", "binary1", file).CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	metadata, ok := readAppMetadata(filepath.Dir(dir))
	if !ok || metadata.bundleID != "org.example.app" || metadata.background {
		t.Fatalf("metadata = %+v, valid=%v", metadata, ok)
	}
}
