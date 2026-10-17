// Command icons rasterizes the editable SVG artwork. It runs at development
// time; the application embeds the resulting PNGs without runtime rasterization.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func main() {
	root := flag.String("root", ".", "project root")
	flag.Parse()
	for _, asset := range []struct {
		name string
		size int
	}{{"app", 1024}, {"tray", 36}} {
		base := filepath.Join(*root, "internal", "branding", asset.name)
		if err := render(base+".svg", base+".png", asset.size); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
func render(source, destination string, size int) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	icon, err := oksvg.ReadIconStream(f, oksvg.StrictErrorMode)
	f.Close()
	if err != nil {
		return err
	}
	icon.SetTarget(0, 0, float64(size), float64(size))
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	icon.Draw(rasterx.NewDasher(size, size, rasterx.NewScannerGV(size, size, img, img.Bounds())), 1)
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	err = encoder.Encode(out, img)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}
