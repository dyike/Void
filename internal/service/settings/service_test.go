package settings_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/dyike/Void/internal/service/settings"
)

func TestSaveSettingsStructHack(t *testing.T) {
	dir, err := os.MkdirTemp("", "void-settings-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	svc := settings.NewService(dir)

	// Simulate frontend's SaveSettings(settings as any).
	// Frontend sends a full AppSettings object which gets deserialized as map[string]interface{}.
	patchJSON := `{
		"globalShortcut": "cmd+a",
		"fontSize": "large",
		"baseColor": "#121212",
		"pinnedCommands": ["system-ai-chat"],
		"recentCommands": [],
		"disabledCommands": [],
		"commandHotkeys": {},
		"ai": {
			"provider": "ollama",
			"openaiApiKey": "",
			"ollamaBaseUrl": "http://localhost:11434",
			"defaultModel": "llama3",
			"openaiCompatibleBaseUrl": "",
			"openaiCompatibleApiKey": "",
			"openaiCompatibleModel": "",
			"enabled": true
		}
	}`

	var patch settings.AppSettings
	if err := json.Unmarshal([]byte(patchJSON), &patch); err != nil {
		t.Fatal(err)
	}

	updated, err := svc.SaveSettings(patch)
	if err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	if updated.GlobalShortcut != "cmd+a" {
		t.Errorf("expected cmd+a, got %q", updated.GlobalShortcut)
	}
	if updated.AI.Provider != "ollama" {
		t.Errorf("expected ollama, got %q", updated.AI.Provider)
	}

	// Verify persistence
	b, _ := os.ReadFile(dir + "/settings.json")
	t.Logf("Saved JSON:\n%s", string(b))
}
