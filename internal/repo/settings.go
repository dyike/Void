package repo

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dyike/Void/internal/model"
)

var DefaultAI = model.AISettings{
	Provider:      "openai",
	OllamaBaseUrl: "http://localhost:11434",
	DefaultModel:  "gpt-4o-mini",
	Enabled:       true,
}

var DefaultSettings = model.AppSettings{
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
	AI:               DefaultAI,
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
	parsed := DefaultSettings
	if err := json.Unmarshal(data, &parsed); err != nil {
		cp := DefaultSettings
		return &cp
	}
	r.applyDefaults(&parsed)
	return &parsed
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

// GetSkills scans skillsPath for subdirectories and returns them as Skill list.
func (r *SettingsRepo) GetSkills(skillsPath string) ([]model.Skill, error) {
	if skillsPath == "" {
		return []model.Skill{}, nil
	}

	if strings.HasPrefix(skillsPath, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			skillsPath = filepath.Join(home, skillsPath[2:])
		}
	}

	entries, err := os.ReadDir(skillsPath)
	if err != nil {
		return []model.Skill{}, nil
	}

	var skills []model.Skill
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		skillPath := filepath.Join(skillsPath, entry.Name())
		desc := ""
		if mdData, err := os.ReadFile(filepath.Join(skillPath, "SKILL.md")); err == nil {
			for _, line := range strings.Split(string(mdData), "\n") {
				if strings.HasPrefix(line, "description:") {
					desc = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
					break
				}
			}
		}
		skills = append(skills, model.Skill{
			Name:        entry.Name(),
			Path:        skillPath,
			Description: desc,
		})
	}
	return skills, nil
}

func (r *SettingsRepo) applyDefaults(s *model.AppSettings) {
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
	if s.AI.OllamaBaseUrl == "" {
		s.AI.OllamaBaseUrl = DefaultAI.OllamaBaseUrl
	}
	if s.AI.DefaultModel == "" {
		s.AI.DefaultModel = DefaultAI.DefaultModel
	}
}
