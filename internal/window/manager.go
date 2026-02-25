package window

import (
	"github.com/dyike/Void/internal/service/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Manager encapsulates all Wails window and system tray operations.
type Manager struct {
	app            *application.App
	mainWindow     *application.WebviewWindow
	settingsWindow *application.WebviewWindow
	tray           *application.SystemTray
	settingsSvc    *settings.Service
}

// NewManager creates a new window manager.
func NewManager() *Manager {
	return &Manager{}
}

// SetSettingsService wires the settings service for theme-aware window creation.
func (m *Manager) SetSettingsService(svc *settings.Service) {
	m.settingsSvc = svc
}

// Init sets the Wails Application instance, creates the system tray, and binds global window events.
func (m *Manager) Init(app *application.App) {
	m.app = app
	m.setupSystemTray()

	// Intercept main app lifecycle events if necessary (Wails v3 manages most via Run())
}

// themeAppearance converts a theme mode string to a macOS appearance.
func themeAppearance(mode string) application.MacAppearanceType {
	switch mode {
	case "dark":
		return application.NSAppearanceNameDarkAqua
	case "light":
		return application.NSAppearanceNameAqua
	default:
		// "system" — no explicit appearance, follow macOS
		return ""
	}
}

// currentThemeMode reads the current theme mode from settings.
func (m *Manager) currentThemeMode() string {
	if m.settingsSvc == nil {
		return "system"
	}
	s, err := m.settingsSvc.GetSettings()
	if err != nil || s.ThemeMode == "" {
		return "system"
	}
	return s.ThemeMode
}

// CreateMainWindow constructs the main search and launcher window.
func (m *Manager) CreateMainWindow(url string) *application.WebviewWindow {
	themeMode := m.currentThemeMode()
	m.mainWindow = m.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Void",
		Width:            680,
		Height:           480,
		MinWidth:         500,
		MinHeight:        300,
		Frameless:        true,
		AlwaysOnTop:      true,
		Hidden:           false,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              url,
		Mac: application.MacWindow{
			Backdrop:   application.MacBackdropTranslucent,
			TitleBar:   application.MacTitleBarHidden,
			Appearance: themeAppearance(themeMode),
		},
	})

	// Hide window on close instead of destroying it
	m.mainWindow.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		e.Cancel()
		m.mainWindow.Hide()
	})

	// Auto-hide when the window loses focus (like Raycast)
	m.mainWindow.OnWindowEvent(events.Common.WindowLostFocus, func(e *application.WindowEvent) {
		m.mainWindow.Hide()
	})

	return m.mainWindow
}

// ToggleLauncher shows/hides the main launcher window.
func (m *Manager) ToggleLauncher() {
	if m.mainWindow == nil {
		return
	}
	if m.mainWindow.IsVisible() {
		m.mainWindow.Hide()
	} else {
		m.mainWindow.Center()
		m.mainWindow.Show()
		m.mainWindow.Focus()
	}
}

// OpenSettings ensures the Settings window is open, creating it if it was closed natively.
func (m *Manager) OpenSettings() {
	if m.app == nil {
		return
	}

	if m.settingsWindow != nil {
		m.settingsWindow.Show()
		m.settingsWindow.Focus()
		return
	}

	themeMode := m.currentThemeMode()

	m.settingsWindow = m.app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Void Settings",
		Width:            780,
		Height:           520,
		MinWidth:         600,
		MinHeight:        400,
		Frameless:        false,
		AlwaysOnTop:      false,
		Hidden:           true,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		URL:              "/#/settings",
		Mac: application.MacWindow{
			Backdrop:   application.MacBackdropTranslucent,
			TitleBar:   application.MacTitleBarHiddenInsetUnified,
			Appearance: themeAppearance(themeMode),
		},
		MinimiseButtonState: application.ButtonDisabled,
		MaximiseButtonState: application.ButtonDisabled,
	})

	m.settingsWindow.Center()

	m.settingsWindow.OnWindowEvent(events.Common.WindowClosing, func(e *application.WindowEvent) {
		m.settingsWindow = nil
	})
}

// GetMainWindow returns the main launcher window (for external use).
func (m *Manager) GetMainWindow() *application.WebviewWindow {
	return m.mainWindow
}

// ShowSettings makes the settings window visible. Called after the frontend signals it is ready.
func (m *Manager) ShowSettings() {
	if m.settingsWindow != nil {
		m.settingsWindow.Show()
		m.settingsWindow.Focus()
	}
}

// ResizeSettings safely adjusts the standalone settings window if it exists.
func (m *Manager) ResizeSettings(width, height int) {
	if m.settingsWindow != nil {
		cx, cy := m.settingsWindow.Position()
		m.settingsWindow.SetSize(width, height)
		m.settingsWindow.SetPosition(cx, cy)
	}
}

// setupSystemTray hooks up the generic menu bar accessory icon (like Raycast's menu item).
func (m *Manager) setupSystemTray() {
	if m.app == nil {
		return
	}
	m.tray = m.app.SystemTray.New()
	m.tray.SetLabel("Void")

	trayMenu := m.app.NewMenu()
	trayMenu.Add("Show Void").OnClick(func(ctx *application.Context) {
		if m.mainWindow != nil {
			m.mainWindow.Center()
			m.mainWindow.Show()
			m.mainWindow.Focus()
		}
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		m.app.Quit()
	})

	m.tray.SetMenu(trayMenu)
	m.tray.OnClick(func() {
		m.ToggleLauncher()
	})

	// Initial tray visibility
	if m.settingsSvc != nil {
		settings, _ := m.settingsSvc.GetSettings()
		if settings != nil && !settings.ShowInMenuBar {
			m.tray.Hide()
		}
	}
}

// ToggleSystemTray shows or hides the menu bar tray icon
func (m *Manager) ToggleSystemTray(show bool) {
	if m.tray == nil {
		return
	}
	if show {
		m.tray.Show()
	} else {
		m.tray.Hide()
	}
}
