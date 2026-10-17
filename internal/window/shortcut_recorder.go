package window

import (
	"errors"
	"strings"

	"github.com/dyike/Void/internal/repo"
	"github.com/dyike/keel/gio/io/key"
	"github.com/dyike/keel/ui/core"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/theme"
)

const shortcutRecorderID = "global-shortcut-recorder"

type shortcutRecorder struct {
	value, error     string
	recording        bool
	focusPending     bool
	changed          func()
	recordingChanged func(bool)
}

func (r *shortcutRecorder) start() {
	r.error = ""
	r.recording = true
	r.focusPending = true
	if r.recordingChanged != nil {
		r.recordingChanged(true)
	}
}

func (r *shortcutRecorder) stop() {
	if !r.recording {
		return
	}
	r.recording = false
	r.focusPending = false
	r.error = ""
	if r.recordingChanged != nil {
		r.recordingChanged(false)
	}
}

func (r *shortcutRecorder) hint() string {
	if r.error != "" {
		return r.error
	}
	if r.recording {
		return "Press a key combination. Esc to cancel."
	}
	return "Click the shortcut, then press a key combination. Save to apply."
}

func (r *shortcutRecorder) onKey(e el.KeyEvent) bool {
	if !r.recording {
		return false
	}
	if e.State == el.KeyRelease {
		return true
	}
	if key.Name(e.Name) == key.NameEscape && e.Modifiers == 0 {
		r.stop()
		return true
	}
	switch key.Name(e.Name) {
	case key.NameCtrl, key.NameCommand, key.NameAlt, key.NameShift, key.NameSuper:
		return true
	}
	spec, err := recordedShortcut(key.Name(e.Name), e.Modifiers)
	if err != nil {
		r.error = err.Error()
		return true
	}
	r.stop()
	if r.value != spec {
		r.value = spec
		if r.changed != nil {
			r.changed()
		}
	}
	return true
}

func (r *shortcutRecorder) Render(cx *el.Context) el.Element {
	label := core.ShortcutLabel(r.value, "darwin")
	if r.recording {
		label = "Press shortcut…"
	}
	border, bg := theme.Border, theme.Surface
	if r.recording {
		border, bg = theme.Primary, theme.Subtle
	}
	return el.Div().ID(shortcutRecorderID).Name("Global shortcut").Role("button").Value(label).
		W(el.Dp(156)).MaxW(el.Full).H(el.Dp(30)).Px(10).Center().
		Border(1, border).Rounded(6).Bg(bg).Focusable(true).
		OnClick(func() {
			if r.recording {
				r.stop()
			} else {
				r.start()
				cx.Focus(shortcutRecorderID)
			}
		}).
		OnKey(r.onKey).Child(el.Text(label).TextSize(13).MaxLines(1))
}

func recordedShortcut(name key.Name, mods key.Modifiers) (string, error) {
	if mods&(key.ModCommand|key.ModAlt|key.ModCtrl) == 0 {
		return "", errors.New("Hold Command, Option or Control and press another key.")
	}
	var parts []string
	for _, item := range []struct {
		mod  key.Modifiers
		name string
	}{
		{key.ModCtrl, "ctrl"}, {key.ModAlt, "option"}, {key.ModShift, "shift"}, {key.ModCommand, "cmd"},
	} {
		if mods.Contain(item.mod) {
			parts = append(parts, item.name)
		}
	}
	parts = append(parts, shortcutKey(name))
	spec := strings.Join(parts, "+")
	if _, err := repo.NewHotkeyRepo().ParseHotkey(spec); err != nil {
		return "", errors.New("That key isn't supported. Try another combination.")
	}
	return spec, nil
}

func shortcutKey(name key.Name) string {
	switch name {
	case key.NameSpace:
		return "space"
	case key.NameReturn, key.NameEnter:
		return "return"
	case key.NameEscape:
		return "escape"
	case key.NameTab:
		return "tab"
	case key.NameDeleteBackward:
		return "backspace"
	case key.NameLeftArrow:
		return "left"
	case key.NameRightArrow:
		return "right"
	case key.NameUpArrow:
		return "up"
	case key.NameDownArrow:
		return "down"
	case key.NameHome:
		return "home"
	case key.NameEnd:
		return "end"
	case key.NamePageUp:
		return "pageup"
	case key.NamePageDown:
		return "pagedown"
	}
	// Gio reports shifted punctuation, whereas the native hotkey uses the
	// underlying ANSI key code plus its separate Shift modifier.
	shifted := "~!@#$%^&*()_+{}|:\"<>?"
	plain := "`1234567890-=[]\\;',./"
	if runes := []rune(name); len(runes) == 1 {
		if i := strings.IndexRune(shifted, runes[0]); i >= 0 {
			return string(plain[i])
		}
	}
	return strings.ToLower(string(name))
}
