package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/dyike/Void/internal/service/ai"
	"github.com/dyike/Void/internal/service/calculator"
	"github.com/dyike/Void/internal/service/clipboard"
	"github.com/dyike/Void/internal/service/commands"
	"github.com/dyike/Void/internal/service/hotkey"
	"github.com/dyike/Void/internal/service/scripts"
	"github.com/dyike/Void/internal/service/settings"
	"github.com/dyike/Void/internal/service/snippets"
	"github.com/dyike/Void/internal/window"
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
	settingsSvc := settings.NewService(dataDir)
	commandsSvc := commands.NewService(dataDir, settingsSvc)
	clipboardSvc := clipboard.NewService(dataDir)
	aiSvc := ai.NewService(settingsSvc)
	snippetsSvc := snippets.NewService(dataDir)
	calcSvc := calculator.NewService()
	scriptsSvc := scripts.NewService(dataDir)
	hotkeySvc := hotkey.NewService()

	// Start background clipboard polling
	clipboardSvc.Start()
	defer clipboardSvc.Stop()

	// Pre-warm the command cache in the background
	go func() {
		if _, err := commandsSvc.RefreshCommands(); err != nil {
			log.Println("command refresh error:", err)
		}
	}()

	// Create the window manager
	winManager := window.NewManager()

	// Expose Settings callbacks to the Settings service
	settingsSvc.SetOpenSettingsFunc(winManager.OpenSettings)
	settingsSvc.SetResizeSettingsFunc(winManager.ResizeSettings)
	settingsSvc.SetShowSettingsFunc(winManager.ShowSettings)

	app := application.New(application.Options{
		Name:        "Void",
		Description: "A powerful launcher for macOS",
		Services: []application.Service{
			application.NewService(settingsSvc),
			application.NewService(commandsSvc),
			application.NewService(clipboardSvc),
			application.NewService(aiSvc),
			application.NewService(snippetsSvc),
			application.NewService(calcSvc),
			application.NewService(scriptsSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			// Keep the app alive in the menu bar even when the window is closed
			ApplicationShouldTerminateAfterLastWindowClosed: false,
			// Hide Dock icon (run as an accessory app like Raycast)
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	})

	// Initialize the main window and system tray via Manager
	winManager.SetSettingsService(settingsSvc)
	winManager.Init(app)
	winManager.CreateMainWindow("/")

	// ── Global Hotkey (Option+Space via CGEventTap) ───────────────────────
	// Start in a goroutine after a short delay to ensure the app run loop is ready.
	go func() {
		time.Sleep(500 * time.Millisecond)
		// Load hotkey from settings, defaulting to "option+space"
		keySpec := settingsSvc.GetGlobalHotkey()
		started := hotkeySvc.Start(winManager.ToggleLauncher, keySpec)
		if started {
			log.Printf("hotkey: active (%s)", keySpec)
			// Start snippet auto-expand after event tap is active
			hotkeySvc.StartSnippetExpand(snippetsSvc.ExpandSnippet, snippetsSvc.GetAllPrefixes)
		} else {
			log.Println("hotkey: CGEventTap not available — use system tray, or grant Accessibility in System Settings → Privacy & Security")
		}
	}()

	defer hotkeySvc.Stop()

	// Run
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
