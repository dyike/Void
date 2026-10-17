package window

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/color"
	"image/png"
	"strings"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/keel/gio/io/key"
	"github.com/dyike/keel/ui/core"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/kit"
	"github.com/dyike/keel/ui/theme"
)

const launcherQueryID = "launcher-query"

func commandIcon(c model.CommandInfo) kit.IconName {
	switch c.ID {
	case "system-clipboard-manager":
		return kit.IconCopy
	case "system-snippets":
		return kit.IconFile
	case "system-scripts", "system-open-terminal":
		return kit.IconReceipt
	case "system-apple-containers":
		return kit.IconArchive
	case "system-file-search", "system-file-contents":
		return kit.IconSearch
	case "system-lock-screen":
		return kit.IconLock
	case "system-empty-trash":
		return kit.IconTrash
	case "system-void-settings":
		return kit.IconSettings
	}
	switch c.Category {
	case "app":
		return kit.IconComputer
	case "settings":
		return kit.IconSettings
	case "file":
		return kit.IconFile
	}
	return kit.IconComputer
}
func categoryLabel(category string) string {
	switch category {
	case "app":
		return "Application"
	case "system":
		return "Command"
	case "settings":
		return "System Settings"
	case "file":
		return "File"
	case "container":
		return "Container"
	}
	return category
}
func (v *launcherView) rowIcon(cx *el.Context, row resultRow) el.Element {
	if row.imagePath != "" {
		return v.clipboardThumbnail(cx, row)
	}
	if row.iconData != "" {
		img := v.iconImages[row.iconData]
		if img == nil {
			encoded := strings.TrimPrefix(row.iconData, "data:image/png;base64,")
			if data, err := base64.StdEncoding.DecodeString(encoded); err == nil {
				if artwork, err := png.Decode(bytes.NewReader(data)); err == nil {
					img = kit.Image(artwork, "").Size(28, 28).Rounded(6)
					v.iconImages[row.iconData] = img
				}
			}
		}
		if img != nil {
			return img.Render(cx)
		}
	}
	icon := row.icon
	if icon == kit.IconNone {
		icon = kit.IconArchive
	}
	tint := theme.PrimaryText
	fill := theme.Primary
	fill.A = 24
	return el.Div().Size(el.Dp(28)).Rounded(7).Center().Bg(fill).Child(kit.Icon(icon).Size(17).Color(tint).Render(cx))
}
func (v *launcherView) renderRow(cx *el.Context, i int) el.Element {
	row := v.rows[i]
	id := "result-" + row.id
	if cx.Hovered(id) && v.hoveredID != id {
		v.hoveredID = id
		v.selected = i
	}
	selected := i == v.selected
	box := el.Div().ID(id).H(el.Dp(48)).Px(12).Row().Items(el.Center).Gap(10).Rounded(8).
		OnClick(func() { v.selected = i; row.run() }).Role("button").Name(row.title).Value(row.subtitle).Selected(selected).
		Hover(func(s *el.Style) { s.Bg(v.manager.hoverFill()) })
	if selected {
		box.Bg(theme.Highlight)
	}
	details := el.Div().Grow().MinW(el.Dp(0)).Gap(2).Child(el.Text(row.title).TextSize(v.manager.fontSize - 1).Medium().MaxLines(1))
	if row.subtitle != "" {
		details.Child(el.Text(row.subtitle).TextSize(11).TextColor(theme.Muted).MaxLines(1))
	}
	box.Child(v.rowIcon(cx, row), details)
	label := categoryLabel(row.category)
	if label != "" {
		box.Child(el.Text(label).TextSize(11).TextColor(theme.Muted).MaxLines(1))
	}
	if selected {
		box.Child(el.Text("↵").TextSize(13).TextColor(theme.Muted).W(el.Dp(14)))
	} else {
		box.Child(el.Div().W(el.Dp(14)))
	}
	return box
}
func (v *launcherView) toggleActions() {
	if v.form != "" || v.loading || len(v.rows) == 0 {
		return
	}
	v.actions = !v.actions
	v.actionIndex = 0
	v.actionFocus = v.actions
}
func (v *launcherView) actionRows() []action {
	if v.selected < 0 || v.selected >= len(v.rows) {
		return nil
	}
	row := v.rows[v.selected]
	label := "Open"
	if v.page == "Clipboard" || v.page == "Snippets" || row.id == "calculator" {
		label = "Copy"
	}
	return append([]action{{label, row.run}}, row.actions...)
}
func (v *launcherView) showActions(cx *el.Context) {
	if !v.actions {
		return
	}
	actions := v.actionRows()
	if len(actions) == 0 {
		v.actions = false
		return
	}
	v.actionIndex = min(max(v.actionIndex, 0), len(actions)-1)
	panel := el.Div().W(el.Dp(310)).P(8).Gap(4).Rounded(12).Bg(theme.Surface).Border(1, v.manager.hairline()).
		Child(el.Div().Px(10).Py(8).Gap(3).Child(el.Text(v.rows[v.selected].title).TextSize(13).Medium().MaxLines(1), el.Text("Actions").TextSize(11).TextColor(theme.Muted)), el.Div().H(el.Dp(1)).Bg(v.manager.hairline()))
	for i, a := range actions {
		id := fmt.Sprintf("launcher-action-%d", i)
		row := el.Div().ID(id).Row().Items(el.Center).Gap(8).H(el.Dp(34)).Px(10).Rounded(6).Focusable(true).
			Role("button").Name(a.title).OnClick(func() { v.actions = false; a.run() }).OnKey(func(e el.KeyEvent) bool {
			if e.State != el.KeyPress {
				return false
			}
			if e.Name == string(key.NameDownArrow) {
				v.actionIndex = (v.actionIndex + 1) % len(actions)
				v.actionFocus = true
				return true
			}
			if e.Name == string(key.NameUpArrow) {
				v.actionIndex = (v.actionIndex + len(actions) - 1) % len(actions)
				v.actionFocus = true
				return true
			}
			return false
		}).Hover(func(s *el.Style) { s.Bg(v.manager.hoverFill()) })
		if i == v.actionIndex {
			row.Bg(theme.Highlight)
		}
		row.Child(el.Text(a.title).TextSize(13).Grow())
		if i == 0 {
			row.Child(keycap("↵"))
		}
		panel.Child(row)
	}
	cx.Overlay("launcher-actions", el.Anchored("actions-button", panel).Placement(el.Top, el.End).Offset(10).Modal().TrapFocus().Scrim(false).OnDismiss(func() { v.actions = false }))
	if v.actionFocus {
		cx.Focus(fmt.Sprintf("launcher-action-%d", v.actionIndex))
		v.actionFocus = false
	}
}
func keycap(label string) el.Element {
	return el.Text(label).TextSize(11).TextColor(theme.Muted).Px(5).Py(2).Rounded(4).Bg(theme.Highlight)
}
func (v *launcherView) quietButton(cx *el.Context, id, label string, icon kit.IconName, fn func(), content ...el.Element) el.Element {
	button := el.Div().ID(id).Row().Items(el.Center).Justify(el.Center).Gap(6).H(el.Dp(28)).Px(8).Rounded(6).Focusable(true).Role("button").Name(label).OnClick(fn).
		Hover(func(s *el.Style) { s.Bg(v.manager.hoverFill()) })
	if icon != kit.IconNone {
		button.Child(kit.Icon(icon).Size(15).Color(theme.Muted).Render(cx))
	}
	return button.Child(content...)
}
func (v *launcherView) searchHeader(cx *el.Context) el.Element {
	header := el.Div().Row().H(el.Dp(66)).Px(20).Items(el.Center).Gap(12)
	if v.page != "Commands" {
		header.Child(v.quietButton(cx, "back-button", "Back to commands", kit.IconChevronLeft, func() {
			if v.form != "" {
				v.form = ""
			} else {
				v.navigate("Commands")
			}
		}))
	} else {
		header.Child(kit.Icon(kit.IconSearch).Size(21).Color(theme.Muted).Render(cx))
	}
	if v.form != "" {
		return header.Child(el.Text(v.page).TextSize(17).Medium(), el.Div().Grow())
	}
	placeholder := "Search for apps and commands…"
	if v.page != "Commands" {
		placeholder = "Search " + strings.ToLower(v.page) + "…"
	}
	input := el.Input().ID(launcherQueryID).Name("Search").Bind(&v.query).Placeholder(placeholder).Border(0, color.NRGBA{}).Rounded(0).Bg(color.NRGBA{}).P(0).Px(5).Grow().MinW(el.Dp(0)).TextSize(20).
		FocusStyle(func(*el.Style) {}).OnChange(func(string) { v.refresh() }).OnSubmit(func(string) { v.activate() }).CaptureKeys(string(key.NameUpArrow), string(key.NameDownArrow)).
		OnKey(func(e el.KeyEvent) bool {
			if e.State != el.KeyPress {
				return false
			}
			switch e.Name {
			case string(key.NameUpArrow):
				v.move(-1)
				return true
			case string(key.NameDownArrow):
				v.move(1)
				return true
			}
			return false
		})
	header.Child(input)
	if v.query != "" {
		header.Child(v.quietButton(cx, "clear-search", "Clear search", kit.IconClose, func() { v.query = ""; v.refresh(); v.focusSearch = true }))
	}
	return header
}
func (v *launcherView) footer(cx *el.Context) el.Element {
	footer := el.Div().H(el.Dp(42)).Px(10).Row().Items(el.Center).Gap(8)
	footer.Child(v.quietButton(cx, "settings-button", "Settings", kit.IconSettings, v.manager.OpenSettings), el.Div().W(el.Dp(1)).H(el.Dp(14)).Bg(v.manager.hairline()))
	text := "Void"
	if v.page != "Commands" {
		text = v.page
	}
	if v.status != "" {
		text = v.status
	}
	footer.Child(el.Text(text).TextSize(11).TextColor(theme.Muted).MaxLines(1).Grow())
	if v.form == "" && len(v.rows) > 0 && !v.loading {
		label := "Open"
		if v.page == "Clipboard" || v.page == "Snippets" || v.rows[v.selected].id == "calculator" {
			label = "Copy"
		}
		footer.Child(v.quietButton(cx, "open-button", label, kit.IconNone, v.activate, el.Text(label).TextSize(12).TextColor(theme.Text), keycap("↵")), el.Div().W(el.Dp(1)).H(el.Dp(14)).Bg(v.manager.hairline()), v.quietButton(cx, "actions-button", "Actions", kit.IconNone, v.toggleActions, el.Text("Actions").TextSize(12).TextColor(theme.Text), keycap("⌘ K")))
	} else {
		footer.Child(el.Text("Esc to go back").TextSize(11).TextColor(theme.Muted))
	}
	return footer
}
func (v *launcherView) Render(cx *el.Context) el.Element {
	if w := v.manager.mainWindow; w != nil {
		if w.Focused() {
			if !v.hadFocus && v.form == "" && !v.actions && !v.dialog.Value() && !v.clipboardPreview.Value() {
				v.focusSearch = true
			}
			v.hadFocus = true
		} else if v.hadFocus && !v.dialog.Value() && !v.manager.preview {
			v.hadFocus = false
			// Native material installation can briefly move the input view out
			// and back. Do not apply an old blur to a refocused or replaced window.
			core.Update(func() {
				if v.manager.mainWindow == w && !w.Closed() && !w.Focused() && !v.dialog.Value() {
					v.manager.HideLauncher()
				}
			})
		}
		// The thin top edge remains draggable without stealing editor clicks.
		w.TitleBarArea(0, 0, 760, 7)
	}
	if v.focusSearch && v.form == "" {
		cx.Focus(launcherQueryID)
		v.focusSearch = false
	}
	body := el.Div().Grow().MinH(el.Dp(0))
	if v.form != "" {
		body.P(16).Child(v.renderForm(cx))
	} else {
		if v.page != "Commands" {
			body.Child(el.Div().Px(16).Py(8).Child(v.renderToolbar(cx)))
		}
		section := v.page
		if section == "Commands" {
			section = "Suggestions"
			if v.query != "" {
				section = "Results"
			}
		}
		body.Child(el.Div().Px(20).Pt(12).Pb(6).Child(el.Text(section).TextSize(11).TextColor(theme.Muted)))
		if v.loading {
			body.Child(el.Div().P(24).Child(el.Text("Searching…").TextSize(13).TextColor(theme.Muted)))
		}
		if !v.loading && len(v.rows) == 0 {
			body.Child(el.Div().Grow().Center().Gap(8).Child(kit.Icon(kit.IconSearch).Size(26).Color(theme.Muted).Render(cx), el.Text("No results").TextSize(14).TextColor(theme.Muted)))
		}
		if v.reveal {
			v.list.ScrollTo(cx, v.selected)
			v.reveal = false
		}
		body.Child(el.Div().Grow().MinH(el.Dp(0)).Px(8).Pb(6).Child(v.list.Render(cx)))
	}
	if v.output != "" {
		body.Child(el.Div().ID("output").H(el.Dp(150)).ScrollY().P(12).Bg(theme.Surface).Child(el.Text(v.output).Mono().TextSize(12)), kit.Button("Close output", func() { v.output = "" }).Variant(kit.ButtonGhost).Compact(true).Render(cx))
	}
	v.showActions(cx)
	border := v.manager.hairline()
	if v.manager.nativeGlassVisible() {
		border = color.NRGBA{}
	}
	return el.Div().WFull().HFull().Bg(v.manager.launcherBackground()).Border(1, border).Rounded(24).TextSize(v.manager.fontSize).
		Child(v.searchHeader(cx), el.Div().H(el.Dp(1)).Bg(v.manager.hairline()), body, el.Div().H(el.Dp(1)).Bg(v.manager.hairline()), v.footer(cx), v.dialog.Render(cx), v.clipboardPreview.Render(cx))
}
