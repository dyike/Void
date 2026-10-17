package window

import (
	"testing"

	"github.com/dyike/keel/gio/io/key"
	"github.com/dyike/keel/ui/el"
)

func TestRecordedShortcutMatchesNativeKeys(t *testing.T) {
	for _, tc := range []struct {
		name key.Name
		mods key.Modifiers
		want string
	}{
		{key.NameSpace, key.ModAlt, "option+space"},
		{"K", key.ModCommand | key.ModShift, "shift+cmd+k"},
		{"!", key.ModCtrl | key.ModShift, "ctrl+shift+1"},
		{"+", key.ModCommand | key.ModShift, "shift+cmd+="},
		{key.NameUpArrow, key.ModCtrl | key.ModAlt, "ctrl+option+up"},
		{key.NameF12, key.ModCommand, "cmd+f12"},
	} {
		got, err := recordedShortcut(tc.name, tc.mods)
		if err != nil || got != tc.want {
			t.Errorf("%q %v = %q, %v; want %q", tc.name, tc.mods, got, err, tc.want)
		}
	}
	for _, tc := range []struct {
		name key.Name
		mods key.Modifiers
	}{
		{"A", 0}, {"A", key.ModShift}, {"F20", key.ModCommand}, {"", key.ModCommand},
	} {
		if got, err := recordedShortcut(tc.name, tc.mods); err == nil {
			t.Errorf("accepted unsupported shortcut %q", got)
		}
	}
}

func TestShortcutRecordingCommitAndCancel(t *testing.T) {
	var paused []bool
	changes := 0
	r := shortcutRecorder{value: "option+space", changed: func() { changes++ }, recordingChanged: func(on bool) { paused = append(paused, on) }}
	r.start()
	r.onKey(el.KeyEvent{Name: "⌘", Modifiers: key.ModCommand})
	r.onKey(el.KeyEvent{Name: "S", Modifiers: key.ModCommand})
	if r.recording || r.value != "cmd+s" || changes != 1 {
		t.Fatalf("recording did not commit: %+v", r)
	}
	r.start()
	r.onKey(el.KeyEvent{Name: "A"})
	if !r.recording || r.error == "" {
		t.Fatal("plain key should remain recording with guidance")
	}
	r.onKey(el.KeyEvent{Name: string(key.NameEscape)})
	if r.recording || r.value != "cmd+s" || changes != 1 {
		t.Fatal("cancel changed shortcut")
	}
	r.start()
	r.onKey(el.KeyEvent{Name: "!", Modifiers: key.ModCtrl | key.ModShift})
	r.onKey(el.KeyEvent{Name: "1", State: el.KeyRelease})
	if r.recording || r.value != "ctrl+shift+1" {
		t.Fatal("shifted punctuation must normalize to a native key")
	}
	r.start()
	r.stop()
	r.stop()
	if len(paused) != 8 {
		t.Fatalf("pause/resume must balance, got %v", paused)
	}
	for i, v := range paused {
		if v != (i%2 == 0) {
			t.Fatalf("unbalanced pause sequence: %v", paused)
		}
	}
}
