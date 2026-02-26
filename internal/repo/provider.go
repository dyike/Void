//go:build darwin

package repo

import "github.com/google/wire"

// ProviderSet exposes all repository implementations for dependency injection.
var ProviderSet = wire.NewSet(
	NewSettingsRepo,
	NewCommandRepo,
	NewCalculatorRepo,
	NewSnippetRepo,
	NewClipboardRepo,
	NewScriptsRepo,
	NewHotkeyRepo,
)
