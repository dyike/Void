package commands

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dyike/Void/internal/service/frecency"
)

// CommandInfo represents a launchable command.
type CommandInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	// Base64 PNG data URL or empty
	IconDataURL string `json:"iconDataUrl,omitempty"`
	IconEmoji   string `json:"iconEmoji,omitempty"`
	Category    string `json:"category"` // "app" | "settings" | "system" | "snippet" | "file"
	// App path for apps, bundle ID for settings
	Path  string  `json:"path,omitempty"`
	Score float64 `json:"score,omitempty"`
}

// Service discovers and caches macOS commands.
type Service struct {
	mu           sync.RWMutex
	cache        []CommandInfo
	cacheTime    time.Time
	iconCacheDir string
	cacheTTL     time.Duration
	frecency     *frecency.Store
}

// NewService creates a new commands service.
func NewService(dataDir string) *Service {
	iconCacheDir := filepath.Join(dataDir, "icon-cache")
	_ = os.MkdirAll(iconCacheDir, 0755)
	return &Service{
		iconCacheDir: iconCacheDir,
		cacheTTL:     20 * time.Minute,
		frecency:     frecency.NewStore(dataDir),
	}
}

// GetCommands returns cached + frecency-boosted commands.
func (s *Service) GetCommands() ([]CommandInfo, error) {
	s.mu.RLock()
	if s.cache != nil && time.Since(s.cacheTime) < s.cacheTTL {
		result := s.withFrecency(s.cache)
		s.mu.RUnlock()
		return result, nil
	}
	s.mu.RUnlock()

	return s.RefreshCommands()
}

// RefreshCommands forces a re-discovery.
func (s *Service) RefreshCommands() ([]CommandInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var all []CommandInfo
	all = append(all, systemCommands()...)

	// Use Spotlight for comprehensive app discovery
	apps := s.discoverAppsSpotlight()
	if len(apps) == 0 {
		// Fallback to directory scan
		apps, _ = s.discoverAppsDir()
	}
	all = append(all, apps...)

	s.cache = all
	s.cacheTime = time.Now()

	return s.withFrecency(all), nil
}

// RecordUsage records that a command was used (for frecency).
func (s *Service) RecordUsage(id string) {
	s.frecency.RecordAccess(id)
}

// LaunchApp opens an application by its path.
func (s *Service) LaunchApp(appPath string) error {
	cmd := exec.Command("open", appPath)
	return cmd.Run()
}

// OpenURL opens a URL in the default browser.
func (s *Service) OpenURL(url string) error {
	return exec.Command("open", url).Run()
}

// OpenSystemSettings opens a macOS Settings pane by bundle identifier.
func (s *Service) OpenSystemSettings(bundleID string) error {
	cmd := exec.Command("open", "x-apple.systempreferences:"+bundleID)
	if err := cmd.Run(); err != nil {
		return exec.Command("open", "-b", "com.apple.systempreferences").Run()
	}
	return nil
}

// SearchFiles searches for files using Spotlight.
func (s *Service) SearchFiles(query string) ([]CommandInfo, error) {
	if query == "" {
		return nil, nil
	}

	out, err := exec.Command(
		"mdfind",
		"-onlyin", os.Getenv("HOME"),
		"-name", query,
		"-attr", "kMDItemPath",
	).Output()
	if err != nil {
		return nil, err
	}

	var results []CommandInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name := filepath.Base(line)
		results = append(results, CommandInfo{
			ID:        "file-" + slugify(line),
			Title:     name,
			Subtitle:  filepath.Dir(line),
			IconEmoji: fileEmoji(line),
			Category:  "file",
			Path:      line,
		})
		if len(results) >= 20 {
			break
		}
	}
	return results, nil
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
			ID:        "system-snippets",
			Title:     "Snippets",
			Subtitle:  "Manage text snippets",
			IconEmoji: "📝",
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

// ─── Spotlight Discovery ───────────────────────────────────────────────────

func (s *Service) discoverAppsSpotlight() []CommandInfo {
	// Query Spotlight for all application bundles
	out, err := exec.Command(
		"mdfind",
		"kMDItemContentType == 'com.apple.application-bundle'",
	).Output()
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var results []CommandInfo

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}

		// Skip hidden apps and OS internals
		if strings.Contains(line, "/Library/Apple/") ||
			strings.Contains(line, "Xcode.app/Contents/") ||
			strings.Contains(line, ".bundle") {
			continue
		}

		seen[line] = true
		name := strings.TrimSuffix(filepath.Base(line), ".app")
		id := "app-" + slugify(name)

		iconDataURL := s.iconForApp(line)

		results = append(results, CommandInfo{
			ID:          id,
			Title:       name,
			Subtitle:    filepath.Dir(line),
			IconDataURL: iconDataURL,
			Category:    "app",
			Path:        line,
		})
	}

	// Sort by name
	sort.Slice(results, func(i, j int) bool {
		return results[i].Title < results[j].Title
	})

	return results
}

// ─── Directory Scan (fallback) ─────────────────────────────────────────────

func (s *Service) discoverAppsDir() ([]CommandInfo, error) {
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

// ─── Frecency ──────────────────────────────────────────────────────────────

func (s *Service) withFrecency(cmds []CommandInfo) []CommandInfo {
	result := make([]CommandInfo, len(cmds))
	copy(result, cmds)
	for i := range result {
		result[i].Score = s.frecency.GetScore(result[i].ID)
	}
	// Stable sort: high frecency first, then original order
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})
	return result
}

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

// ─── Helpers ───────────────────────────────────────────────────────────────

func fileEmoji(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return "📄"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return "🖼️"
	case ".mp3", ".wav", ".flac", ".aac":
		return "🎵"
	case ".mp4", ".mov", ".avi", ".mkv":
		return "🎬"
	case ".zip", ".tar", ".gz", ".rar":
		return "📦"
	case ".go", ".py", ".js", ".ts", ".rs", ".swift":
		return "💻"
	case ".md", ".txt":
		return "📝"
	case ".xlsx", ".csv":
		return "📊"
	case ".app":
		return "🖥️"
	default:
		return "📄"
	}
}

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
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return result
}
