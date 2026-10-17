package window

import (
	"image/color"
	"os"

	"github.com/dyike/keel/ui/theme"
	"github.com/dyike/keel/ui/window"
)

func launcherPalette(dark bool) theme.Palette {
	p := theme.Light()
	if dark {
		p = theme.Dark()
		p.Bg = theme.RGB(0x202024)
		p.Surface = theme.RGB(0x29292e)
		p.Text = theme.RGB(0xf4f4f5)
		p.Muted = theme.RGB(0xa1a1aa)
		p.Border = theme.RGB(0x3a3a40)
		p.Highlight = theme.RGB(0x484850)
		p.Highlight.A = 190
		p.Subtle = theme.RGB(0x343439)
		p.SubtleHover = theme.RGB(0x414147)
		p.Primary = theme.RGB(0x8b86ee)
		p.PrimaryHover = theme.RGB(0x9e99f5)
		p.PrimaryText = theme.RGB(0xc9c6ff)
	} else {
		p.Bg = theme.RGB(0xf5f5f7)
		p.Surface = theme.RGB(0xffffff)
		p.Text = theme.RGB(0x242428)
		p.Muted = theme.RGB(0x77777f)
		p.Border = theme.RGB(0xdcdce1)
		p.Highlight = theme.RGB(0xdedee6)
		p.Highlight.A = 180
		p.Subtle = theme.RGB(0xe9e9ed)
		p.SubtleHover = theme.RGB(0xdddde3)
		p.Primary = theme.RGB(0x6860c6)
		p.PrimaryHover = theme.RGB(0x5750b0)
		p.PrimaryText = theme.RGB(0x5750b0)
	}
	return p
}

func (m *Manager) launcherBackground() color.NRGBA {
	// Native glass is supplied by AppKit. Offscreen renders use a solid fallback.
	if m.screenshot || os.Getenv("KEEL_HEADLESS") == "1" || !window.GlassSupported() {
		return theme.Bg
	}
	return color.NRGBA{}
}
func (m *Manager) hairline() color.NRGBA {
	if m.dark {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 22}
	}
	return color.NRGBA{A: 24}
}
func (m *Manager) hoverFill() color.NRGBA {
	if m.dark {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 13}
	}
	return color.NRGBA{A: 9}
}

func materialStyle(name string) window.GlassStyle {
	switch name {
	case "regular":
		return window.GlassRegular
	case "clear":
		return window.GlassClear
	default:
		return window.GlassFrosted
	}
}
func (m *Manager) nativeGlassVisible() bool {
	return !m.screenshot && os.Getenv("KEEL_HEADLESS") != "1" && window.GlassSupported()
}
