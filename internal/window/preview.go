package window

import (
	"github.com/dyike/keel/ui/core"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/theme"
	"github.com/dyike/keel/ui/window"
)

// OpenPreviewBackdrop provides a controlled surface for real compositor checks.
// It is opened only by the explicit material diagnostics flag.
func (m *Manager) OpenPreviewBackdrop() {
	window.Open(window.Options{Title: "Void material reference", Width: 1100, Height: 800, Shortcuts: map[string]func(){"mod+0": m.ShowLauncher}, OnResize: func(int, int) { core.Update(m.ShowLauncher) }, Content: el.Root(el.ViewFunc(func(*el.Context) el.Element {
		row := el.Div().WFull().HFull().Row()
		colors := []uint32{0x19356d, 0x25b5b4, 0xee8b74, 0x993cb2}
		for i, c := range colors {
			row.Child(el.Div().Grow().HFull().BgGradient(theme.Gradient{From: theme.RGB(c), To: theme.RGB(colors[(i+1)%len(colors)]), Angle: 90}).Center().Child(el.Text("Aa").TextSize(100).Bold().TextColor(theme.RGB(0xffffff))))
		}
		return row
	}))})
}
