package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dyike/Void/internal/service"
	"github.com/dyike/Void/internal/window"
	"github.com/dyike/keel/ui/core"
	keelwindow "github.com/dyike/keel/ui/window"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	dataDir := flag.String("data-dir", filepath.Join(home, "Library", "Application Support", "Void"), "application data directory")
	preview := flag.Bool("preview", false, "show the UI without global hotkeys, clipboard polling or login integration")
	material := flag.String("material", "", "window material: clear, regular or frosted")
	backdrop := flag.Bool("preview-backdrop", false, "show a colored reference behind the launcher for material diagnostics")
	screenshot := flag.String("screenshot", "", "render the launcher to a PNG and exit")
	flag.Parse()
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatal(err)
	}
	container, err := InitApp(*dataDir)
	if err != nil {
		log.Fatal(err)
	}
	svc := container.Services
	manager := svc.WindowManager
	manager.SetPreview(*preview)
	manager.Init(window.Services{
		ShortcutRecording: svc.HotkeyService.SetRecording,
		Settings:          svc.SettingsService, Commands: svc.CommandsService,
		Clipboard: svc.ClipboardService,
		Snippets:  svc.SnippetsService, Calculator: svc.CalcService,
		Scripts: svc.ScriptsService, Containers: svc.ContainerService,
	})
	if err := manager.SetGlassStyle(*material); err != nil {
		log.Fatal(err)
	}
	if *screenshot != "" {
		if err := manager.Screenshot(*screenshot); err != nil {
			log.Fatal(err)
		}
		return
	}
	if err := setFallbackAppIcon(); err != nil {
		log.Print(err)
	}
	svc.CommandsService.SetHideWindowFunc(func() { core.Update(manager.HideLauncher) })
	svc.SettingsService.SetOpenSettingsFunc(func() { core.Update(manager.OpenSettings) })
	svc.SettingsService.SetResizeSettingsFunc(func(w, h int) { core.Update(func() { manager.ResizeSettings(w, h) }) })
	svc.SettingsService.SetResizeMainFunc(func(w, h int) { core.Update(func() { manager.ResizeMain(w, h) }) })
	svc.SettingsService.SetShowSettingsFunc(func() { core.Update(manager.ShowSettings) })
	headless := os.Getenv("KEEL_HEADLESS") == "1" || *preview
	if headless {
		svc.SettingsService.SetLaunchAtLoginFunc(nil)
	} else {
		svc.SettingsService.SetToggleSystemTrayFunc(manager.ToggleSystemTray)
	}
	if !headless {
		svc.ClipboardService.Start()
	}
	var hotkeyMu sync.Mutex
	done := make(chan struct{})
	startHotkey := func(spec string) bool {
		if headless {
			return false
		}
		hotkeyMu.Lock()
		defer hotkeyMu.Unlock()
		select {
		case <-done:
			return false
		default:
		}
		svc.HotkeyService.Stop()
		if !svc.HotkeyService.Start(func() { core.Update(manager.ToggleLauncher) }, spec) {
			return false
		}
		svc.HotkeyService.StartSnippetExpand(svc.SnippetsService.ExpandSnippet, svc.SnippetsService.GetAllPrefixes)
		return true
	}
	service.RegisterSettingsHotkeyChangedFunc(svc.SettingsService, func(spec string) { go startHotkey(spec) })
	// Native tray creation must run outside the UI frame lock.
	if !headless {
		manager.ToggleSystemTrayInitial()
	}
	if *backdrop {
		manager.OpenPreviewBackdrop()
	}
	manager.ShowLauncher()
	go func() {
		if headless {
			return
		}
		timer := time.NewTimer(500 * time.Millisecond)
		defer timer.Stop()
		for {
			select {
			case <-done:
				return
			case <-timer.C:
			}
			if startHotkey(svc.SettingsService.GetGlobalHotkey()) {
				return
			}
			log.Print("hotkey: grant Accessibility and Input Monitoring permissions in System Settings")
			timer.Reset(time.Minute)
		}
	}()
	keelwindow.MainWithOptions(keelwindow.MainOptions{KeepAlive: true, OnQuit: func() {
		close(done)
		hotkeyMu.Lock()
		svc.HotkeyService.Stop()
		hotkeyMu.Unlock()
		if !headless {
			svc.ClipboardService.Stop()
		}
		manager.Close()
	}})
}
