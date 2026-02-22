package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// AISettings holds AI provider configuration.
type AISettings struct {
	Provider                string `json:"provider"` // "openai" | "anthropic" | "ollama" | "openai-compatible"
	OpenAIApiKey            string `json:"openaiApiKey"`
	AnthropicApiKey         string `json:"anthropicApiKey"`
	OllamaBaseUrl           string `json:"ollamaBaseUrl"`
	DefaultModel            string `json:"defaultModel"`
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
	FontSize         string            `json:"fontSize"` // "small" | "medium" | "large"
	BaseColor        string            `json:"baseColor"`
	AI               AISettings        `json:"ai"`
}

var defaultAI = AISettings{
	Provider:      "openai",
	OllamaBaseUrl: "http://localhost:11434",
	DefaultModel:  "gpt-4o-mini",
	Enabled:       true,
}

var defaultSettings = AppSettings{
	GlobalShortcut:   "Alt+Space",
	PinnedCommands:   []string{},
	RecentCommands:   []string{},
	DisabledCommands: []string{},
	CommandHotkeys:   map[string]string{},
	FontSize:         "medium",
	BaseColor:        "#101113",
	AI:               defaultAI,
}

// Service manages app settings.
type Service struct {
	mu    sync.RWMutex
	cache *AppSettings
	path  string
}

// NewService creates a new settings service, storing settings in the given directory.
func NewService(dataDir string) *Service {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// best effort
	}
	return &Service{
		path: filepath.Join(dataDir, "settings.json"),
	}
}

// GetSettings returns the current settings.
func (s *Service) GetSettings() (*AppSettings, error) {
	s.mu.RLock()
	if s.cache != nil {
		cp := *s.cache
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
func (s *Service) SaveSettings(patch map[string]interface{}) (*AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.load()

	// Marshal current to map, apply patch, unmarshal back
	data, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	for k, v := range patch {
		m[k] = v
	}
	merged, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var updated AppSettings
	if err := json.Unmarshal(merged, &updated); err != nil {
		return nil, err
	}

	// Persist
	if out, err := json.MarshalIndent(updated, "", "  "); err == nil {
		_ = os.WriteFile(s.path, out, 0644)
	}

	s.cache = &updated
	cp := updated
	return &cp, nil
}

// ResetCache forces a reload from disk on next access.
func (s *Service) ResetCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = nil
}

func (s *Service) load() *AppSettings {
	data, err := os.ReadFile(s.path)
	if err != nil {
		cp := defaultSettings
		return &cp
	}
	var parsed AppSettings
	if err := json.Unmarshal(data, &parsed); err != nil {
		cp := defaultSettings
		return &cp
	}
	// Fill in zero-value fields with defaults
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
	if parsed.AI.OllamaBaseUrl == "" {
		parsed.AI.OllamaBaseUrl = defaultAI.OllamaBaseUrl
	}
	if parsed.AI.DefaultModel == "" {
		parsed.AI.DefaultModel = defaultAI.DefaultModel
	}
	return &parsed
}
