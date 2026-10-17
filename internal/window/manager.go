package window

import (
	"bytes"
	"fmt"
	"image/png"
	"log"
	"sync"
	"sync/atomic"

	"github.com/dyike/Void/internal/branding"
	"github.com/dyike/Void/internal/service"
	"github.com/dyike/keel/native/tray"
	"github.com/dyike/keel/ui/core"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/locale"
	"github.com/dyike/keel/ui/theme"
	"github.com/dyike/keel/ui/window"
)

// Services keeps business logic independent of the native presentation layer.
type Services struct {
	ShortcutRecording func(bool)
	Settings          *service.SettingsService
	Commands          *service.CommandService
	Clipboard         *service.ClipboardService
	Snippets          *service.SnippetService
	Calculator        *service.CalculatorService
	Scripts           *service.ScriptsService
	Containers        *service.ContainerService
}

// Manager owns resident windows; closing a window preserves its view state.
// Window state is accessed on Keel's UI queue. Tray state has its own lock.
type Manager struct {
	glassStyle                 window.GlassStyle
	dark                       bool
	preview                    bool
	screenshot                 bool
	fontSize                   float32
	services                   Services
	mainWindow, settingsWindow *window.Window
	launcher                   *launcherView
	settings                   *settingsView
	trayMu                     sync.Mutex
	tray                       *tray.Tray
	trayVersion                uint64
	quitting                   atomic.Bool
}

func NewManager() *Manager { return &Manager{glassStyle: window.GlassFrosted} }
func (m *Manager) Init(s Services) {
	text := locale.English()
	text.SearchSettings = "Search"
	locale.Apply(text)
	m.services = s
	m.launcher = newLauncher(m)
	m.applyTheme()
}
func (m *Manager) applyTheme() {
	cfg, err := m.services.Settings.GetSettings()
	if err != nil {
		return
	}
	dark := cfg.ThemeMode == "dark" || (cfg.ThemeMode == "system" && window.SystemAppearance() == window.AppearanceDark)
	previousMaterial := m.glassStyle
	m.glassStyle = materialStyle(cfg.WindowMaterial)
	m.dark = dark
	theme.Apply(launcherPalette(dark))

	m.fontSize = 15
	switch cfg.FontSize {
	case "small":
		m.fontSize = 13
	case "large":
		m.fontSize = 17
	}
	_ = window.SetNativeAppearance(window.Appearance(cfg.ThemeMode))
	if previousMaterial != m.glassStyle && m.mainWindow != nil {
		previous := m.mainWindow
		m.mainWindow = nil
		previous.Close()
		m.ShowLauncher()
	}
	if previousMaterial != m.glassStyle && m.settingsWindow != nil {
		previous := m.settingsWindow
		m.settingsWindow = nil
		previous.Close()
		m.OpenSettings()
	}
}
func (m *Manager) SetGlassStyle(style string) error {
	if style == "" {
		return nil
	}
	switch style {
	case "clear":
		m.glassStyle = window.GlassClear
	case "regular":
		m.glassStyle = window.GlassRegular
	case "frosted":
		m.glassStyle = window.GlassFrosted
	default:
		return fmt.Errorf("unknown material %q", style)
	}
	return nil
}

func (m *Manager) SetPreview(enabled bool) { m.preview = enabled }

func (m *Manager) ShowLauncher() {
	if m.mainWindow != nil && !m.mainWindow.Closed() {
		m.launcher.focusSearch = m.launcher.form == "" && !m.launcher.actions && !m.launcher.dialog.Value()
		m.mainWindow.Raise()
		return
	}
	m.launcher.focusSearch = true
	m.launcher.hadFocus = false
	m.launcher.refresh()
	var opened *window.Window
	opened = window.Open(window.Options{
		Title: "Void", Width: 760, Height: 560, MinWidth: 560, MinHeight: 360,
		Frameless: true, TopMost: true, Glass: &window.GlassOptions{Style: m.glassStyle, CornerRadius: 24},
		Content: el.Root(m.launcher),
		Shortcuts: map[string]func(){
			"mod+,": m.OpenSettings, "mod+q": window.Quit,
			"esc":   m.launcher.escape,
			"mod+k": m.launcher.toggleActions,
			"mod+0": func() { m.launcher.navigate("Commands") },
			"mod+1": func() { m.launcher.navigate("Clipboard") },
			"mod+2": func() { m.launcher.navigate("Snippets") },
			"mod+3": func() { m.launcher.navigate("Scripts") },
			"mod+4": func() { m.launcher.navigate("Files") },
			"mod+5": func() { m.launcher.navigate("Containers") },
		},
		OnResize: func(int, int) { configureLauncherSpaces() },
		OnClose: func() {
			if m.mainWindow == opened {
				m.mainWindow = nil
			}
		},
	})
	m.mainWindow = opened
	// Creating an accessory app's window only orders it onscreen. Raise also
	// activates the process so typing and window shortcuts reach this window.
	opened.Raise()
}
func (m *Manager) HideLauncher() {
	if m.mainWindow != nil {
		m.mainWindow.Close()
	}
}
func (m *Manager) ToggleLauncher() {
	if m.mainWindow != nil && !m.mainWindow.Closed() {
		m.HideLauncher()
	} else {
		m.ShowLauncher()
	}
}
func (m *Manager) OpenSettings() {
	if m.settingsWindow != nil && !m.settingsWindow.Closed() {
		m.settingsWindow.Raise()
		return
	}
	if m.settings == nil {
		m.settings = newSettings(m)
	} else {
		m.settings.load()
	}
	var opened *window.Window
	opened = window.Open(window.Options{
		Title: "Settings", Width: 980, Height: 680, MinWidth: 820, MinHeight: 540,
		Frameless: true, NativeTrafficLights: true, TopMost: true,
		TrafficLightLayout: &window.TrafficLightLayout{Height: 52, Left: 18},
		Glass:              &window.GlassOptions{Style: m.glassStyle, CornerRadius: 24},
		Content:            &settingsContent{view: m.settings, root: el.Root(m.settings)},
		OnClose: func() {
			if m.settingsWindow == opened {
				m.settings.shortcut.stop()
				m.settingsWindow = nil
			}
		},
	})
	m.settingsWindow = opened
	opened.Raise()
}
func (m *Manager) ShowSettings() { m.OpenSettings() }
func (m *Manager) ResizeSettings(w, h int) {
	if m.settingsWindow != nil {
		_ = m.settingsWindow.Resize(w, h)
	}
}
func (m *Manager) ResizeMain(w, h int) {
	if m.mainWindow != nil {
		_ = m.mainWindow.Resize(w, h)
	}
}
func (m *Manager) GetMainWindow() *window.Window { return m.mainWindow }
func (m *Manager) Screenshot(path string) error {
	m.screenshot = true
	defer func() { m.screenshot = false }()
	rows, err := m.launcher.loadRows("Commands", "")
	if err != nil {
		return err
	}
	m.launcher.rows = rows
	m.launcher.list.SetCount(len(rows))
	return window.Screenshot(el.Root(m.launcher), 760, 560, path)
}
func (m *Manager) ToggleSystemTrayInitial() {
	cfg, _ := m.services.Settings.GetSettings()
	if cfg != nil && cfg.ShowInMenuBar {
		m.setTray(true, 0)
	}
}
func (m *Manager) ToggleSystemTray(show bool) {
	m.trayMu.Lock()
	m.trayVersion++
	version := m.trayVersion
	m.trayMu.Unlock()
	go m.setTray(show, version)
}
func (m *Manager) setTray(show bool, version uint64) {
	m.trayMu.Lock()
	defer m.trayMu.Unlock()
	if m.quitting.Load() || version != m.trayVersion {
		return
	}
	if !show {
		if m.tray != nil {
			_ = m.tray.Close()
			m.tray = nil
		}
		return
	}
	if m.tray != nil {
		return
	}
	art, err := png.Decode(bytes.NewReader(branding.TrayIcon))
	if err != nil {
		log.Print(err)
		return
	}
	m.tray, err = tray.New(tray.Options{Icon: art, Template: true, Tooltip: "Void", OnActivate: func() { core.Update(m.ToggleLauncher) }, Menu: []tray.MenuItem{
		{ID: "open", Title: "Show Void", OnSelect: func() { core.Update(m.ShowLauncher) }},
		{ID: "settings", Title: "Settings…", OnSelect: func() { core.Update(m.OpenSettings) }},
		{Separator: true}, {ID: "quit", Title: "Quit Void", OnSelect: window.Quit},
	}})
	if err != nil {
		log.Printf("system tray: %v", err)
	} else if m.quitting.Load() {
		_ = m.tray.Close()
		m.tray = nil
	}
}
func (m *Manager) Close() {
	// Tray creation may be waiting for AppKit. Never wait for it under the frame lock.
	m.quitting.Store(true)
	if !m.trayMu.TryLock() {
		return
	}
	defer m.trayMu.Unlock()
	if m.tray != nil {
		_ = m.tray.Close()
		m.tray = nil
	}
}
