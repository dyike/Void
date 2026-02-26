package service

import (
	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/repo"
)

// ScriptsService discovers and runs script commands.
type ScriptsService struct {
	repo *repo.ScriptsRepo
}

// NewScriptsService creates a new scripts service.
func NewScriptsService(r *repo.ScriptsRepo) *ScriptsService {
	return &ScriptsService{repo: r}
}

// GetScriptCommands returns all discovered script commands.
func (s *ScriptsService) GetScriptCommands() []model.ScriptCommand {
	return s.repo.GetAll()
}

// RefreshScriptCommands rescans the scripts directory.
func (s *ScriptsService) RefreshScriptCommands() []model.ScriptCommand {
	return s.repo.Refresh()
}

// GetScriptsDir returns the directory where scripts live.
func (s *ScriptsService) GetScriptsDir() string {
	return s.repo.GetDir()
}

// OpenScriptsDir opens the scripts directory in Finder.
func (s *ScriptsService) OpenScriptsDir() error {
	return s.repo.OpenDir()
}

// CreateScript creates a new script file from a template.
func (s *ScriptsService) CreateScript(name, mode string) (string, error) {
	return s.repo.CreateScript(name, mode)
}

// Execute runs a script command and returns the output.
func (s *ScriptsService) Execute(id string, args map[string]string) (*model.ExecutionResult, error) {
	return s.repo.Execute(id, args)
}
