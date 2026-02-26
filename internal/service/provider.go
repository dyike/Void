//go:build darwin

package service

import "github.com/google/wire"

// ProviderSet exposes all service constructors for dependency injection.
var ProviderSet = wire.NewSet(
	NewSettingsService,
	NewCommandService,
	NewClipboardService,
	NewAIService,
	NewSnippetService,
	NewCalculatorService,
	NewScriptsService,
	NewHotkeyService,
)
