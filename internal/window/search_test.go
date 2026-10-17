package window

import (
	"github.com/dyike/Void/internal/model"
	"testing"
)

func TestApplicationSearchPrefersTitleAndRejectsUnrelatedHelperPaths(t *testing.T) {
	commands := []model.CommandInfo{
		{ID: "helper", Title: "ABAssistantService", Subtitle: "/System/Library/Frameworks/AddressBook.framework/Versions/A/Helpers"},
		{ID: "preview", Title: "Safari Technology Preview"},
		{ID: "safari", Title: "Safari"},
		{ID: "vscode", Title: "Visual Studio Code"},
	}
	results := searchCommands(commands, "Safari")
	if len(results) != 2 || results[0].ID != "safari" || results[1].ID != "preview" {
		t.Fatalf("unexpected Safari results: %+v", results)
	}
	results = searchCommands(commands, "vsc")
	if len(results) != 1 || results[0].ID != "vscode" {
		t.Fatalf("fuzzy title search failed: %+v", results)
	}
}
