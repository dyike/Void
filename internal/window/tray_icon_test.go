package window

import (
	"bytes"
	"image/png"
	"testing"
)

func TestBuildTrayTemplateIcon(t *testing.T) {
	icon := buildTrayTemplateIcon()
	if len(icon) == 0 {
		t.Fatal("expected tray icon bytes")
	}

	img, err := png.Decode(bytes.NewReader(icon))
	if err != nil {
		t.Fatalf("expected valid PNG: %v", err)
	}
	if img.Bounds().Dx() != 32 || img.Bounds().Dy() != 32 {
		t.Fatalf("expected 32x32 icon, got %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}

	var opaque int
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				opaque++
			}
		}
	}
	if opaque < 120 || opaque > 420 {
		t.Fatalf("unexpected tray icon coverage: %d pixels", opaque)
	}
}
