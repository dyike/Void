package service

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dyike/Void/internal/autostart"
)

// AISettings holds AI provider configuration.
type AISettings struct {
	Provider                string `json:"provider"`
	OpenAIApiKey            string `json:"openaiApiKey"`
	AnthropicApiKey         string `json:"anthropicApiKey"`
	OllamaBaseUrl           string `json:"ollamaBaseUrl"`
	DefaultModel            string `json:"defaultModel"`
	DeepSeekApiKey          string `json:"deepseekApiKey"`
	OpenAICompatibleBaseUrl string `json:"openaiCompatibleBaseUrl"`
	OpenAICompatibleApiKey  string `json:"openaiCompatibleApiKey"`
	OpenAICompatibleModel   string `json:"openaiCompatibleModel"`
	Enabled                 bool   `json:"enabled"`
}

// AppSettings is the persisted application configuration.
type AppSettings struct {
	GlobalShortcut   string            `json:"globalShortcut"`
	PinnedCommands   []string          `json:"pinnedCommands"`
	RecentCommands   []string          `json:"recentCommands"`
	DisabledCommands []string          `json:"disabledCommands"`
	CommandHotkeys   map[string]string `json:"commandHotkeys"`
	FontSize         string            `json:"fontSize"`
	BaseColor        string            `json:"baseColor"`
	TerminalApp      string            `json:"terminalApp"`
	SkillsPath       string            `json:"skillsPath"`
	ThemeMode        string            `json:"themeMode"`
	LaunchAtLogin    bool              `json:"launchAtLogin"`
	ShowInMenuBar    bool              `json:"showInMenuBar"`
	AI               AISettings        `json:"ai"`
}

var defaultAI = AISettings{
	Provider:      "openai",
	OllamaBaseUrl: "http://localhost:11434",
	DefaultModel:  "gpt-4o-mini",
	Enabled:       true,
}

var defaultSettings = AppSettings{
	GlobalShortcut:   "option+space",
	PinnedCommands:   []string{},
	RecentCommands:   []string{},
	DisabledCommands: []string{},
	CommandHotkeys:   map[string]string{},
	FontSize:         "medium",
	BaseColor:        "#101113",
	TerminalApp:      "Terminal",
	SkillsPath:       "",
	ThemeMode:        "system",
	LaunchAtLogin:    false,
	ShowInMenuBar:    true,
	AI:               defaultAI,
}

// SettingsService manages app settings.
type SettingsService struct {
	mu                   sync.RWMutex
	cache                *AppSettings
	path                 string
	openSettingsFunc     func()
	resizeSettingsFunc   func(width, height int)
	showSettingsFunc     func()
	toggleSystemTrayFunc func(show bool)
}

// NewSettingsService creates a new settings service.
func NewSettingsService(dataDir string) *SettingsService {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// best effort
	}
	return &SettingsService{
		path: filepath.Join(dataDir, "settings.json"),
	}
}

func (s *SettingsService) SetOpenSettingsFunc(fn func()) {
	s.openSettingsFunc = fn
}

func (s *SettingsService) SetResizeSettingsFunc(fn func(width, height int)) {
	s.resizeSettingsFunc = fn
}

func (s *SettingsService) SetToggleSystemTrayFunc(fn func(show bool)) {
	s.toggleSystemTrayFunc = fn
}

func (s *SettingsService) SetShowSettingsFunc(fn func()) {
	s.showSettingsFunc = fn
}

func (s *SettingsService) NotifySettingsReady() {
	if s.showSettingsFunc != nil {
		s.showSettingsFunc()
	}
}

func (s *SettingsService) OpenSettingsWindow() {
	if s.openSettingsFunc != nil {
		s.openSettingsFunc()
	}
}

func (s *SettingsService) ResizeSettingsWindow(width, height int) {
	if s.resizeSettingsFunc != nil {
		s.resizeSettingsFunc(width, height)
	}
}

// GetSettings returns the current settings.
func (s *SettingsService) GetSettings() (*AppSettings, error) {
	s.mu.RLock()
	if s.cache != nil {
		cp := *s.cache
		s.mu.RUnlock()
		return &cp, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	settings := s.load()
	s.cache = settings
	cp := *settings
	return &cp, nil
}

// SaveSettings merges a patch into the current settings and persists.
func (s *SettingsService) SaveSettings(updated AppSettings) (*AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if updated.GlobalShortcut == "" {
		updated.GlobalShortcut = defaultSettings.GlobalShortcut
	}
	if updated.FontSize == "" {
		updated.FontSize = defaultSettings.FontSize
	}
	if updated.BaseColor == "" {
		updated.BaseColor = defaultSettings.BaseColor
	}
	if updated.ThemeMode == "" {
		updated.ThemeMode = defaultSettings.ThemeMode
	}
	if updated.AI.OllamaBaseUrl == "" {
		updated.AI.OllamaBaseUrl = defaultAI.OllamaBaseUrl
	}
	if updated.AI.DefaultModel == "" {
		updated.AI.DefaultModel = defaultAI.DefaultModel
	}

	if out, err := json.MarshalIndent(updated, "", "  "); err == nil {
		if err := os.WriteFile(s.path, out, 0644); err != nil {
			log.Printf("settings: failed to write file: %v", err)
		} else {
			log.Printf("settings: saved successfully to %s", s.path)
		}
	} else {
		log.Printf("settings: failed to marshal for persist: %v", err)
	}

	if updated.LaunchAtLogin {
		_ = autostart.Enable()
	} else {
		_ = autostart.Disable()
	}

	if s.toggleSystemTrayFunc != nil {
		s.toggleSystemTrayFunc(updated.ShowInMenuBar)
	}

	s.cache = &updated
	cp := updated
	return &cp, nil
}

// SetGlobalHotkey updates only the GlobalShortcut setting.
func (s *SettingsService) SetGlobalHotkey(spec string) error {
	s.mu.RLock()
	current := s.cache
	if current == nil {
		current = s.load()
	}
	s.mu.RUnlock()

	cp := *current
	cp.GlobalShortcut = spec
	_, err := s.SaveSettings(cp)
	return err
}

// GetGlobalHotkey returns the persisted global hotkey spec string.
func (s *SettingsService) GetGlobalHotkey() string {
	st, err := s.GetSettings()
	if err != nil || st.GlobalShortcut == "" {
		return "option+space"
	}
	legacy := map[string]string{
		"Alt+Space": "option+space",
		"Cmd+Space": "cmd+space",
	}
	if v, ok := legacy[st.GlobalShortcut]; ok {
		return v
	}
	return st.GlobalShortcut
}

// ResetCache forces a reload from disk on next access.
func (s *SettingsService) ResetCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = nil
}

func (s *SettingsService) load() *AppSettings {
	data, err := os.ReadFile(s.path)
	if err != nil {
		cp := defaultSettings
		return &cp
	}
	parsed := defaultSettings
	if err := json.Unmarshal(data, &parsed); err != nil {
		cp := defaultSettings
		return &cp
	}
	if parsed.GlobalShortcut == "" {
		parsed.GlobalShortcut = defaultSettings.GlobalShortcut
	}
	if parsed.FontSize == "" {
		parsed.FontSize = defaultSettings.FontSize
	}
	if parsed.BaseColor == "" {
		parsed.BaseColor = defaultSettings.BaseColor
	}
	if parsed.PinnedCommands == nil {
		parsed.PinnedCommands = []string{}
	}
	if parsed.RecentCommands == nil {
		parsed.RecentCommands = []string{}
	}
	if parsed.DisabledCommands == nil {
		parsed.DisabledCommands = []string{}
	}
	if parsed.CommandHotkeys == nil {
		parsed.CommandHotkeys = map[string]string{}
	}
	if parsed.ThemeMode == "" {
		parsed.ThemeMode = defaultSettings.ThemeMode
	}
	if parsed.AI.OllamaBaseUrl == "" {
		parsed.AI.OllamaBaseUrl = defaultAI.OllamaBaseUrl
	}
	if parsed.AI.DefaultModel == "" {
		parsed.AI.DefaultModel = defaultAI.DefaultModel
	}
	return &parsed
}

// Skill represents a found skill directory.
type Skill struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

// GetSkills scans the configured SkillsPath for directories and returns them.
func (s *SettingsService) GetSkills() ([]Skill, error) {
	st, err := s.GetSettings()
	if err != nil {
		return nil, err
	}
	if st.SkillsPath == "" {
		return []Skill{}, nil
	}

	skillsPath := st.SkillsPath
	if strings.HasPrefix(skillsPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			skillsPath = filepath.Join(home, skillsPath[2:])
		}
	}

	entries, err := os.ReadDir(skillsPath)
	if err != nil {
		return []Skill{}, nil
	}

	var skills []Skill
	for _, entry := range entries {
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			skillPath := filepath.Join(skillsPath, entry.Name())
			desc := ""
			mdPath := filepath.Join(skillPath, "SKILL.md")
			mdData, err := os.ReadFile(mdPath)
			if err == nil {
				lines := strings.Split(string(mdData), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "description:") {
						desc = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
						break
					}
				}
			}
			skills = append(skills, Skill{
				Name:        entry.Name(),
				Path:        skillPath,
				Description: desc,
			})
		}
	}
	return skills, nil
}
