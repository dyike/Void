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

	// Discover System Settings panes
	settingsPanes := s.discoverSettingsPanes()
	all = append(all, settingsPanes...)

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
		return exec.Command("open", "-a", "Terminal").Run()
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
		// ── Void built-ins ──────────────────────────
		{
			ID: "system-clipboard-manager", Title: "Clipboard History",
			Subtitle: "Browse your clipboard history", IconEmoji: "📋", Category: "system",
		},
		{
			ID: "system-ai-chat", Title: "AI Chat",
			Subtitle: "Chat with AI assistant", IconEmoji: "✨", Category: "system",
		},
		{
			ID: "system-snippets", Title: "Snippets",
			Subtitle: "Manage text snippets", IconEmoji: "📝", Category: "system",
		},
		{
			ID: "system-settings", Title: "Void Settings",
			Subtitle: "Configure Void", IconEmoji: "⚙️", Category: "system",
		},
		// ── macOS system actions ───────────────────
		{
			ID: "system-lock-screen", Title: "Lock Screen",
			Subtitle: "Turn off displays and lock", IconEmoji: "🔒", Category: "system",
		},
		{
			ID: "system-sleep", Title: "Sleep",
			Subtitle: "Put Mac to sleep", IconEmoji: "💤", Category: "system",
		},
		{
			ID: "system-empty-trash", Title: "Empty Trash",
			Subtitle: "Permanently delete trashed files", IconEmoji: "🗑️", Category: "system",
		},
		{
			ID: "system-restart", Title: "Restart",
			Subtitle: "Restart your Mac", IconEmoji: "🔄", Category: "system",
		},
		{
			ID: "system-shutdown", Title: "Shut Down",
			Subtitle: "Shut down your Mac", IconEmoji: "⏻", Category: "system",
		},
		{
			ID: "system-open-finder", Title: "Open Finder",
			Subtitle: "Open Finder", IconEmoji: "📂", Category: "system",
		},
		{
			ID: "system-open-terminal", Title: "Open Terminal",
			Subtitle: "Open Terminal", IconEmoji: "💻", Category: "system",
		},
		// ── Window management ──────────────────────
		{
			ID: "window-left-half", Title: "Left Half",
			Subtitle: "Move window to left half", IconEmoji: "◀️", Category: "system",
		},
		{
			ID: "window-right-half", Title: "Right Half",
			Subtitle: "Move window to right half", IconEmoji: "▶️", Category: "system",
		},
		{
			ID: "window-top-half", Title: "Top Half",
			Subtitle: "Move window to top half", IconEmoji: "🔼", Category: "system",
		},
		{
			ID: "window-bottom-half", Title: "Bottom Half",
			Subtitle: "Move window to bottom half", IconEmoji: "🔽", Category: "system",
		},
		{
			ID: "window-top-left", Title: "Top Left Quarter",
			Subtitle: "Move window to top-left quarter", IconEmoji: "↖️", Category: "system",
		},
		{
			ID: "window-top-right", Title: "Top Right Quarter",
			Subtitle: "Move window to top-right quarter", IconEmoji: "↗️", Category: "system",
		},
		{
			ID: "window-bottom-left", Title: "Bottom Left Quarter",
			Subtitle: "Move window to bottom-left quarter", IconEmoji: "↙️", Category: "system",
		},
		{
			ID: "window-bottom-right", Title: "Bottom Right Quarter",
			Subtitle: "Move window to bottom-right quarter", IconEmoji: "↘️", Category: "system",
		},
		{
			ID: "window-maximize", Title: "Maximize",
			Subtitle: "Maximize window to fill screen", IconEmoji: "⬜", Category: "system",
		},
		{
			ID: "window-center", Title: "Center Window",
			Subtitle: "Center window on screen", IconEmoji: "🎯", Category: "system",
		},
	}
}

// ─── Window Management ─────────────────────────────────────────────────────

// moveWindow positions the frontmost window using fractional screen coordinates.
// xFrac, yFrac = position as fraction of screen; wFrac, hFrac = size as fraction.
func moveWindow(xFrac, yFrac, wFrac, hFrac float64) error {
	script := fmt.Sprintf(`
tell application "Finder"
	set _b to bounds of window of desktop
	set _sw to (item 3 of _b) - (item 1 of _b)
	set _sh to (item 4 of _b) - (item 2 of _b)
end tell
set _x to round(%f * _sw)
set _y to round(%f * _sh)
set _w to round(%f * _sw)
set _h to round(%f * _sh)
tell application "System Events"
	set frontApp to first application process whose frontmost is true
	tell window 1 of frontApp
		set position to {_x, _y}
		set size to {_w, _h}
	end tell
end tell
`, xFrac, yFrac, wFrac, hFrac)
	return exec.Command("osascript", "-e", script).Run()
}

// centerWindow centers the frontmost window on screen at 60%% width, 80%% height.
func centerWindow() error {
	script := `
tell application "Finder"
	set _b to bounds of window of desktop
	set _sw to (item 3 of _b) - (item 1 of _b)
	set _sh to (item 4 of _b) - (item 2 of _b)
end tell
set _w to round(0.6 * _sw)
set _h to round(0.8 * _sh)
set _x to round((_sw - _w) / 2)
set _y to round((_sh - _h) / 2)
tell application "System Events"
	set frontApp to first application process whose frontmost is true
	tell window 1 of frontApp
		set position to {_x, _y}
		set size to {_w, _h}
	end tell
end tell
`
	return exec.Command("osascript", "-e", script).Run()
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

// ─── System Settings Pane Discovery ───────────────────────────────────────

// discoverSettingsPanes discovers macOS System Settings panes from PreferencePanes.
func (s *Service) discoverSettingsPanes() []CommandInfo {
	searchDirs := []string{
		"/System/Library/PreferencePanes",
		"/Library/PreferencePanes",
		filepath.Join(os.Getenv("HOME"), "Library", "PreferencePanes"),
	}

	seen := map[string]bool{}
	var results []CommandInfo

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".prefPane") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".prefPane")
			id := "settings-" + slugify(name)
			if seen[id] {
				continue
			}
			seen[id] = true
			fullPath := filepath.Join(dir, e.Name())
			results = append(results, CommandInfo{
				ID:        id,
				Title:     name,
				Subtitle:  "System Settings",
				IconEmoji: "🔧",
				Category:  "settings",
				Path:      fullPath,
			})
		}
	}
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
