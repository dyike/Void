package commands

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ─── Icon Extraction ───────────────────────────────────────────────────────

func (s *Service) iconForApp(appPath string) string {
	cacheKey := fmt.Sprintf("%x", md5.Sum([]byte(appPath)))
	cacheFile := filepath.Join(s.iconCacheDir, "v1-"+cacheKey+".b64")

	if data, err := os.ReadFile(cacheFile); err == nil {
		return string(data)
	}

	dataURL := s.extractIconSips(appPath)
	if dataURL != "" {
		_ = os.WriteFile(cacheFile, []byte(dataURL), 0644)
	}
	return dataURL
}

func (s *Service) extractIconSips(appPath string) string {
	resourcesDir := filepath.Join(appPath, "Contents", "Resources")
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return ""
	}

	// Prefer AppIcon.icns
	var icnsPath string
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".icns") {
			if icnsPath == "" || strings.EqualFold(name, "AppIcon.icns") {
				icnsPath = filepath.Join(resourcesDir, name)
			}
		}
	}
	if icnsPath == "" {
		return ""
	}

	tmpPNG, err := os.CreateTemp("", "void-icon-*.png")
	if err != nil {
		return ""
	}
	tmpPNG.Close()
	defer os.Remove(tmpPNG.Name())

	cmd := exec.Command(
		"/usr/bin/sips",
		"-s", "format", "png",
		"-z", "64", "64",
		icnsPath,
		"--out", tmpPNG.Name(),
	)
	if err := cmd.Run(); err != nil {
		return ""
	}

	data, err := os.ReadFile(tmpPNG.Name())
	if err != nil || len(data) < 100 {
		return ""
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}
