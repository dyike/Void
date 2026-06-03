package window

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

type trayPoint struct {
	x float64
	y float64
}

func buildTrayTemplateIcon() []byte {
	const (
		size  = 32
		scale = 4
		hi    = size * scale
	)

	alpha := make([]uint8, hi*hi)
	fillTrayPolygon(alpha, hi, scale, []trayPoint{
		{7.0, 6.5}, {12.0, 6.5}, {16.3, 22.2}, {13.2, 28.0},
	})
	fillTrayPolygon(alpha, hi, scale, []trayPoint{
		{20.0, 6.5}, {25.0, 6.5}, {18.8, 28.0}, {15.7, 22.2},
	})
	fillTrayPolygon(alpha, hi, scale, []trayPoint{
		{15.1, 3.8}, {18.1, 3.8}, {20.1, 6.0}, {17.6, 8.4}, {14.7, 8.4}, {12.6, 6.0},
	})

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var sum int
			for yy := 0; yy < scale; yy++ {
				for xx := 0; xx < scale; xx++ {
					sum += int(alpha[(y*scale+yy)*hi+(x*scale+xx)])
				}
			}
			a := uint8(sum / (scale * scale))
			img.SetRGBA(x, y, color.RGBA{A: a})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

func fillTrayPolygon(alpha []uint8, size int, scale int, points []trayPoint) {
	scaled := make([]trayPoint, len(points))
	for i, p := range points {
		scaled[i] = trayPoint{x: p.x * float64(scale), y: p.y * float64(scale)}
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if trayPointInPolygon(float64(x)+0.5, float64(y)+0.5, scaled) {
				alpha[y*size+x] = 255
			}
		}
	}
}

func trayPointInPolygon(x, y float64, points []trayPoint) bool {
	inside := false
	j := len(points) - 1
	for i := range points {
		yi := points[i].y
		yj := points[j].y
		if (yi > y) != (yj > y) {
			xi := points[i].x
			xj := points[j].x
			if x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}
