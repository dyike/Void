package repo

import "testing"

func TestClipboardUnchangedSkipsImageConversion(t *testing.T) {
	var state clipboardReadState
	version, calls := int64(10), 0
	read := func() (string, string) { calls++; return "/saved/image.png", "image" }
	current := func() int64 { return version }
	if path, kind := state.read(current, read); path != "/saved/image.png" || kind != "image" {
		t.Fatal("initial image missed")
	}
	for range 10000 {
		if path, kind := state.read(current, read); path != "" || kind != "" {
			t.Fatal("unchanged image repeated")
		}
	}
	if calls != 1 {
		t.Fatalf("converted unchanged image %d times", calls)
	}
	version++
	if path, _ := state.read(current, read); path == "" {
		t.Fatal("new clipboard copy was skipped")
	}
	if calls != 2 {
		t.Fatalf("snapshot calls = %d", calls)
	}
}

func TestClipboardRetriesSnapshotChangedDuringRead(t *testing.T) {
	var state clipboardReadState
	version := int64(1)
	current := func() int64 { return version }
	path, kind := state.read(current, func() (string, string) { version++; return "partial", "image" })
	if path != "" || kind != "" {
		t.Fatal("recorded a torn clipboard snapshot")
	}
	path, kind = state.read(current, func() (string, string) { return "complete", "text" })
	if path != "complete" || kind != "text" {
		t.Fatal("did not retry current clipboard version")
	}
	version++
	state.read(current, func() (string, string) { return "", "" })
	state.read(current, func() (string, string) { t.Fatal("empty clipboard was repeatedly read"); return "", "" })
}
