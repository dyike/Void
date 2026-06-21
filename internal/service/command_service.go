package service

import (
	"log"
	"sort"
	"sync"
	"time"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/repo"
)

// CommandService discovers and caches macOS commands.
type CommandService struct {
	mu        sync.RWMutex
	cache     []model.CommandInfo
	cacheTime time.Time
	cacheTTL  time.Duration
	frecency  *FrecencyStore
	settings  *SettingsService
	repo      *repo.CommandRepo
}

// NewCommandService creates a new commands service.
func NewCommandService(dataDir string, settingsSvc *SettingsService, commandRepo *repo.CommandRepo) *CommandService {
	return &CommandService{
		cacheTTL: 20 * time.Minute,
		frecency: NewFrecencyStore(dataDir),
		settings: settingsSvc,
		repo:     commandRepo,
	}
}

// GetCommands returns cached + frecency-boosted commands.
func (s *CommandService) GetCommands() ([]model.CommandInfo, error) {
	s.mu.RLock()
	if len(s.cache) > 0 && time.Since(s.cacheTime) < s.cacheTTL {
		c := make([]model.CommandInfo, len(s.cache))
		copy(c, s.cache)
		s.mu.RUnlock()
		return s.withFrecency(c), nil
	}
	s.mu.RUnlock()
	return s.RefreshCommands()
}

// RefreshCommands forces a re-discovery.
func (s *CommandService) RefreshCommands() ([]model.CommandInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var all []model.CommandInfo
	all = append(all, systemCommands()...)

	apps := s.repo.DiscoverAppsSpotlight()
	if len(apps) == 0 {
		log.Println("Spotlight found no apps, falling back to directory scan.")
		apps, _ = s.repo.DiscoverAppsDir()
	}
	all = append(all, apps...)

	panes := s.repo.DiscoverSettingsPanes()
	all = append(all, panes...)

	s.cache = make([]model.CommandInfo, len(all))
	copy(s.cache, all)
	s.cacheTime = time.Now()

	log.Printf("commands: refreshed %d items (apps=%d settings=%d builtins=%d)\n",
		len(all), len(apps), len(panes), len(systemCommands()))

	return s.withFrecency(all), nil
}

// RecordUsage records that a command was used (for frecency).
func (s *CommandService) RecordUsage(id string) {
	s.frecency.RecordAccess(id)
}

// SearchFiles searches for files by name.
func (s *CommandService) SearchFiles(query string) ([]model.CommandInfo, error) {
	return s.repo.SearchFiles(query)
}

// SearchFileContents searches inside file contents.
func (s *CommandService) SearchFileContents(query string) ([]model.CommandInfo, error) {
	return s.repo.SearchFileContents(query)
}

// LaunchApp opens an application by its path.
func (s *CommandService) LaunchApp(appPath string) error {
	return s.repo.LaunchApp(appPath)
}

// OpenURL opens a URL in the default browser.
func (s *CommandService) OpenURL(url string) error {
	return s.repo.OpenURL(url)
}

// HideWindow hides the main application window.
func (s *CommandService) HideWindow() {
	s.repo.HideWindow()
}

// OpenSystemSettings opens a macOS Settings pane by bundle identifier.
func (s *CommandService) OpenSystemSettings(bundleID string) error {
	return s.repo.OpenSystemSettings(bundleID)
}

// RevealInFinder reveals a file in Finder.
func (s *CommandService) RevealInFinder(path string) error {
	return s.repo.RevealInFinder(path)
}

// QuickLook opens Quick Look preview for a file.
func (s *CommandService) QuickLook(path string) error {
	return s.repo.QuickLook(path)
}

// CopyToClipboard copies text to the macOS clipboard.
func (s *CommandService) CopyToClipboard(text string) error {
	return s.repo.CopyToClipboard(text)
}

// ExecuteSystemCommand runs a built-in macOS system command by ID.
func (s *CommandService) ExecuteSystemCommand(id string) error {
	terminalApp := ""
	if s.settings != nil {
		if cfg, err := s.settings.GetSettings(); err == nil && cfg != nil {
			terminalApp = cfg.TerminalApp
		}
	}
	return s.repo.ExecuteSystemCommand(id, terminalApp)
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func (s *CommandService) withFrecency(cmds []model.CommandInfo) []model.CommandInfo {
	result := make([]model.CommandInfo, len(cmds))
	copy(result, cmds)
	for i := range result {
		result[i].Score = s.frecency.GetScore(result[i].ID)
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})
	return result
}

func systemCommands() []model.CommandInfo {
	return []model.CommandInfo{
		// ── Void built-ins ──────────────────────────
		{
			ID: "system-clipboard-manager", Title: "Clipboard History",
			Subtitle: "Browse your clipboard history", Icon: "Clipboard", Category: "system",
		},
		{
			ID: "system-ai-chat", Title: "AI Chat",
			Subtitle: "Chat with AI assistant", Icon: "Sparkles", Category: "system",
		},
		{
			ID: "system-snippets", Title: "Snippets",
			Subtitle: "Manage text snippets", Icon: "FileText", Category: "system",
		},
		{
			ID: "system-apple-containers", Title: "Apple Containers",
			Subtitle: "Manage Apple container workloads", Icon: "Container", Category: "system",
		},
		// ── macOS system actions ───────────────────
		{
			ID: "system-lock-screen", Title: "Lock Screen",
			Subtitle: "Turn off displays and lock", Icon: "Lock", Category: "system",
		},
		{
			ID: "system-sleep", Title: "Sleep",
			Subtitle: "Put Mac to sleep", Icon: "Moon", Category: "system",
		},
		{
			ID: "system-empty-trash", Title: "Empty Trash",
			Subtitle: "Permanently delete trashed files", Icon: "Trash", Category: "system",
		},
		{
			ID: "system-restart", Title: "Restart",
			Subtitle: "Restart your Mac", Icon: "RotateCw", Category: "system",
		},
		{
			ID: "system-shutdown", Title: "Shut Down",
			Subtitle: "Shut down your Mac", Icon: "Power", Category: "system",
		},
		{
			ID: "system-open-finder", Title: "Open Finder",
			Subtitle: "Open Finder", Icon: "Folder", Category: "system",
		},
		{
			ID: "system-open-terminal", Title: "Open Terminal",
			Subtitle: "Open Terminal", Icon: "Terminal", Category: "system",
		},
		// ── Window management ─────────────────────
		{
			ID: "window-left-half", Title: "Left Half",
			Subtitle: "Move window to left half", Icon: "ArrowLeft", Category: "system",
		},
		{
			ID: "window-right-half", Title: "Right Half",
			Subtitle: "Move window to right half", Icon: "ArrowRight", Category: "system",
		},
		{
			ID: "window-top-half", Title: "Top Half",
			Subtitle: "Move window to top half", Icon: "ArrowUp", Category: "system",
		},
		{
			ID: "window-bottom-half", Title: "Bottom Half",
			Subtitle: "Move window to bottom half", Icon: "ArrowDown", Category: "system",
		},
		{
			ID: "window-top-left", Title: "Top Left Quarter",
			Subtitle: "Move window to top-left quarter", Icon: "ArrowUpLeft", Category: "system",
		},
		{
			ID: "window-top-right", Title: "Top Right Quarter",
			Subtitle: "Move window to top-right quarter", Icon: "ArrowUpRight", Category: "system",
		},
		{
			ID: "window-bottom-left", Title: "Bottom Left Quarter",
			Subtitle: "Move window to bottom-left quarter", Icon: "ArrowDownLeft", Category: "system",
		},
		{
			ID: "window-bottom-right", Title: "Bottom Right Quarter",
			Subtitle: "Move window to bottom-right quarter", Icon: "ArrowDownRight", Category: "system",
		},
		{
			ID: "window-maximize", Title: "Maximize",
			Subtitle: "Maximize window to fill screen", Icon: "Maximize", Category: "system",
		},
		{
			ID: "window-center", Title: "Center Window",
			Subtitle: "Center window on screen", Icon: "Target", Category: "system",
		},
	}
}
