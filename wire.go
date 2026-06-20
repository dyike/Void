//go:build wireinject
// +build wireinject

package main

import (
	"github.com/dyike/Void/internal/repo"
	"github.com/dyike/Void/internal/service"
	"github.com/dyike/Void/internal/window"
	"github.com/google/wire"
	"github.com/wailsapp/wails/v3/pkg/services/dock"
)

// AppServices holds all instantiated services for injection into Wails.
type AppServices struct {
	SettingsService  *service.SettingsService
	CommandsService  *service.CommandService
	ClipboardService *service.ClipboardService
	AiService        *service.AIService
	SnippetsService  *service.SnippetService
	CalcService      *service.CalculatorService
	ScriptsService   *service.ScriptsService
	ContainerService *service.ContainerService
	HotkeyService    *service.HotkeyService
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
	repo.ProviderSet,
	service.ProviderSet,
	window.ProviderSet,
	dock.New,
	ProvideAppServices,
	ProvideInitializedApp,
)

// ProvideAppServices aggregates all services into a single struct.
func ProvideAppServices(
	settingsSvc *service.SettingsService,
	commandsSvc *service.CommandService,
	clipboardSvc *service.ClipboardService,
	aiSvc *service.AIService,
	snippetsSvc *service.SnippetService,
	calcSvc *service.CalculatorService,
	scriptsSvc *service.ScriptsService,
	containerSvc *service.ContainerService,
	hotkeySvc *service.HotkeyService,
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
		ContainerService: containerSvc,
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
	return nil, nil
}
