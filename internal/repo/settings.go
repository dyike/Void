package repo

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"github.com/dyike/Void/internal/model"
)

var DefaultSettings = model.AppSettings{
	WindowMaterial:   "frosted",
	GlobalShortcut:   "option+space",
	PinnedCommands:   []string{},
	RecentCommands:   []string{},
	DisabledCommands: []string{},
	CommandHotkeys:   map[string]string{},
	FontSize:         "medium",
	BaseColor:        "#101113",
	TerminalApp:      "Terminal",
	ThemeMode:        "system",
	LaunchAtLogin:    false,
	ShowInMenuBar:    true,
}

// SettingsRepo handles settings persistence.
type SettingsRepo struct {
	path string
}

// NewSettingsRepo creates a new settings repo.
func NewSettingsRepo(dataDir string) *SettingsRepo {
	_ = os.MkdirAll(dataDir, 0755)
	return &SettingsRepo{path: filepath.Join(dataDir, "settings.json")}
}

// Load reads settings from disk, merging missing fields with defaults.
func (r *SettingsRepo) Load() *model.AppSettings {
	data, err := os.ReadFile(r.path)
	if err != nil {
		cp := DefaultSettings
		return &cp
	}
	if cleaned, changed, err := removeRetiredAISettings(data); err == nil && changed {
		if err := os.WriteFile(r.path, cleaned, 0600); err != nil {
			log.Printf("settings: failed to remove retired configuration: %v", err)
		}
		data = cleaned
	}
	parsed := DefaultSettings
	if err := json.Unmarshal(data, &parsed); err != nil {
		cp := DefaultSettings
		return &cp
	}
	r.applyDefaults(&parsed)
	return &parsed
}

// Remove retired configuration without rewriting unrelated settings or deleting
// files in the user's former skills directory.
func removeRetiredAISettings(data []byte) ([]byte, bool, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, false, err
	}
	changed := false
	for _, key := range []string{"ai", "skillsPath"} {
		if _, exists := fields[key]; exists {
			delete(fields, key)
			changed = true
		}
	}
	const retiredCommand = "system-ai-chat"
	for _, key := range []string{"pinnedCommands", "recentCommands", "disabledCommands"} {
		if raw, exists := fields[key]; exists {
			var commands []string
			if err := json.Unmarshal(raw, &commands); err != nil {
				return nil, false, err
			}
			kept := make([]string, 0, len(commands))
			for _, command := range commands {
				if command != retiredCommand {
					kept = append(kept, command)
				}
			}
			if len(kept) != len(commands) {
				fields[key], _ = json.Marshal(kept)
				changed = true
			}
		}
	}
	if raw, exists := fields["commandHotkeys"]; exists {
		var hotkeys map[string]string
		if err := json.Unmarshal(raw, &hotkeys); err != nil {
			return nil, false, err
		}
		if _, exists := hotkeys[retiredCommand]; exists {
			delete(hotkeys, retiredCommand)
			fields["commandHotkeys"], _ = json.Marshal(hotkeys)
			changed = true
		}
	}
	if !changed {
		return data, false, nil
	}
	out, err := json.MarshalIndent(fields, "", "  ")
	return out, true, err
}

// Save persists settings to disk.
func (r *SettingsRepo) Save(settings model.AppSettings) error {
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		log.Printf("settings: failed to marshal: %v", err)
		return err
	}
	if err := os.WriteFile(r.path, out, 0644); err != nil {
		log.Printf("settings: failed to write file: %v", err)
		return err
	}
	log.Printf("settings: saved successfully to %s", r.path)
	return nil
}

func (r *SettingsRepo) applyDefaults(s *model.AppSettings) {
	switch s.WindowMaterial {
	case "clear", "regular", "frosted":
	default:
		s.WindowMaterial = DefaultSettings.WindowMaterial
	}
	if s.GlobalShortcut == "" {
		s.GlobalShortcut = DefaultSettings.GlobalShortcut
	}
	if s.FontSize == "" {
		s.FontSize = DefaultSettings.FontSize
	}
	if s.BaseColor == "" {
		s.BaseColor = DefaultSettings.BaseColor
	}
	if s.ThemeMode == "" {
		s.ThemeMode = DefaultSettings.ThemeMode
	}
	if s.PinnedCommands == nil {
		s.PinnedCommands = []string{}
	}
	if s.RecentCommands == nil {
		s.RecentCommands = []string{}
	}
	if s.DisabledCommands == nil {
		s.DisabledCommands = []string{}
	}
	if s.CommandHotkeys == nil {
		s.CommandHotkeys = map[string]string{}
	}
}
