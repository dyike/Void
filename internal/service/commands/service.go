package commands

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/service/frecency"
	"github.com/dyike/Void/internal/service/settings"
)

// Service discovers and caches macOS commands.
type Service struct {
	mu           sync.RWMutex
	cache        []model.CommandInfo
	cacheTime    time.Time
	iconCacheDir string
	cacheTTL     time.Duration
	frecency     *frecency.Store
	settings     *settings.Service
}

// NewService creates a new commands service.
func NewService(dataDir string, settingsSvc *settings.Service) *Service {
	iconCacheDir := filepath.Join(dataDir, "icon-cache")
	_ = os.MkdirAll(iconCacheDir, 0755)
	return &Service{
		iconCacheDir: iconCacheDir,
		cacheTTL:     20 * time.Minute,
		frecency:     frecency.NewStore(dataDir),
		settings:     settingsSvc,
	}
}

// GetCommands returns cached + frecency-boosted commands.
func (s *Service) GetCommands() ([]model.CommandInfo, error) {
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
func (s *Service) RefreshCommands() ([]model.CommandInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var all []model.CommandInfo
	all = append(all, systemCommands()...)

	apps := s.discoverAppsSpotlight()
	if len(apps) == 0 {
		log.Println("Spotlight found no apps, falling back to directory scan.")
		apps, _ = s.discoverAppsDir()
	}
	all = append(all, apps...)

	panes := s.discoverSettingsPanes()
	all = append(all, panes...)

	s.cache = make([]model.CommandInfo, len(all))
	copy(s.cache, all)
	s.cacheTime = time.Now()

	log.Printf("commands: refreshed %d items (apps=%d settings=%d builtins=%d)\n",
		len(all), len(apps), len(panes), len(systemCommands()))

	return s.withFrecency(all), nil
}

// RecordUsage records that a command was used (for frecency).
func (s *Service) RecordUsage(id string) {
	s.frecency.RecordAccess(id)
}
