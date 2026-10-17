package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBundledAppIconFallback(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "Void.app", "Contents", "MacOS", "void")
	icon := filepath.Join(root, "Void.app", "Contents", "Resources", "icon.icns")
	if hasBundledAppIcon(executable) {
		t.Fatal("missing bundle icon must retain fallback")
	}
	if err := os.MkdirAll(filepath.Dir(icon), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(icon, []byte("icns"), 0644); err != nil {
		t.Fatal(err)
	}
	if !hasBundledAppIcon(executable) {
		t.Fatal("packaged icon was not detected")
	}
	if hasBundledAppIcon(filepath.Join(root, "Contents", "MacOS", "void")) {
		t.Fatal("plain binary must retain fallback")
	}
	if err := os.Truncate(icon, 0); err != nil {
		t.Fatal(err)
	}
	if hasBundledAppIcon(executable) {
		t.Fatal("empty icon must retain fallback")
	}
}
