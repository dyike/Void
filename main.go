package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dyike/Void/internal/service"
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

	var hotkeyMu sync.Mutex
	startHotkey := func(keySpec string) bool {
		hotkeyMu.Lock()
		defer hotkeyMu.Unlock()

		svc.HotkeyService.Stop()
		if started := svc.HotkeyService.Start(svc.WindowManager.ToggleLauncher, keySpec); started {
			log.Printf("hotkey: active (%s)", keySpec)
			svc.HotkeyService.StartSnippetExpand(svc.SnippetsService.ExpandSnippet, svc.SnippetsService.GetAllPrefixes)
			return true
		}
		return false
	}

	service.RegisterSettingsHotkeyChangedFunc(svc.SettingsService, func(spec string) {
		go func() {
			log.Printf("hotkey: restarting (%s)", spec)
			if !startHotkey(spec) {
				log.Println("hotkey: restart pending — grant Accessibility and Input Monitoring permissions in System Settings → Privacy & Security")
			}
		}()
	})

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
		retryDelay := 60 * time.Second
		for attempt := 1; ; attempt++ {
			keySpec := svc.SettingsService.GetGlobalHotkey()
			if startHotkey(keySpec) {
				return
			}
			if attempt == 1 || attempt%5 == 0 {
				log.Println("hotkey: waiting for Accessibility and Input Monitoring permissions in System Settings → Privacy & Security")
			}
			time.Sleep(retryDelay)
		}
	}()

	defer svc.HotkeyService.Stop()

	// Run
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
