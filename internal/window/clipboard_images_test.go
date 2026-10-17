package window

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/keel/gio/f32"
	"github.com/dyike/keel/gio/io/input"
	"github.com/dyike/keel/gio/io/key"
	"github.com/dyike/keel/gio/io/pointer"
	"github.com/dyike/keel/gio/layout"
	"github.com/dyike/keel/gio/op"
	"github.com/dyike/keel/gio/unit"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/kit"
)

func TestClipboardArtworkLoadsLocalFilesAndBoundsPixels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample #50%.png")
	img := image.NewNRGBA(image.Rect(0, 0, 2000, 1000))
	for y := range 1000 {
		for x := range 2000 {
			img.SetNRGBA(x, y, color.NRGBA{R: 200, G: 50, B: 80, A: 128})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, img)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	thumb, preview, size, err := loadClipboardArtwork(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if size != image.Pt(2000, 1000) || thumb.Bounds().Size() != image.Pt(72, 36) || preview.Bounds().Size() != image.Pt(960, 480) {
		t.Fatalf("unexpected sizes: original=%v thumbnail=%v preview=%v", size, thumb.Bounds(), preview.Bounds())
	}
	if got := color.NRGBAModel.Convert(preview.At(100, 100)).(color.NRGBA); got.A != 128 || got.R < 198 {
		t.Fatalf("lost translucent colors: %v", got)
	}
	if _, _, _, err := loadClipboardArtwork(context.Background(), path+".missing"); err == nil {
		t.Fatal("missing file did not return a preview error")
	}
	if err := os.WriteFile(path, []byte("invalid image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := loadClipboardArtwork(context.Background(), path); err == nil {
		t.Fatal("corrupt image did not return a preview error")
	}
	for _, tc := range []struct{ kind, path, want string }{
		{"image", path, path}, {"file", "/tmp/Photo.JPG", "/tmp/Photo.JPG"},
		{"file", "/tmp/a.png\n/tmp/b.png", ""}, {"file", "/tmp/readme.txt", ""},
		{"text", path, ""}, {"image", "https://example.com/a.png", ""},
	} {
		if got := clipboardImagePath(model.ClipboardItem{Type: tc.kind, Content: tc.path}); got != tc.want {
			t.Errorf("%s %q: got %q, want %q", tc.kind, tc.path, got, tc.want)
		}
	}
}

func TestClipboardThumbnailHoverAndKeyboardPreview(t *testing.T) {
	v := newLauncher(NewManager())
	v.manager.fontSize = 15
	v.page = "Clipboard"
	copied := 0
	v.rows = []resultRow{
		{id: "picture", title: "Sample image", imagePath: "/tmp/picture.png", run: func() { copied++ }},
		{id: "text", title: "Sample text", run: func() {}},
	}
	v.list.SetCount(len(v.rows))
	img := image.NewNRGBA(image.Rect(0, 0, 120, 60))
	v.clipboardImages[v.rows[0].imagePath] = &clipboardArtwork{
		thumbnail: kit.Image(img, "Clipboard image thumbnail").Size(36, 36),
		preview:   kit.Image(img, "Clipboard image preview"), size: image.Pt(120, 60), cancel: func() {},
	}
	root := el.Root(v)
	var ops op.Ops
	var router input.Router
	now := time.Unix(100, 0)
	frame := func() {
		ops.Reset()
		root.Layout(layout.Context{Ops: &ops, Now: now, Source: router.Source(), Constraints: layout.Exact(image.Pt(760, 560)), Metric: unit.Metric{PxPerDp: 1, PxPerSp: 1}})
		router.Frame(&ops)
	}
	find := func(name string) (image.Rectangle, bool) {
		var bounds image.Rectangle
		var walk func(input.SemanticNode)
		walk = func(n input.SemanticNode) {
			if n.Desc.Label == name {
				bounds = n.Desc.Bounds
			}
			for _, child := range n.Children {
				walk(child)
			}
		}
		for _, node := range router.AppendSemantics(nil) {
			walk(node)
		}
		return bounds, !bounds.Empty()
	}
	move := func(p image.Point) {
		router.Queue(pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: f32.Pt(float32(p.X), float32(p.Y))})
		frame()
		frame()
	}
	frame()
	frame()
	thumb, ok := find("Clipboard image thumbnail")
	if !ok || thumb.Dx() > 36 || thumb.Dy() > 36 {
		t.Fatalf("thumbnail missing or too large: %v", thumb)
	}
	move(thumb.Min.Add(thumb.Size().Div(2)))
	now = now.Add(200 * time.Millisecond)
	frame()
	if _, ok := find("Clipboard image preview"); ok {
		t.Fatal("preview opened before hover delay")
	}
	now = now.Add(150 * time.Millisecond)
	frame()
	frame()
	preview, ok := find("Clipboard image preview")
	if !ok || preview.Dx() < thumb.Dx()*4 || !preview.In(image.Rect(0, 0, 760, 560)) || copied != 0 {
		t.Fatalf("invalid hover preview: %v, copied=%d", preview, copied)
	}
	move(preview.Min.Add(preview.Size().Div(2)))
	now = now.Add(time.Second)
	frame()
	if _, ok := find("Clipboard image preview"); !ok {
		t.Fatal("moving onto the preview closed it")
	}
	// Keyboard selection must dismiss the old row's preview even if the
	// pointer stays over that row or its floating image.
	v.move(1)
	frame()
	frame()
	if _, ok := find("Clipboard image preview"); ok {
		t.Fatal("changing the selected row left its old image preview open")
	}
	v.move(-1)
	move(thumb.Min.Add(thumb.Size().Div(2)))
	now = now.Add(350 * time.Millisecond)
	frame()
	frame()
	if _, ok := find("Clipboard image preview"); !ok {
		t.Fatal("reselecting the image did not rearm hover preview")
	}
	move(image.Pt(700, 440))
	now = now.Add(200 * time.Millisecond)
	frame()
	frame()
	if _, ok := find("Clipboard image preview"); ok {
		t.Fatal("preview did not close after moving away")
	}
	v.previewClipboardImage(v.rows[0].imagePath)
	frame()
	frame()
	if _, ok := find("Clipboard image preview"); !ok {
		t.Fatal("Preview action did not show image pixels")
	}
	router.Queue(key.Event{Name: key.NameEscape, State: key.Press}, key.Event{Name: key.NameEscape, State: key.Release})
	frame()
	frame()
	if v.clipboardPreview.Value() {
		t.Fatal("Escape did not dismiss image dialog")
	}
	v.activate()
	if copied != 1 {
		t.Fatal("image preview broke the selected item's copy action")
	}
}
