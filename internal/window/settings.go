package window

import (
	"image/color"
	"os"
	"strings"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/keel/gio/io/key"
	"github.com/dyike/keel/ui/core"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/kit"
	"github.com/dyike/keel/ui/theme"
)

type settingsView struct {
	manager               *Manager
	config                model.AppSettings
	inputs                map[string]*kit.InputView
	shortcut              *shortcutRecorder
	theme, font, material *kit.SelectView
	login, tray           *settingsSwitch
	pages                 *kit.SettingsView
	status                string
	busy, dirty           bool
}

// Keep switch thumbs visible on dark surfaces without changing other controls.
type settingsSwitch struct{ *kit.SwitchView }

func (v *settingsSwitch) Render(cx *el.Context) el.Element {
	p := theme.Current()
	p.Surface = theme.RGB(0xffffff)
	return cx.Themed(p, v.SwitchView)
}

// Keep fields compact while retaining each kit control's identity and accessible name.
type settingsInput struct {
	*kit.InputView
	width float32
}

func (v *settingsInput) Render(cx *el.Context) el.Element {
	return v.InputView.Render(cx).(*el.DivEl).W(el.Dp(v.width)).MaxW(el.Full)
}

type settingsSelect struct{ *kit.SelectView }

func (v *settingsSelect) Render(cx *el.Context) el.Element {
	return v.SelectView.Render(cx).(*el.DivEl).W(el.Dp(110)).MaxW(el.Full)
}

func compactSettingsSelect(control *kit.SelectView) *settingsSelect {
	control.MenuWidth(156).MenuAlign(el.End).RenderValue(func(_ *el.Context, options []kit.SelectOption) el.Element {
		if len(options) == 0 {
			return nil
		}
		return el.Div().WFull().Items(el.End).Child(el.Text(options[0].Label).MaxLines(1))
	})
	return &settingsSelect{control}
}

func newSettings(m *Manager) *settingsView { v := &settingsView{manager: m}; v.load(); return v }
func (v *settingsView) load() {
	page, query := "General", ""
	if v.pages != nil {
		page, query = v.pages.Value(), v.pages.Query()
	}
	cfg, err := v.manager.services.Settings.GetSettings()
	if err != nil {
		v.status = err.Error()
		return
	}
	v.config = *cfg
	v.inputs = map[string]*kit.InputView{}
	for _, field := range []struct {
		key, value string
	}{
		{"terminal", cfg.TerminalApp},
	} {
		f := kit.Input("").Size(kit.InputSizeSmall).OnChange(func(string) { v.changed() })
		f.SetValue(field.value)
		v.inputs[field.key] = f
	}
	if v.shortcut != nil {
		v.shortcut.stop()
	}
	v.shortcut = &shortcutRecorder{value: cfg.GlobalShortcut, changed: v.changed, recordingChanged: v.manager.services.ShortcutRecording}
	v.inputs["terminal"].Placeholder("Terminal")
	v.material = kit.Select("").Size(28).Appearance(false).OnChange(func(string) { v.changed() })
	v.material.SetEntries(kit.SelectOption{Value: "clear", Label: "Clear glass"}, kit.SelectOption{Value: "regular", Label: "Soft glass"}, kit.SelectOption{Value: "frosted", Label: "Frosted"})
	v.material.SetValue(cfg.WindowMaterial)
	v.theme = kit.Select("").Size(28).Appearance(false).OnChange(func(string) { v.changed() })
	v.theme.SetEntries(kit.SelectOption{Value: "system", Label: "System"}, kit.SelectOption{Value: "light", Label: "Light"}, kit.SelectOption{Value: "dark", Label: "Dark"})
	v.theme.SetValue(cfg.ThemeMode)
	v.font = kit.Select("").Size(28).Appearance(false).OnChange(func(string) { v.changed() })
	v.font.SetEntries(kit.SelectOption{Value: "small", Label: "Small"}, kit.SelectOption{Value: "medium", Label: "Medium"}, kit.SelectOption{Value: "large", Label: "Large"})
	v.font.SetValue(cfg.FontSize)
	v.login = &settingsSwitch{kit.Switch("", cfg.LaunchAtLogin).OnChange(func(bool) { v.changed() })}
	v.tray = &settingsSwitch{kit.Switch("", cfg.ShowInMenuBar).OnChange(func(bool) { v.changed() })}
	v.pages = kit.Settings().SearchInSidebar(true).PageHeading(false).
		Size(kit.SettingsSizeSmall).RowSpacing(6).GroupVariant(kit.GroupBoxFill).
		Page(kit.SettingPage{Title: "General", Icon: kit.IconSettings, Groups: []kit.SettingGroup{
			{Title: "General", Items: []kit.SettingItem{
				{Label: "Launch at login", Description: "Keep Void ready whenever you log in to your Mac.", Control: v.login, Keywords: []string{"startup", "launch", "开机"}},
				{Label: "Show in menu bar", Description: "Quickly open Void and access settings from the menu bar.", Control: v.tray, Keywords: []string{"tray", "菜单栏"}},
			}},
			{Title: "Global Hotkey", Items: []kit.SettingItem{
				{Label: "Global shortcut", Description: "Open or close Void from anywhere.", Control: v.shortcut, Keywords: []string{"keyboard", "hotkey", "快捷键"}},
			}, Footer: el.ViewFunc(func(*el.Context) el.Element {
				return el.Text(v.shortcut.hint()).TextSize(12).TextColor(theme.Muted)
			})},
		}}).
		Page(kit.SettingPage{Title: "Appearance", Icon: kit.IconComputer, Groups: []kit.SettingGroup{
			{Title: "Appearance", Items: []kit.SettingItem{
				{Label: "Appearance", Description: "Follow your Mac, or choose a light or dark appearance.", Control: compactSettingsSelect(v.theme), Keywords: []string{"theme", "system", "light", "dark", "外观", "主题"}},
				{Label: "Text size", Description: "Adjust the size of text in the launcher.", Control: compactSettingsSelect(v.font), Keywords: []string{"font", "文字", "字体"}},
			}},
			{Title: "Window", Items: []kit.SettingItem{
				{Label: "Launcher material", Description: "Choose how the launcher and settings blend with your desktop.", Control: compactSettingsSelect(v.material), Keywords: []string{"glass", "liquid", "frosted", "blur", "玻璃", "透明"}},
			}},
		}}).
		Page(kit.SettingPage{Title: "Advanced", Icon: kit.IconReceipt, Groups: []kit.SettingGroup{
			{Title: "Advanced", Items: []kit.SettingItem{
				{Label: "Terminal application", Description: "The app used to open terminal commands.", Control: &settingsInput{v.inputs["terminal"], 132}, Keywords: []string{"shell", "scripts", "终端", "脚本"}},
			}},
		}})
	v.pages.SetValue(page)
	v.pages.SetQuery(query)
	v.status = ""
	v.dirty = false
}
func (v *settingsView) changed() {
	v.dirty = true
	v.status = ""
}
func (v *settingsView) save() {
	if v.busy {
		return
	}
	updated := v.config
	updated.GlobalShortcut = v.shortcut.value
	if _, _, err := core.ParseShortcut(updated.GlobalShortcut); err != nil {
		v.pages.SetQuery("")
		v.pages.SetValue("General")
		v.shortcut.error = "Record a valid shortcut."
		v.status = err.Error()
		return
	}
	updated.TerminalApp = strings.TrimSpace(v.inputs["terminal"].Value())
	updated.ThemeMode = v.theme.Value()
	updated.WindowMaterial = v.material.Value()
	updated.FontSize = v.font.Value()
	updated.LaunchAtLogin = v.login.Value()
	updated.ShowInMenuBar = v.tray.Value()
	v.busy = true
	v.status = "Saving…"
	go func() {
		// Preserve pins changed in the launcher since this settings window opened.
		if current, err := v.manager.services.Settings.GetSettings(); err == nil {
			updated.PinnedCommands = current.PinnedCommands
			updated.RecentCommands = current.RecentCommands
			updated.DisabledCommands = current.DisabledCommands
			updated.CommandHotkeys = current.CommandHotkeys
		}
		saved, err := v.manager.services.Settings.SaveSettings(updated)
		core.Update(func() {
			v.busy = false
			if err != nil {
				v.status = err.Error()
				return
			}
			v.config = *saved
			v.dirty = false
			v.manager.applyTheme()
			v.status = "Settings saved"
		})
	}()
}
func (v *settingsView) Render(cx *el.Context) el.Element {
	p := launcherPalette(v.manager.dark)
	if v.manager.dark {
		p.Bg = theme.RGB(0x171719)
		p.Subtle = theme.RGB(0x242426)
		p.Surface = theme.RGB(0x29292c)
		p.Border = theme.RGB(0x343437)
	}
	return cx.Themed(p, el.ViewFunc(v.render))
}

func (v *settingsView) render(cx *el.Context) el.Element {
	if v.shortcut.recording && cx.Focused(shortcutRecorderID) {
		v.shortcut.focusPending = false
	}
	if v.shortcut.recording && ((!v.shortcut.focusPending && !cx.Focused(shortcutRecorderID)) || (v.manager.settingsWindow != nil && !v.manager.settingsWindow.Focused() && os.Getenv("KEEL_HEADLESS") != "1")) {
		v.shortcut.stop()
	}
	width, _ := cx.ViewportSize()
	if w := v.manager.settingsWindow; w != nil {
		w.TitleBarArea(0, 0, width, 52)
	}
	header := el.Div().H(el.Dp(52)).NoShrink().Center().
		Child(el.Text("Settings").TextSize(13).Medium().TextColor(theme.Muted))
	status := ""
	if v.dirty {
		status = "Unsaved changes"
	}
	if v.status != "" {
		status = v.status
	}
	save := kit.Button("Save settings", v.save).Size(30).Loading(v.busy)
	save.SetDisabled(!v.dirty || v.busy)
	revert := kit.Button("Revert", v.load).Variant(kit.ButtonGhost).Size(30)
	revert.SetDisabled(!v.dirty || v.busy)
	footer := el.Div().H(el.Dp(60)).NoShrink().Row().Items(el.Center).Px(24).Gap(12).
		Child(el.Text(status).TextSize(12).TextColor(theme.Muted).MaxLines(2).Grow().W(el.Dp(0)), revert.Render(cx), keycap("⌘ S"), save.Render(cx))
	bottom := el.Div().H(el.Dp(61)).NoShrink().Row().Items(el.Stretch)
	if width >= 800 {
		bottom.Child(el.Div().W(el.Dp(211)).NoShrink())
	}
	bottom.Child(el.Div().Grow().W(el.Dp(0)).Items(el.Stretch).Child(
		el.Div().H(el.Dp(1)).NoShrink().Bg(v.manager.hairline()), footer.Bg(v.footerBackground()),
	))
	root := el.Div().WFull().HFull().TextSize(13).Bg(v.manager.launcherBackground()).Rounded(24).Child(
		header,
		el.Div().Grow().H(el.Dp(0)).Items(el.Stretch).Disabled(v.busy).Child(v.pages.Render(cx)),
		bottom,
	)
	if width >= 800 {
		root.Child(el.Div().ID("settings-sidebar-divider").Absolute().Left(210).Top(0).Bottom(0).W(el.Dp(1)).Bg(theme.Border))
	}
	return root
}

func (v *settingsView) footerBackground() color.NRGBA {
	bg := theme.Bg
	if v.manager.nativeGlassVisible() {
		bg.A = 160
	}
	return bg
}

// Handle window chords before the settings tree's focused controls can consume
// them. While recording, the same keys belong exclusively to the recorder.
type settingsContent struct {
	view *settingsView
	root *el.RootWidget
}

func (*settingsContent) FillsWindow() bool { return true }
func (v *settingsContent) Layout(gtx core.C) core.D {
	if !v.view.shortcut.recording {
		for {
			event, ok := gtx.Event(key.Filter{Name: "S", Required: key.ModCommand}, key.Filter{Name: key.NameEscape})
			if !ok {
				break
			}
			if e, ok := event.(key.Event); ok && e.State == key.Press {
				core.Call(gtx, func() {
					if e.Name == "S" {
						v.view.save()
					} else if w := v.view.manager.settingsWindow; w != nil {
						w.Close()
					}
				})
			}
		}
	}
	return v.root.Layout(gtx)
}
