package commands

import (
	"fmt"
	"os/exec"
)

// ─── Window Management ─────────────────────────────────────────────────────

// moveWindow positions the frontmost window using fractional screen coordinates.
// xFrac, yFrac = position as fraction of screen; wFrac, hFrac = size as fraction.
func moveWindow(xFrac, yFrac, wFrac, hFrac float64) error {
	script := fmt.Sprintf(`
tell application "Finder"
	set _b to bounds of window of desktop
	set _sw to (item 3 of _b) - (item 1 of _b)
	set _sh to (item 4 of _b) - (item 2 of _b)
end tell
set _x to round(%f * _sw)
set _y to round(%f * _sh)
set _w to round(%f * _sw)
set _h to round(%f * _sh)
tell application "System Events"
	set frontApp to first application process whose frontmost is true
	tell window 1 of frontApp
		set position to {_x, _y}
		set size to {_w, _h}
	end tell
end tell
`, xFrac, yFrac, wFrac, hFrac)
	return exec.Command("osascript", "-e", script).Run()
}

// centerWindow centers the frontmost window on screen at 60%% width, 80%% height.
func centerWindow() error {
	script := `
tell application "Finder"
	set _b to bounds of window of desktop
	set _sw to (item 3 of _b) - (item 1 of _b)
	set _sh to (item 4 of _b) - (item 2 of _b)
end tell
set _w to round(0.6 * _sw)
set _h to round(0.8 * _sh)
set _x to round((_sw - _w) / 2)
set _y to round((_sh - _h) / 2)
tell application "System Events"
	set frontApp to first application process whose frontmost is true
	tell window 1 of frontApp
		set position to {_x, _y}
		set size to {_w, _h}
	end tell
end tell
`
	return exec.Command("osascript", "-e", script).Run()
}
