//go:build wireinject
// +build wireinject

package main

import (
	"github.com/dyike/Void/internal/repo"
	"github.com/dyike/Void/internal/service"
	"github.com/dyike/Void/internal/window"
	"github.com/google/wire"
)

// AppServices holds all instantiated services for use by the native UI.
type AppServices struct {
	SettingsService  *service.SettingsService
	CommandsService  *service.CommandService
	ClipboardService *service.ClipboardService
	SnippetsService  *service.SnippetService
	CalcService      *service.CalculatorService
	ScriptsService   *service.ScriptsService
	ContainerService *service.ContainerService
	HotkeyService    *service.HotkeyService
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
	ProvideAppServices,
	ProvideInitializedApp,
)

// ProvideAppServices aggregates all services into a single struct.
func ProvideAppServices(
	settingsSvc *service.SettingsService,
	commandsSvc *service.CommandService,
	clipboardSvc *service.ClipboardService,
	snippetsSvc *service.SnippetService,
	calcSvc *service.CalculatorService,
	scriptsSvc *service.ScriptsService,
	containerSvc *service.ContainerService,
	hotkeySvc *service.HotkeyService,
	winManager *window.Manager,
) *AppServices {
	return &AppServices{
		SettingsService:  settingsSvc,
		CommandsService:  commandsSvc,
		ClipboardService: clipboardSvc,
		SnippetsService:  snippetsSvc,
		CalcService:      calcSvc,
		ScriptsService:   scriptsSvc,
		ContainerService: containerSvc,
		HotkeyService:    hotkeySvc,
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
