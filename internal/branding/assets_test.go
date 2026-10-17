package branding

import (
	"bytes"
	"image/png"
	"testing"
)

func TestAppArtworkIsFullBleed(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(AppIcon))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 1024 || img.Bounds().Dy() != 1024 {
		t.Fatalf("app icon size: %v", img.Bounds())
	}
	// The packager applies the native mask. An already rounded image would
	// produce a second margin around the Dock icon.
	for _, p := range [][2]int{{0, 0}, {1023, 0}, {0, 1023}, {1023, 1023}} {
		if _, _, _, a := img.At(p[0], p[1]).RGBA(); a != 65535 {
			t.Fatal("app artwork must be full bleed")
		}
	}
}

func TestTrayArtworkIsATemplate(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(TrayIcon))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 36 || img.Bounds().Dy() != 36 {
		t.Fatalf("18pt @2x template size: %v", img.Bounds())
	}
	opaque, partial := 0, 0
	for y := 0; y < 36; y++ {
		for x := 0; x < 36; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if r != 0 || g != 0 || b != 0 {
				t.Fatal("template must contain black alpha-mask pixels only")
			}
			if a > 0 {
				opaque++
			}
			if a > 0 && a < 65535 {
				partial++
			}
			if (x < 3 || y < 3 || x >= 33 || y >= 33) && a != 0 {
				t.Fatal("template needs transparent margins")
			}
		}
	}
	if opaque < 180 || opaque > 500 {
		t.Fatalf("template lost its silhouette: %d pixels", opaque)
	}
	if partial < 20 {
		t.Fatal("template edges must be antialiased")
	}
	if _, _, _, a := img.At(18, 18).RGBA(); a != 0 {
		t.Fatal("central opening must remain transparent")
	}
}
