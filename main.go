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
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
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
	commandsSvc := commands.NewService(dataDir)
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

	// Main launcher window — frameless, translucent, always on top
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Void",
		Width:            680,
		Height:           480,
		MinWidth:         500,
		MinHeight:        300,
		Frameless:        true,
		AlwaysOnTop:      true,
		Hidden:           false,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              "/",
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarHidden,
		},
	})

	// Settings window — standard macOS native window with title bar
	var settingsWindow *application.WebviewWindow
	openSettingsWindow := func() {
		if settingsWindow != nil {
			settingsWindow.Show()
			settingsWindow.Focus()
			return
		}
		settingsWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
			Title:            "Void Settings",
			Width:            780,
			Height:           520,
			MinWidth:         600,
			MinHeight:        400,
			Frameless:        false,
			AlwaysOnTop:      false,
			Hidden:           false,
			BackgroundColour: application.NewRGBA(40, 40, 40, 255),
			URL:              "/#/settings",
			Mac: application.MacWindow{
				Backdrop: application.MacBackdropTranslucent,
				TitleBar: application.MacTitleBarHiddenInsetUnified,
			},
			MinimiseButtonState: application.ButtonDisabled,
			MaximiseButtonState: application.ButtonDisabled,
		})
		settingsWindow.Center()
		settingsWindow.Show()
		settingsWindow.Focus()
		settingsWindow.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
			settingsWindow = nil
		})
	}

	// Expose OpenSettingsWindow to frontend via settings service
	settingsSvc.SetOpenSettingsFunc(openSettingsWindow)

	// toggleWindow shows/hides the main launcher window.
	toggleWindow := func() {
		if mainWindow.IsVisible() {
			mainWindow.Hide()
		} else {
			mainWindow.Center()
			mainWindow.Show()
			mainWindow.Focus()
		}
	}

	// Hide window on close instead of destroying it
	mainWindow.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		mainWindow.Hide()
	})

	// Auto-hide when the window loses focus (like Raycast)
	mainWindow.OnWindowEvent(events.Common.WindowLostFocus, func(e *application.WindowEvent) {
		mainWindow.Hide()
	})

	// ── Global Hotkey (Option+Space via CGEventTap) ───────────────────────
	// Start in a goroutine after a short delay to ensure the app run loop is ready.
	go func() {
		time.Sleep(500 * time.Millisecond)
		// Load hotkey from settings, defaulting to "option+space"
		keySpec := settingsSvc.GetGlobalHotkey()
		started := hotkeySvc.Start(toggleWindow, keySpec)
		if started {
			log.Printf("hotkey: active (%s)", keySpec)
			// Start snippet auto-expand after event tap is active
			hotkeySvc.StartSnippetExpand(snippetsSvc.ExpandSnippet, snippetsSvc.GetAllPrefixes)
		} else {
			log.Println("hotkey: CGEventTap not available — use system tray, or grant Accessibility in System Settings → Privacy & Security")
		}
	}()

	defer hotkeySvc.Stop()

	// ── System tray with show/quit menu ───────────────────────────────────
	tray := app.SystemTray.New()
	tray.SetLabel("Void")
	trayMenu := app.NewMenu()
	trayMenu.Add("Show Void").OnClick(func(ctx *application.Context) {
		mainWindow.Center()
		mainWindow.Show()
		mainWindow.Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	tray.SetMenu(trayMenu)
	tray.OnClick(func() {
		toggleWindow()
	})

	// Run
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
