package main

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/dyike/Void/internal/branding"
	keelwindow "github.com/dyike/keel/ui/window"
)

// Keel's macOS build and development bundles already carry the complete
// multi-resolution icon. Rebuilding it at launch retains an unnecessary
// decoded image and native backing surfaces. Plain binaries still need it.
func setFallbackAppIcon() error {
	if runtime.GOOS == "darwin" {
		if executable, err := os.Executable(); err == nil && hasBundledAppIcon(executable) {
			return nil
		}
	}
	return keelwindow.SetIcon(branding.AppIcon)
}

func hasBundledAppIcon(executable string) bool {
	macOS := filepath.Dir(executable)
	contents := filepath.Dir(macOS)
	if filepath.Base(macOS) != "MacOS" || filepath.Base(contents) != "Contents" || filepath.Ext(filepath.Dir(contents)) != ".app" {
		return false
	}
	info, err := os.Stat(filepath.Join(contents, "Resources", "icon.icns"))
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}
