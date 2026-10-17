package repo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRemovesRetiredAIConfiguration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	legacy := `{
		"globalShortcut":"option+shift+space",
		"themeMode":"dark",
		"ai":{"enabled":true,"openaiApiKey":"test-secret"},
		"skillsPath":"/some/user/directory",
		"pinnedCommands":["system-ai-chat","system-clipboard-manager"],
		"recentCommands":["system-ai-chat"],
		"disabledCommands":["system-ai-chat","system-sleep"],
		"commandHotkeys":{"system-ai-chat":"cmd+j","system-sleep":"cmd+s"},
		"customSetting":{"preserve":true}
	}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsRepo(dir).Load()
	if settings.GlobalShortcut != "option+shift+space" || settings.ThemeMode != "dark" {
		t.Fatal("migration changed unrelated preferences")
	}
	if len(settings.PinnedCommands) != 1 || settings.PinnedCommands[0] != "system-clipboard-manager" || len(settings.RecentCommands) != 0 {
		t.Fatal("migration did not clean obsolete command references")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ai", "skillsPath"} {
		if _, exists := fields[key]; exists {
			t.Fatalf("retired setting %q remains on disk", key)
		}
	}
	if _, exists := fields["customSetting"]; !exists {
		t.Fatal("migration dropped an unrelated setting")
	}
	if settings.CommandHotkeys["system-sleep"] != "cmd+s" || len(settings.CommandHotkeys) != 1 {
		t.Fatal("migration changed unrelated command shortcuts")
	}
	// Running it again leaves the persisted configuration byte-for-byte intact.
	NewSettingsRepo(dir).Load()
	again, err := os.ReadFile(path)
	if err != nil || string(again) != string(data) {
		t.Fatal("migration is not idempotent")
	}
}
