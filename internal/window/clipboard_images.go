package window

import (
	"context"
	"fmt"
	"image"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/keel/ui/core"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/kit"
	"github.com/dyike/keel/ui/theme"
	"golang.org/x/image/draw"
)

const clipboardArtworkLimit = 12

type clipboardArtwork struct {
	thumbnail, preview *kit.ImageView
	card               *kit.HoverCardView
	size               image.Point
	err                error
	loading            bool
	used               uint64
	cancel             context.CancelFunc
}

func clipboardImagePath(item model.ClipboardItem) string {
	if !filepath.IsAbs(item.Content) || strings.ContainsAny(item.Content, "\r\n") {
		return ""
	}
	if item.Type == "image" {
		return item.Content
	}
	if item.Type == "file" {
		switch strings.ToLower(filepath.Ext(item.Content)) {
		case ".png", ".jpg", ".jpeg", ".gif", ".webp":
			return item.Content
		}
	}
	return ""
}

// Only visible rows request artwork. Retain scaled pixels, with at most two
// concurrent decodes; a history of large screenshots must not retain originals.
func (v *launcherView) clipboardArtwork(path string) *clipboardArtwork {
	v.clipboardImageClock++
	if art := v.clipboardImages[path]; art != nil {
		art.used = v.clipboardImageClock
		return art
	}
	if len(v.clipboardImages) >= clipboardArtworkLimit {
		oldest := ""
		for key, art := range v.clipboardImages {
			if oldest == "" || art.used < v.clipboardImages[oldest].used {
				oldest = key
			}
		}
		v.clipboardImages[oldest].cancel()
		delete(v.clipboardImages, oldest)
	}
	ctx, cancel := context.WithCancel(context.Background())
	art := &clipboardArtwork{loading: true, used: v.clipboardImageClock, cancel: cancel}
	v.clipboardImages[path] = art
	go func() {
		defer cancel()
		select {
		case v.clipboardImageLoads <- struct{}{}:
			defer func() { <-v.clipboardImageLoads }()
		case <-ctx.Done():
			return
		}
		thumb, preview, size, err := loadClipboardArtwork(ctx, path)
		if ctx.Err() != nil {
			return
		}
		core.Update(func() {
			if v.clipboardImages[path] != art {
				return
			}
			art.loading, art.err, art.size = false, err, size
			if err == nil {
				art.thumbnail = kit.Image(thumb, "Clipboard image thumbnail").Size(36, 36).Fit(kit.ImageContain).Rounded(4)
				art.preview = kit.Image(preview, "Clipboard image preview").Fit(kit.ImageContain).Rounded(6)
			}
		})
	}()
	return art
}

func loadClipboardArtwork(ctx context.Context, path string) (image.Image, image.Image, image.Point, error) {
	// A file URL preserves literal '%' and '#' in local filenames.
	source := (&url.URL{Scheme: "file", Path: path}).String()
	img, err := core.DecodeImage(ctx, source)
	if err != nil {
		return nil, nil, image.Point{}, err
	}
	size := img.Bounds().Size()
	return fitClipboardImage(img, 72, 72), fitClipboardImage(img, 960, 640), size, nil
}

func fitClipboardImage(src image.Image, width, height int) image.Image {
	b := src.Bounds()
	scale := min(1, float64(width)/float64(b.Dx()), float64(height)/float64(b.Dy()))
	size := image.Pt(max(1, int(float64(b.Dx())*scale)), max(1, int(float64(b.Dy())*scale)))
	dst := image.NewNRGBA(image.Rectangle{Max: size})
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return dst
}

func (v *launcherView) pruneClipboardImages(rows []resultRow) {
	keep := make(map[string]bool, len(rows))
	for _, row := range rows {
		keep[row.imagePath] = true
	}
	for path, art := range v.clipboardImages {
		if !keep[path] {
			art.cancel()
			delete(v.clipboardImages, path)
		}
	}
}

func (v *launcherView) clipboardThumbnail(cx *el.Context, row resultRow) el.Element {
	art := v.clipboardArtwork(row.imagePath)
	thumbnail := el.ViewFunc(func(cx *el.Context) el.Element {
		box := el.Div().Size(el.Dp(36)).Rounded(5).Bg(theme.Subtle).Center()
		if art.thumbnail != nil {
			box.Child(art.thumbnail.Render(cx))
		} else {
			box.Child(kit.Icon(kit.IconFile).Size(18).Color(theme.Muted).Render(cx))
		}
		return box
	})
	if art.card == nil {
		art.card = kit.HoverCard(thumbnail, el.ViewFunc(func(cx *el.Context) el.Element {
			return v.clipboardImageContent(cx, row.imagePath, 332, 230)
		})).Width(380).Placement(el.Right, el.Center).Offset(10).
			OpenDelay(300 * time.Millisecond).CloseDelay(120 * time.Millisecond)
	}
	selected := v.selected >= 0 && v.selected < len(v.rows) && v.rows[v.selected].id == row.id
	art.card.SetDisabled(!selected || v.actions || v.clipboardPreview.Value())
	if !selected || v.actions || v.clipboardPreview.Value() {
		return thumbnail.Render(cx)
	}
	return el.Div().W(el.Dp(36)).Child(art.card.Render(cx))
}

func (v *launcherView) clipboardImageContent(cx *el.Context, path string, width, height float32) el.Element {
	art := v.clipboardArtwork(path)
	w, h := cx.ViewportSize()
	width, height = min(width, max(1, w-96)), min(height, max(1, h-180))
	box := el.Div().Gap(8).W(el.Dp(width)).Role("group").Name("Image preview")
	switch {
	case art.loading:
		box.Child(el.Div().H(el.Dp(height)).Center().Child(el.Text("Loading image…").TextColor(theme.Muted)))
	case art.err != nil:
		box.Child(el.Text("Image unavailable").Medium(), el.Text("The image may have been moved, deleted, or could not be read.").TextSize(12).TextColor(theme.Muted),
			kit.Button("Retry image", func() { art.cancel(); delete(v.clipboardImages, path) }).Variant(kit.ButtonGhost).Render(cx))
	default:
		box.Child(el.Div().Rounded(6).Bg(theme.Subtle).Child(art.preview.Size(width, height).Render(cx)),
			el.Text(fmt.Sprintf("%d × %d", art.size.X, art.size.Y)).TextSize(11).TextColor(theme.Muted))
	}
	return box.Child(el.Text(filepath.Base(path)).TextSize(11).TextColor(theme.Muted).MaxLines(1))
}

func (v *launcherView) previewClipboardImage(path string) {
	v.clipboardPreview.Body(el.ViewFunc(func(cx *el.Context) el.Element {
		return v.clipboardImageContent(cx, path, 552, 340)
	}))
	v.clipboardPreview.SetValue(true)
}
