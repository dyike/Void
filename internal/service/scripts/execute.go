package scripts

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dyike/Void/internal/model"
)

// Execute runs a script command and returns the output.
func (s *Service) Execute(id string, args map[string]string) (*model.ExecutionResult, error) {
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

func (s *Service) runScript(cmd *model.ScriptCommand, args map[string]string) (*model.ExecutionResult, error) {
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
