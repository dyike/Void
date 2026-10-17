package model

// AppSettings is the persisted application configuration.
type AppSettings struct {
	WindowMaterial   string            `json:"windowMaterial"`
	GlobalShortcut   string            `json:"globalShortcut"`
	PinnedCommands   []string          `json:"pinnedCommands"`
	RecentCommands   []string          `json:"recentCommands"`
	DisabledCommands []string          `json:"disabledCommands"`
	CommandHotkeys   map[string]string `json:"commandHotkeys"`
	FontSize         string            `json:"fontSize"`
	BaseColor        string            `json:"baseColor"`
	TerminalApp      string            `json:"terminalApp"`
	ThemeMode        string            `json:"themeMode"`
	LaunchAtLogin    bool              `json:"launchAtLogin"`
	ShowInMenuBar    bool              `json:"showInMenuBar"`
}
