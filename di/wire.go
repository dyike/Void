//go:build wireinject
// +build wireinject

package di

import (
	"github.com/dyike/Void/internal/service/ai"
	"github.com/dyike/Void/internal/service/calculator"
	"github.com/dyike/Void/internal/service/clipboard"
	"github.com/dyike/Void/internal/service/commands"
	"github.com/dyike/Void/internal/service/hotkey"
	"github.com/dyike/Void/internal/service/scripts"
	"github.com/dyike/Void/internal/service/settings"
	"github.com/dyike/Void/internal/service/snippets"
	"github.com/dyike/Void/internal/window"
	"github.com/google/wire"
	"github.com/wailsapp/wails/v3/pkg/services/dock"
)

// AppServices holds all instantiated services for injection into Wails.
type AppServices struct {
	SettingsService  *settings.Service
	CommandsService  *commands.Service
	ClipboardService *clipboard.Service
	AiService        *ai.Service
	SnippetsService  *snippets.Service
	CalcService      *calculator.Service
	ScriptsService   *scripts.Service
	HotkeyService    *hotkey.Service
	DockService      *dock.DockService
	WindowManager    *window.Manager
}

// InitializedApp is the final container returned to main.go.
type InitializedApp struct {
	Services *AppServices
	DataDir  string
}

// ServiceProviderSet aggregates all core service constructors.
var ServiceProviderSet = wire.NewSet(
	settings.ProviderSet,
	commands.ProviderSet,
	clipboard.ProviderSet,
	ai.ProviderSet,
	snippets.ProviderSet,
	calculator.ProviderSet,
	scripts.ProviderSet,
	hotkey.ProviderSet,
	window.ProviderSet,
	dock.New,
	ProvideAppServices,
	ProvideInitializedApp,
)

// ProvideAppServices aggregates all services into a single struct.
func ProvideAppServices(
	settingsSvc *settings.Service,
	commandsSvc *commands.Service,
	clipboardSvc *clipboard.Service,
	aiSvc *ai.Service,
	snippetsSvc *snippets.Service,
	calcSvc *calculator.Service,
	scriptsSvc *scripts.Service,
	hotkeySvc *hotkey.Service,
	dockSvc *dock.DockService,
	winManager *window.Manager,
) *AppServices {
	return &AppServices{
		SettingsService:  settingsSvc,
		CommandsService:  commandsSvc,
		ClipboardService: clipboardSvc,
		AiService:        aiSvc,
		SnippetsService:  snippetsSvc,
		CalcService:      calcSvc,
		ScriptsService:   scriptsSvc,
		HotkeyService:    hotkeySvc,
		DockService:      dockSvc,
		WindowManager:    winManager,
	}
}

// ProvideInitializedApp packages the services and the dataDir for the main loop.
func ProvideInitializedApp(services *AppServices, dataDir string) *InitializedApp {
	return &InitializedApp{
		Services: services,
		DataDir:  dataDir,
	}
}

// InitApp wires everything together given the base data directory.
func InitApp(dataDir string) (*InitializedApp, error) {
	wire.Build(ServiceProviderSet)
	return nil, nil // Return is handled by wire generated code
}
