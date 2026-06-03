package repo

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/dyike/Void/internal/model"
)

func TestNormalizeClipboardTextRejectsBinary(t *testing.T) {
	if text, ok := normalizeClipboardText([]byte{0xff, 0xfe, 0x00, 0x01}); ok {
		t.Fatalf("expected binary clipboard payload to be rejected, got %q", text)
	}
}

func TestNormalizeClipboardTextRejectsUnreadableText(t *testing.T) {
	cases := []string{
		"hello\u0000world",
		"hello\uFFFDworld",
	}

	for _, input := range cases {
		if text, ok := normalizeClipboardText([]byte(input)); ok {
			t.Fatalf("expected unreadable text to be rejected, got %q", text)
		}
	}
}

func TestNormalizeClipboardTextAcceptsReadableUnicode(t *testing.T) {
	text, ok := normalizeClipboardText([]byte("中文内容\nemoji 🚀\n"))
	if !ok {
		t.Fatal("expected readable unicode text to be accepted")
	}
	if text != "中文内容\nemoji 🚀" {
		t.Fatalf("unexpected normalized text: %q", text)
	}
}

func TestTruncateRunesPreservesUTF8(t *testing.T) {
	input := strings.Repeat("剪", 210)
	got := truncateRunes(input, 200)

	if !utf8.ValidString(got) {
		t.Fatalf("truncated preview is not valid UTF-8: %q", got)
	}
	if utf8.RuneCountInString(got) != 200 {
		t.Fatalf("expected 200 runes, got %d", utf8.RuneCountInString(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis suffix, got %q", got)
	}
}

func TestSanitizeHistoryDropsBadTextItems(t *testing.T) {
	items := []model.ClipboardItem{
		{ID: "good", Type: "text", Content: "正常文本", Preview: "正常文本"},
		{ID: "bad", Type: "text", Content: "bad\uFFFDtext", Preview: "bad\uFFFDtext"},
	}

	got := sanitizeHistory(items)
	if len(got) != 1 || got[0].ID != "good" {
		t.Fatalf("unexpected sanitized history: %#v", got)
	}
}

func TestAddItemMovesDuplicateToTop(t *testing.T) {
	clipboardRepo := NewClipboardRepo(t.TempDir())
	clipboardRepo.AddItem("first", "text")
	clipboardRepo.AddItem("second", "text")
	clipboardRepo.AddItem("first", "text")

	got := clipboardRepo.GetHistory()
	if len(got) != 2 {
		t.Fatalf("expected unique history, got %d items: %#v", len(got), got)
	}
	if got[0].Content != "first" || got[1].Content != "second" {
		t.Fatalf("unexpected order after duplicate insert: %#v", got)
	}
}

func TestSanitizeHistoryDeduplicatesItems(t *testing.T) {
	items := []model.ClipboardItem{
		{ID: "new", Type: "text", Content: "same", Preview: "same"},
		{ID: "old", Type: "text", Content: "same", Preview: "same"},
		{ID: "other", Type: "url", Content: "https://example.com", Preview: "https://example.com"},
	}

	got := sanitizeHistory(items)
	if len(got) != 2 {
		t.Fatalf("expected sanitized unique history, got %#v", got)
	}
	if got[0].ID != "new" || got[1].ID != "other" {
		t.Fatalf("unexpected sanitized history order: %#v", got)
	}
}
