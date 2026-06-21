package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Data directory: ~/Library/Application Support/Void
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("cannot find home directory:", err)
	}
	dataDir := filepath.Join(homeDir, "Library", "Application Support", "Void")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatal("cannot create data directory:", err)
	}

	// Instantiate services
	appContainer, err := InitApp(dataDir)
	if err != nil {
		log.Fatal("failed to initialize dependencies:", err)
	}
	svc := appContainer.Services

	// Start background clipboard polling
	svc.ClipboardService.Start()
	defer svc.ClipboardService.Stop()

	// Pre-warm the command cache in the background
	go func() {
		if _, err := svc.CommandsService.RefreshCommands(); err != nil {
			log.Println("command refresh error:", err)
		}
	}()

	// Expose Settings callbacks to the Settings service
	svc.SettingsService.SetOpenSettingsFunc(svc.WindowManager.OpenSettings)
	svc.SettingsService.SetResizeSettingsFunc(svc.WindowManager.ResizeSettings)
	svc.SettingsService.SetResizeMainFunc(svc.WindowManager.ResizeMain)
	svc.SettingsService.SetShowSettingsFunc(svc.WindowManager.ShowSettings)
	svc.SettingsService.SetToggleSystemTrayFunc(svc.WindowManager.ToggleSystemTray)

	app := application.New(application.Options{
		Name:        "Void",
		Description: "A powerful launcher for macOS",
		Services: []application.Service{
			application.NewService(svc.SettingsService),
			application.NewService(svc.CommandsService),
			application.NewService(svc.ClipboardService),
			application.NewService(svc.AiService),
			application.NewService(svc.SnippetsService),
			application.NewService(svc.CalcService),
			application.NewService(svc.ScriptsService),
			application.NewService(svc.ContainerService),
			application.NewService(svc.DockService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	})

	// Initialize the main window and system tray via Manager
	svc.WindowManager.SetSettingsService(svc.SettingsService)
	svc.WindowManager.SetDockService(svc.DockService)
	svc.WindowManager.Init(app)
	svc.WindowManager.CreateMainWindow("/")

	// ── Global Hotkey (Option+Space via CGEventTap) ───────────────────────
	go func() {
		time.Sleep(500 * time.Millisecond)
		keySpec := svc.SettingsService.GetGlobalHotkey()
		started := svc.HotkeyService.Start(svc.WindowManager.ToggleLauncher, keySpec)
		if started {
			log.Printf("hotkey: active (%s)", keySpec)
			svc.HotkeyService.StartSnippetExpand(svc.SnippetsService.ExpandSnippet, svc.SnippetsService.GetAllPrefixes)
		} else {
			log.Println("hotkey: CGEventTap not available — use system tray, or grant Accessibility in System Settings → Privacy & Security")
		}
	}()

	defer svc.HotkeyService.Stop()

	// Run
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
