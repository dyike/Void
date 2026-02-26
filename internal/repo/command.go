package repo

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/service/filesearch"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type CommandRepo struct {
	iconCacheDir string
}

func NewCommandRepo(dataDir string) *CommandRepo {
	iconCacheDir := filepath.Join(dataDir, "icon-cache")
	_ = os.MkdirAll(iconCacheDir, 0755)
	return &CommandRepo{
		iconCacheDir: iconCacheDir,
	}
}

// ─── Spotlight Discovery ───────────────────────────────────────────────────

func (r *CommandRepo) DiscoverAppsSpotlight() []model.CommandInfo {
	// Query Spotlight for all application bundles
	out, err := exec.Command(
		"mdfind",
		"kMDItemContentType == 'com.apple.application-bundle'",
	).Output()
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var results []model.CommandInfo

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

		iconDataURL := r.iconForApp(line)

		results = append(results, model.CommandInfo{
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

// DiscoverSettingsPanes discovers macOS System Settings panes from PreferencePanes.
func (r *CommandRepo) DiscoverSettingsPanes() []model.CommandInfo {
	searchDirs := []string{
		"/System/Library/PreferencePanes",
		"/Library/PreferencePanes",
		filepath.Join(os.Getenv("HOME"), "Library", "PreferencePanes"),
	}

	seen := map[string]bool{}
	var results []model.CommandInfo

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
			results = append(results, model.CommandInfo{
				ID:       id,
				Title:    name,
				Subtitle: "System Settings",
				Icon:     "Wrench",
				Category: "settings",
				Path:     fullPath,
			})
		}
	}
	return results
}

// ─── Directory Scan (fallback) ─────────────────────────────────────────────

func (r *CommandRepo) DiscoverAppsDir() ([]model.CommandInfo, error) {
	searchDirs := []string{
		"/Applications",
		"/System/Applications",
		"/System/Applications/Utilities",
		filepath.Join(os.Getenv("HOME"), "Applications"),
	}

	seen := map[string]bool{}
	var results []model.CommandInfo

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
			iconDataURL := r.iconForApp(fullPath)

			results = append(results, model.CommandInfo{
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

// ─── File Search ──────────────────────────────────────────────────────────

// SearchFiles searches for files by name using fuzzy matching (pure Go, no Spotlight).
func (r *CommandRepo) SearchFiles(query string) ([]model.CommandInfo, error) {
	if query == "" {
		return nil, nil
	}
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		log.Println("filesearch: HOME env is empty")
		return nil, nil
	}
	log.Printf("filesearch: searching query=%q in %s", query, homeDir)
	start := time.Now()
	results := filesearch.SearchFiles(homeDir, query)
	elapsed := time.Since(start)
	log.Printf("filesearch: found %d results in %v", len(results), elapsed)
	if len(results) > 0 {
		log.Printf("filesearch: top result: %s (score=%d)", results[0].Path, results[0].Score)
	}
	var cmds []model.CommandInfo
	for _, res := range results {
		cmds = append(cmds, model.CommandInfo{
			ID:       "file-" + slugify(res.Path),
			Title:    res.Name,
			Subtitle: res.Dir,
			Icon:     fileIcon(res.Path),
			Category: "file",
			Path:     res.Path,
			Score:    float64(res.Score),
		})
	}
	return cmds, nil
}

// SearchFileContents searches inside file contents using grep (pure Go).
func (r *CommandRepo) SearchFileContents(query string) ([]model.CommandInfo, error) {
	if query == "" {
		return nil, nil
	}
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return nil, nil
	}
	results := filesearch.SearchFileContents(homeDir, query)
	var cmds []model.CommandInfo
	for _, res := range results {
		subtitle := res.Dir
		if res.LineNum > 0 {
			subtitle = fmt.Sprintf("L%d: %s", res.LineNum, res.LineText)
		}
		cmds = append(cmds, model.CommandInfo{
			ID:       "file-" + slugify(res.Path),
			Title:    res.Name,
			Subtitle: subtitle,
			Icon:     fileIcon(res.Path),
			Category: "file",
			Path:     res.Path,
			Score:    float64(res.Score),
		})
	}
	return cmds, nil
}

// ─── Icon Extraction ───────────────────────────────────────────────────────

func (r *CommandRepo) iconForApp(appPath string) string {
	cacheKey := fmt.Sprintf("%x", md5.Sum([]byte(appPath)))
	cacheFile := filepath.Join(r.iconCacheDir, "v1-"+cacheKey+".b64")

	if data, err := os.ReadFile(cacheFile); err == nil {
		return string(data)
	}

	dataURL := r.extractIconSips(appPath)
	if dataURL != "" {
		_ = os.WriteFile(cacheFile, []byte(dataURL), 0644)
	}
	return dataURL
}

func (r *CommandRepo) extractIconSips(appPath string) string {
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

// ─── Execute ───────────────────────────────────────────────────────────────

// LaunchApp opens an application by its path.
func (r *CommandRepo) LaunchApp(appPath string) error {
	cmd := exec.Command("open", appPath)
	return cmd.Run()
}

// OpenURL opens a URL in the default browser.
func (r *CommandRepo) OpenURL(url string) error {
	return exec.Command("open", url).Run()
}

// HideWindow hides the main application window.
func (r *CommandRepo) HideWindow() {
	if app := application.Get(); app != nil {
		app.Hide()
	}
}

// OpenSystemSettings opens a macOS Settings pane by bundle identifier.
func (r *CommandRepo) OpenSystemSettings(bundleID string) error {
	cmd := exec.Command("open", "x-apple.systempreferences:"+bundleID)
	if err := cmd.Run(); err != nil {
		return exec.Command("open", "-b", "com.apple.systempreferences").Run()
	}
	return nil
}

// RevealInFinder reveals a file in Finder (selects it).
func (r *CommandRepo) RevealInFinder(path string) error {
	return exec.Command("open", "-R", path).Run()
}

// QuickLook opens Quick Look preview for a file.
func (r *CommandRepo) QuickLook(path string) error {
	return exec.Command("qlmanage", "-p", path).Start()
}

// CopyToClipboard copies text to the macOS clipboard.
func (r *CommandRepo) CopyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// ExecuteSystemCommand runs a built-in macOS system command by ID.
func (r *CommandRepo) ExecuteSystemCommand(id string, terminalApp string) error {
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
		if terminalApp != "" {
			appName = terminalApp
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
