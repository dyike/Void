package scripts

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ScriptMode controls how the output is displayed.
type ScriptMode string

const (
	ModeSilent     ScriptMode = "silent"     // run and ignore output
	ModeCompact    ScriptMode = "compact"    // show last line in HUD
	ModeFullOutput ScriptMode = "fullOutput" // show full output in results view
	ModeInline     ScriptMode = "inline"     // replaces the command item with output
)

// Argument defines a script input parameter.
type Argument struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "text"|"password"|"dropdown"
	Placeholder string `json:"placeholder"`
	Required    bool   `json:"required"`
}

// ScriptCommand is a discovered shell script with metadata.
type ScriptCommand struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Mode        ScriptMode `json:"mode"`
	Icon        string     `json:"icon,omitempty"`
	ScriptPath  string     `json:"scriptPath"`
	Arguments   []Argument `json:"arguments,omitempty"`
	RefreshTime string     `json:"refreshTime,omitempty"`
	PackageName string     `json:"packageName,omitempty"`
}

// ExecutionResult holds the output of a script run.
type ExecutionResult struct {
	CommandID string     `json:"commandId"`
	Title     string     `json:"title"`
	Mode      ScriptMode `json:"mode"`
	ExitCode  int        `json:"exitCode"`
	Stdout    string     `json:"stdout"`
	Stderr    string     `json:"stderr"`
	Output    string     `json:"output"`
	FirstLine string     `json:"firstLine"`
	LastLine  string     `json:"lastLine"`
}

// Service discovers and runs script commands.
type Service struct {
	mu         sync.RWMutex
	scriptsDir string
	cache      []ScriptCommand
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
func (s *Service) GetScriptCommands() []ScriptCommand {
	s.mu.RLock()
	if s.cache != nil && time.Since(s.cacheTime) < s.cacheTTL {
		result := make([]ScriptCommand, len(s.cache))
		copy(result, s.cache)
		s.mu.RUnlock()
		return result
	}
	s.mu.RUnlock()

	return s.RefreshScriptCommands()
}

// RefreshScriptCommands rescans the scripts directory.
func (s *Service) RefreshScriptCommands() []ScriptCommand {
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

// Execute runs a script command and returns the output.
func (s *Service) Execute(id string, args map[string]string) (*ExecutionResult, error) {
	cmds := s.GetScriptCommands()

	var target *ScriptCommand
	for i := range cmds {
		if cmds[i].ID == id {
			target = &cmds[i]
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("script command %q not found", id)
	}

	return s.runScript(target, args)
}

// ─── Discovery ────────────────────────────────────────────────────────────

func (s *Service) scanDir(dir string) []ScriptCommand {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []ScriptCommand
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

// parseScriptMetadata reads @raycast.xxx metadata from the script header.
func parseScriptMetadata(path string) *ScriptCommand {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	meta := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Stop after the metadata block (past first non-comment line)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "#") {
			break
		}
		// Parse @raycast.key value
		line = strings.TrimPrefix(trimmed, "#")
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "@raycast.") {
			continue
		}
		line = strings.TrimPrefix(line, "@raycast.")
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			meta[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	title := meta["title"]
	if title == "" {
		// Use filename as fallback
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	schemaVersion := meta["schemaVersion"]
	_ = schemaVersion // We accept any version

	mode := ScriptMode(meta["mode"])
	if mode == "" {
		mode = ModeFullOutput
	}

	id := "script-" + slugify(title)
	cmd := &ScriptCommand{
		ID:          id,
		Title:       title,
		Description: meta["description"],
		Mode:        mode,
		Icon:        meta["icon"],
		ScriptPath:  path,
		PackageName: meta["packageName"],
		RefreshTime: meta["refreshTime"],
	}
	if cmd.Icon == "" {
		cmd.Icon = "FileCode"
	}

	// Parse argument definitions: @raycast.argument1, @raycast.argument2, ...
	for i := 1; i <= 5; i++ {
		raw := meta[fmt.Sprintf("argument%d", i)]
		if raw == "" {
			break
		}
		// Format: {"type": "text", "placeholder": "Query", "required": true}
		var arg Argument
		if err := json.Unmarshal([]byte(raw), &arg); err != nil {
			// Fallback: treat whole value as placeholder
			arg = Argument{Name: fmt.Sprintf("arg%d", i), Type: "text", Placeholder: raw, Required: false}
		}
		if arg.Name == "" {
			arg.Name = fmt.Sprintf("arg%d", i)
		}
		cmd.Arguments = append(cmd.Arguments, arg)
	}

	return cmd
}

// ─── Execution ────────────────────────────────────────────────────────────

func (s *Service) runScript(cmd *ScriptCommand, args map[string]string) (*ExecutionResult, error) {
	// Build arg list in order
	argv := []string{cmd.ScriptPath}
	for i := 1; i <= len(cmd.Arguments); i++ {
		argDef := cmd.Arguments[i-1]
		val := args[argDef.Name]
		if val == "" {
			val = args[fmt.Sprintf("arg%d", i)]
		}
		argv = append(argv, val)
	}

	c := exec.Command(argv[0], argv[1:]...)
	c.Dir = filepath.Dir(cmd.ScriptPath)

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	exitCode := 0
	if err := c.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, err
		}
	}

	stdoutStr := strings.TrimRight(stdout.String(), "\n")
	stderrStr := strings.TrimRight(stderr.String(), "\n")
	output := stdoutStr
	if output == "" {
		output = stderrStr
	}

	lines := strings.Split(output, "\n")
	first, last := "", ""
	if len(lines) > 0 {
		first = strings.TrimSpace(lines[0])
		last = strings.TrimSpace(lines[len(lines)-1])
	}

	return &ExecutionResult{
		CommandID: cmd.ID,
		Title:     cmd.Title,
		Mode:      cmd.Mode,
		ExitCode:  exitCode,
		Stdout:    stdoutStr,
		Stderr:    stderrStr,
		Output:    output,
		FirstLine: first,
		LastLine:  last,
	}, nil
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

// ─── Helpers ─────────────────────────────────────────────────────────────

func isScriptFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".sh" || ext == ".py" || ext == ".rb" || ext == ".js" ||
		ext == ".ts" || ext == ".zsh" || ext == ".fish" || ext == ".swift"
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return result
}
