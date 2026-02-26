package scripts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/dyike/Void/internal/model"
)

// Service discovers and runs script commands.
type Service struct {
	mu         sync.RWMutex
	scriptsDir string
	cache      []model.ScriptCommand
	cacheTime  time.Time
	cacheTTL   time.Duration
}

// NewService creates a script command service.
func NewService(dataDir string) *Service {
	scriptsDir := filepath.Join(dataDir, "script-commands")
	_ = os.MkdirAll(scriptsDir, 0755)

	svc := &Service{
		scriptsDir: scriptsDir,
		cacheTTL:   12 * time.Second,
	}

	// Create example scripts on first run
	svc.createExampleScripts()
	return svc
}

// GetScriptCommands returns all discovered script commands.
func (s *Service) GetScriptCommands() []model.ScriptCommand {
	s.mu.RLock()
	if s.cache != nil && time.Since(s.cacheTime) < s.cacheTTL {
		result := make([]model.ScriptCommand, len(s.cache))
		copy(result, s.cache)
		s.mu.RUnlock()
		return result
	}
	s.mu.RUnlock()

	return s.RefreshScriptCommands()
}

// RefreshScriptCommands rescans the scripts directory.
func (s *Service) RefreshScriptCommands() []model.ScriptCommand {
	commands := s.scanDir(s.scriptsDir)

	s.mu.Lock()
	s.cache = commands
	s.cacheTime = time.Now()
	s.mu.Unlock()

	return commands
}

// GetScriptsDir returns the directory where scripts live.
func (s *Service) GetScriptsDir() string {
	return s.scriptsDir
}

// OpenScriptsDir opens the scripts directory in Finder.
func (s *Service) OpenScriptsDir() error {
	return exec.Command("open", s.scriptsDir).Run()
}

// CreateScript creates a new script file from a template.
func (s *Service) CreateScript(name, mode string) (string, error) {
	slug := slugify(name)
	path := filepath.Join(s.scriptsDir, slug+".sh")

	content := fmt.Sprintf(`#!/bin/bash
# Required parameters:
# @raycast.schemaVersion 1
# @raycast.title %s
# @raycast.mode %s

# Optional parameters:
# @raycast.icon Rocket
# @raycast.packageName Custom Scripts
# @raycast.description A script command

# ─── Script body ─────────────────────────────────────────────────
echo "Hello from %s"
`, name, mode, name)

	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		return "", err
	}
	return path, nil
}
