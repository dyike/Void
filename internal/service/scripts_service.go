package service

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

	"github.com/dyike/Void/internal/model"
)

// ScriptsService discovers and runs script commands.
type ScriptsService struct {
	mu         sync.RWMutex
	scriptsDir string
	cache      []model.ScriptCommand
	cacheTime  time.Time
	cacheTTL   time.Duration
}

// NewScriptsService creates a script command service.
func NewScriptsService(dataDir string) *ScriptsService {
	scriptsDir := filepath.Join(dataDir, "script-commands")
	_ = os.MkdirAll(scriptsDir, 0755)

	svc := &ScriptsService{
		scriptsDir: scriptsDir,
		cacheTTL:   12 * time.Second,
	}
	svc.createExampleScripts()
	return svc
}

// GetScriptCommands returns all discovered script commands.
func (s *ScriptsService) GetScriptCommands() []model.ScriptCommand {
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
func (s *ScriptsService) RefreshScriptCommands() []model.ScriptCommand {
	commands := s.scanDir(s.scriptsDir)
	s.mu.Lock()
	s.cache = commands
	s.cacheTime = time.Now()
	s.mu.Unlock()
	return commands
}

// GetScriptsDir returns the directory where scripts live.
func (s *ScriptsService) GetScriptsDir() string {
	return s.scriptsDir
}

// OpenScriptsDir opens the scripts directory in Finder.
func (s *ScriptsService) OpenScriptsDir() error {
	return exec.Command("open", s.scriptsDir).Run()
}

// CreateScript creates a new script file from a template.
func (s *ScriptsService) CreateScript(name, mode string) (string, error) {
	slug := scriptSlugify(name)
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
func (s *ScriptsService) Execute(id string, args map[string]string) (*model.ExecutionResult, error) {
	cmds := s.GetScriptCommands()

	var target *model.ScriptCommand
	for i := range cmds {
		if cmds[i].ID == id {
			target = &cmds[i]
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("script not found: %s", id)
	}
	return s.runScript(target, args)
}

func (s *ScriptsService) runScript(cmd *model.ScriptCommand, args map[string]string) (*model.ExecutionResult, error) {
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

	return &model.ExecutionResult{
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

// ─── Discovery ────────────────────────────────────────────────────────────

func (s *ScriptsService) scanDir(dir string) []model.ScriptCommand {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var results []model.ScriptCommand
	for _, entry := range entries {
		if entry.IsDir() {
			results = append(results, s.scanDir(filepath.Join(dir, entry.Name()))...)
			continue
		}
		if !isScriptFile(entry.Name()) {
			continue
		}
		cmd := parseScriptMetadata(filepath.Join(dir, entry.Name()))
		if cmd == nil {
			continue
		}
		results = append(results, *cmd)
	}
	return results
}

func (s *ScriptsService) createExampleScripts() {
	examples := []struct {
		name    string
		content string
	}{
		{"whoami.sh", "#!/bin/bash\n# @raycast.schemaVersion 1\n# @raycast.title Who am I\n# @raycast.mode compact\n# @raycast.icon User\n# @raycast.description Show current user and hostname\n\necho \"$(whoami)@$(hostname)\"\n"},
		{"ip-address.sh", "#!/bin/bash\n# @raycast.schemaVersion 1\n# @raycast.title My IP Address\n# @raycast.mode compact\n# @raycast.icon Globe\n# @raycast.description Show your local IP address\n\nipconfig getifaddr en0 2>/dev/null || ipconfig getifaddr en1 2>/dev/null || echo \"No network\"\n"},
		{"git-status.sh", "#!/bin/bash\n# @raycast.schemaVersion 1\n# @raycast.title Git Status\n# @raycast.mode fullOutput\n# @raycast.icon Terminal\n# @raycast.description Show git status of current directory\n\ngit -C \"$HOME\" status --short 2>/dev/null || echo \"Not in a git repository\"\n"},
		{"clock.sh", "#!/bin/bash\n# @raycast.schemaVersion 1\n# @raycast.title Current Time\n# @raycast.mode compact\n# @raycast.icon Clock\n# @raycast.description Show current date and time\n\ndate \"+%Y-%m-%d %H:%M:%S %Z\"\n"},
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

func scriptSlugify(s string) string {
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

func parseScriptMetadata(path string) *model.ScriptCommand {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	meta := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(trimmed, "#") {
			break
		}
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
		title = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	mode := model.ScriptMode(meta["mode"])
	if mode == "" {
		mode = model.ModeFullOutput
	}

	id := "script-" + scriptSlugify(title)
	cmd := &model.ScriptCommand{
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

	for i := 1; i <= 5; i++ {
		raw := meta[fmt.Sprintf("argument%d", i)]
		if raw == "" {
			break
		}
		var arg model.Argument
		if err := json.Unmarshal([]byte(raw), &arg); err != nil {
			arg = model.Argument{Name: fmt.Sprintf("arg%d", i), Type: "text", Placeholder: raw, Required: false}
		}
		if arg.Name == "" {
			arg.Name = fmt.Sprintf("arg%d", i)
		}
		cmd.Arguments = append(cmd.Arguments, arg)
	}

	return cmd
}
