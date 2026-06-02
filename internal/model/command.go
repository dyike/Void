package model

// CommandInfo represents a launchable command.
type CommandInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	// Base64 PNG data URL or empty
	IconDataURL string `json:"iconDataUrl,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Category    string `json:"category"` // "app" | "settings" | "system" | "snippet" | "file"
	// App path for apps, bundle ID for settings
	Path  string  `json:"path,omitempty"`
	Score float64 `json:"score,omitempty"`
}
