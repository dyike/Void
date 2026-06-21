package service

import (
	"sync"

	"github.com/dyike/Void/internal/autostart"
	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/repo"
)

// SettingsService manages app settings with in-memory cache and window callbacks.
type SettingsService struct {
	mu                   sync.RWMutex
	cache                *model.AppSettings
	repo                 *repo.SettingsRepo
	openSettingsFunc     func()
	resizeSettingsFunc   func(width, height int)
	resizeMainFunc       func(width, height int)
	showSettingsFunc     func()
	toggleSystemTrayFunc func(show bool)
}

// NewSettingsService creates a new settings service.
func NewSettingsService(r *repo.SettingsRepo) *SettingsService {
	return &SettingsService{repo: r}
}

func (s *SettingsService) SetOpenSettingsFunc(fn func()) {
	s.openSettingsFunc = fn
}

func (s *SettingsService) SetResizeSettingsFunc(fn func(width, height int)) {
	s.resizeSettingsFunc = fn
}

func (s *SettingsService) SetResizeMainFunc(fn func(width, height int)) {
	s.resizeMainFunc = fn
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

func (s *SettingsService) ResizeMainWindow(width, height int) {
	if s.resizeMainFunc != nil {
		s.resizeMainFunc(width, height)
	}
}

// GetSettings returns the current settings, using cache when available.
func (s *SettingsService) GetSettings() (*model.AppSettings, error) {
	s.mu.RLock()
	if s.cache != nil {
		cp := *s.cache
		s.mu.RUnlock()
		return &cp, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	settings := s.repo.Load()
	s.cache = settings
	cp := *settings
	return &cp, nil
}

// SaveSettings persists settings and triggers side effects.
func (s *SettingsService) SaveSettings(updated model.AppSettings) (*model.AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Apply defaults for blank fields
	if updated.GlobalShortcut == "" {
		updated.GlobalShortcut = repo.DefaultSettings.GlobalShortcut
	}
	if updated.FontSize == "" {
		updated.FontSize = repo.DefaultSettings.FontSize
	}
	if updated.BaseColor == "" {
		updated.BaseColor = repo.DefaultSettings.BaseColor
	}
	if updated.ThemeMode == "" {
		updated.ThemeMode = repo.DefaultSettings.ThemeMode
	}
	if updated.AI.OllamaBaseUrl == "" {
		updated.AI.OllamaBaseUrl = repo.DefaultAI.OllamaBaseUrl
	}
	if updated.AI.DefaultModel == "" {
		updated.AI.DefaultModel = repo.DefaultAI.DefaultModel
	}

	_ = s.repo.Save(updated)

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
		current = s.repo.Load()
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

// GetSkills returns skills discovered at the configured SkillsPath.
func (s *SettingsService) GetSkills() ([]model.Skill, error) {
	st, err := s.GetSettings()
	if err != nil {
		return nil, err
	}
	return s.repo.GetSkills(st.SkillsPath)
}

// ResetCache forces a reload from disk on next access.
func (s *SettingsService) ResetCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = nil
}
