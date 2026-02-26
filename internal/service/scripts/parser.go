package scripts

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dyike/Void/internal/model"
)

// parseScriptMetadata reads @raycast.xxx metadata from the script header.
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

	mode := model.ScriptMode(meta["mode"])
	if mode == "" {
		mode = model.ModeFullOutput
	}

	id := "script-" + slugify(title)
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

	// Parse argument definitions: @raycast.argument1, @raycast.argument2, ...
	for i := 1; i <= 5; i++ {
		raw := meta[fmt.Sprintf("argument%d", i)]
		if raw == "" {
			break
		}
		// Format: {"type": "text", "placeholder": "Query", "required": true}
		var arg model.Argument
		if err := json.Unmarshal([]byte(raw), &arg); err != nil {
			// Fallback: treat whole value as placeholder
			arg = model.Argument{Name: fmt.Sprintf("arg%d", i), Type: "text", Placeholder: raw, Required: false}
		}
		if arg.Name == "" {
			arg.Name = fmt.Sprintf("arg%d", i)
		}
		cmd.Arguments = append(cmd.Arguments, arg)
	}

	return cmd
}
