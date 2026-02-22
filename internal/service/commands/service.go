package commands

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CommandInfo represents a launchable command.
type CommandInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	// Base64 PNG data URL or empty
	IconDataURL string `json:"iconDataUrl,omitempty"`
	IconEmoji   string `json:"iconEmoji,omitempty"`
	Category    string `json:"category"` // "app" | "settings" | "system"
	// App path for apps, bundle ID for settings
	Path string `json:"path,omitempty"`
}

// Service discovers and caches macOS commands.
type Service struct {
	mu           sync.RWMutex
	cache        []CommandInfo
	cacheTime    time.Time
	iconCacheDir string
	cacheTTL     time.Duration
}

// NewService creates a new commands service.
func NewService(dataDir string) *Service {
	iconCacheDir := filepath.Join(dataDir, "icon-cache")
	_ = os.MkdirAll(iconCacheDir, 0755)
	return &Service{
		iconCacheDir: iconCacheDir,
		cacheTTL:     30 * time.Minute,
	}
}

// GetCommands returns cached commands, refreshing if stale.
func (s *Service) GetCommands() ([]CommandInfo, error) {
	s.mu.RLock()
	if s.cache != nil && time.Since(s.cacheTime) < s.cacheTTL {
		result := make([]CommandInfo, len(s.cache))
		copy(result, s.cache)
		s.mu.RUnlock()
		return result, nil
	}
	s.mu.RUnlock()

	return s.RefreshCommands()
}

// RefreshCommands forces a re-discovery and returns the new list.
func (s *Service) RefreshCommands() ([]CommandInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var all []CommandInfo

	// System commands first
	all = append(all, systemCommands()...)

	// Discover apps
	apps, err := s.discoverApps()
	if err == nil {
		all = append(all, apps...)
	}

	s.cache = all
	s.cacheTime = time.Now()

	result := make([]CommandInfo, len(all))
	copy(result, all)
	return result, nil
}

// LaunchApp opens an application by its path.
func (s *Service) LaunchApp(appPath string) error {
	cmd := exec.Command("open", appPath)
	return cmd.Run()
}

// OpenSystemSettings opens a macOS Settings pane by bundle identifier.
func (s *Service) OpenSystemSettings(bundleID string) error {
	script := fmt.Sprintf(
		`tell application "System Settings" to reveal anchor "%s" of pane id "%s"`,
		bundleID, bundleID,
	)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		// Fallback: just open System Settings
		return exec.Command("open", "-b", "com.apple.systempreferences").Run()
	}
	return nil
}

// ─── System Commands ───────────────────────────────────────────────────────

func systemCommands() []CommandInfo {
	return []CommandInfo{
		{
			ID:        "system-clipboard-manager",
			Title:     "Clipboard History",
			Subtitle:  "Browse your clipboard history",
			IconEmoji: "📋",
			Category:  "system",
		},
		{
			ID:        "system-ai-chat",
			Title:     "AI Chat",
			Subtitle:  "Chat with AI assistant",
			IconEmoji: "✨",
			Category:  "system",
		},
		{
			ID:        "system-settings",
			Title:     "Void Settings",
			Subtitle:  "Configure Void",
			IconEmoji: "⚙️",
			Category:  "system",
		},
	}
}

// ─── App Discovery ─────────────────────────────────────────────────────────

func (s *Service) discoverApps() ([]CommandInfo, error) {
	searchDirs := []string{
		"/Applications",
		"/System/Applications",
		"/System/Applications/Utilities",
		filepath.Join(os.Getenv("HOME"), "Applications"),
	}

	seen := map[string]bool{}
	var results []CommandInfo

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".app") {
				continue
			}
			fullPath := filepath.Join(dir, e.Name())
			if seen[fullPath] {
				continue
			}
			seen[fullPath] = true

			name := strings.TrimSuffix(e.Name(), ".app")
			id := "app-" + slugify(name)

			iconDataURL := s.iconForApp(fullPath)

			results = append(results, CommandInfo{
				ID:          id,
				Title:       name,
				Subtitle:    dir,
				IconDataURL: iconDataURL,
				Category:    "app",
				Path:        fullPath,
			})
		}
	}

	return results, nil
}

// ─── Icon Extraction ───────────────────────────────────────────────────────

func (s *Service) iconForApp(appPath string) string {
	cacheKey := fmt.Sprintf("%x", md5.Sum([]byte(appPath)))
	cacheFile := filepath.Join(s.iconCacheDir, "v1-"+cacheKey+".b64")

	// Check disk cache
	if data, err := os.ReadFile(cacheFile); err == nil {
		return string(data)
	}

	// Try to extract icon via sips
	dataURL := s.extractIconSips(appPath)
	if dataURL != "" {
		_ = os.WriteFile(cacheFile, []byte(dataURL), 0644)
	}
	return dataURL
}

func (s *Service) extractIconSips(appPath string) string {
	// Find the first .icns file in Contents/Resources
	resourcesDir := filepath.Join(appPath, "Contents", "Resources")
	entries, err := os.ReadDir(resourcesDir)
	if err != nil {
		return ""
	}

	var icnsPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".icns") {
			icnsPath = filepath.Join(resourcesDir, e.Name())
			break
		}
	}
	if icnsPath == "" {
		return ""
	}

	// Convert to PNG via sips
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

// ─── Helpers ───────────────────────────────────────────────────────────────

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	// Collapse multiple dashes
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return result
}
