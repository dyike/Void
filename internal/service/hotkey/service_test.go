//go:build darwin

package hotkey

import "testing"

func TestMatchSnippetTriggerExact(t *testing.T) {
	match, ok := matchSnippetTrigger("hello ;addr", []string{";addr"})
	if !ok {
		t.Fatal("expected exact snippet trigger match")
	}
	if match.prefix != ";addr" || match.deleteText != ";addr" || match.keepText != "" {
		t.Fatalf("unexpected match: %#v", match)
	}
}

func TestMatchSnippetTriggerWithDelimiter(t *testing.T) {
	match, ok := matchSnippetTrigger("hello ;addr ", []string{";addr"})
	if !ok {
		t.Fatal("expected delimited snippet trigger match")
	}
	if match.prefix != ";addr" || match.deleteText != ";addr " || match.keepText != " " {
		t.Fatalf("unexpected match: %#v", match)
	}
}

func TestMatchSnippetTriggerPrefersProvidedOrder(t *testing.T) {
	match, ok := matchSnippetTrigger("hello ;addr ", []string{";addr", ";ad"})
	if !ok {
		t.Fatal("expected snippet trigger match")
	}
	if match.prefix != ";addr" {
		t.Fatalf("expected longer prefix to be matched by caller-sorted order, got %#v", match)
	}
}
