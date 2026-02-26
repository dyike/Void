package model

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

// Skill represents a found skill directory.
type Skill struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
}
