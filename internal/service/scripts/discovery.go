package scripts

import (
	"os"
	"path/filepath"

	"github.com/dyike/Void/internal/model"
)

// ─── Discovery ────────────────────────────────────────────────────────────

func (s *Service) scanDir(dir string) []model.ScriptCommand {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []model.ScriptCommand
	for _, entry := range entries {
		if entry.IsDir() {
			// Recurse into subdirectories (packages)
			results = append(results, s.scanDir(filepath.Join(dir, entry.Name()))...)
			continue
		}

		name := entry.Name()
		if !isScriptFile(name) {
			continue
		}

		fullPath := filepath.Join(dir, name)
		cmd := parseScriptMetadata(fullPath)
		if cmd == nil {
			continue
		}
		results = append(results, *cmd)
	}
	return results
}

// ─── Example scripts ──────────────────────────────────────────────────────

func (s *Service) createExampleScripts() {
	examples := []struct {
		name    string
		content string
	}{
		{
			"whoami.sh",
			`#!/bin/bash
# @raycast.schemaVersion 1
# @raycast.title Who am I
# @raycast.mode compact
# @raycast.icon User
# @raycast.description Show current user and hostname

echo "$(whoami)@$(hostname)"
`,
		},
		{
			"ip-address.sh",
			`#!/bin/bash
# @raycast.schemaVersion 1
# @raycast.title My IP Address
# @raycast.mode compact
# @raycast.icon Globe
# @raycast.description Show your local IP address

ipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || echo "No network"
`,
		},
		{
			"git-status.sh",
			`#!/bin/bash
# @raycast.schemaVersion 1
# @raycast.title Git Status
# @raycast.mode fullOutput
# @raycast.icon Terminal
# @raycast.description Show git status of current directory

git -C "$HOME" status --short 2>/dev/null || echo "Not in a git repository"
`,
		},
		{
			"clock.sh",
			`#!/bin/bash
# @raycast.schemaVersion 1
# @raycast.title Current Time
# @raycast.mode compact
# @raycast.icon Clock
# @raycast.description Show current date and time

date "+%Y-%m-%d %H:%M:%S %Z"
`,
		},
	}

	for _, ex := range examples {
		path := filepath.Join(s.scriptsDir, ex.name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			_ = os.WriteFile(path, []byte(ex.content), 0755)
		}
	}
}
