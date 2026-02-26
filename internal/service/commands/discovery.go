package commands

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dyike/Void/internal/model"
)

// ─── Spotlight Discovery ───────────────────────────────────────────────────

func (s *Service) discoverAppsSpotlight() []model.CommandInfo {
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

		iconDataURL := s.iconForApp(line)

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

// discoverSettingsPanes discovers macOS System Settings panes from PreferencePanes.
func (s *Service) discoverSettingsPanes() []model.CommandInfo {
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

func (s *Service) discoverAppsDir() ([]model.CommandInfo, error) {
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
			iconDataURL := s.iconForApp(fullPath)

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

// ─── Frecency ──────────────────────────────────────────────────────────────

func (s *Service) withFrecency(cmds []model.CommandInfo) []model.CommandInfo {
	result := make([]model.CommandInfo, len(cmds))
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
