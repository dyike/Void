package window

import (
	"image"
	"image/color"
	"testing"
	"time"

	"github.com/dyike/keel/gio/gpu/headless"
	"github.com/dyike/keel/gio/io/input"
	"github.com/dyike/keel/gio/layout"
	"github.com/dyike/keel/gio/op"
	"github.com/dyike/keel/gio/unit"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/theme"
	keelwindow "github.com/dyike/keel/ui/window"
)

// Exercise the actual launcher tree: editorstyle's caret can be valid yet
// disappear under the input element's own clip when its padding is removed.
func TestLauncherEmptyCaretAndTransparentContent(t *testing.T) {
	if !keelwindow.GlassSupported() {
		t.Skip("native glass unsupported")
	}
	t.Setenv("KEEL_HEADLESS", "")
	previous := theme.Current()
	defer theme.Apply(previous)
	for _, dark := range []bool{false, true} {
		palette := launcherPalette(dark)
		palette.Text = color.NRGBA{R: 255, A: 255} // distinguish caret from the muted hint
		theme.Apply(palette)
		manager := NewManager()
		manager.dark, manager.fontSize = dark, 14
		view := newLauncher(manager)
		view.focusSearch = true
		root := el.Root(view)
		size := image.Pt(1520, 1120)
		renderer, err := headless.NewWindow(size.X, size.Y)
		if err != nil {
			t.Fatal(err)
		}
		var ops op.Ops
		var router input.Router
		now := time.Now()
		for range 4 {
			ops.Reset()
			gtx := layout.Context{Ops: &ops, Now: now, Source: router.Source(), Constraints: layout.Exact(size), Metric: unit.Metric{PxPerDp: 2, PxPerSp: 2}}
			root.Layout(gtx)
			router.Frame(&ops)
		}
		if err := renderer.Frame(&ops); err != nil {
			renderer.Release()
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rectangle{Max: size})
		err = renderer.Screenshot(img)
		renderer.Release()
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range []image.Point{{1400, 60}, {1000, 600}, {600, 1080}} {
			if pixel := img.RGBAAt(p.X, p.Y); pixel.A != 0 {
				t.Errorf("dark=%v: native backdrop covered at %v by %v", dark, p, pixel)
			}
		}
		caretPixels := 0
		for y := 20; y < 110; y++ {
			for x := 90; x < 140; x++ {
				p := img.RGBAAt(x, y)
				if p.R > 180 && p.G < 80 && p.B < 80 && p.A > 180 {
					caretPixels++
				}
			}
		}
		if caretPixels < 30 {
			t.Errorf("dark=%v: empty focused search has only %d visible caret pixels", dark, caretPixels)
		}
	}
}
