//go:build darwin

package service

import (
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/dyike/Void/internal/repo"
)

var running atomic.Bool

// HotkeyService manages global hotkey registration and snippet auto-expansion.
type HotkeyService struct {
	repo            *repo.HotkeyRepo
	recording       atomic.Bool
	snippetMu       sync.RWMutex
	snippetLookup   SnippetLookupFn
	snippetPrefixes func() []string
}

// NewHotkeyService creates a new hotkey service.
func NewHotkeyService(r *repo.HotkeyRepo) *HotkeyService {
	return &HotkeyService{repo: r}
}

// Start configures the hotkey and installs the event tap.
func (s *HotkeyService) Start(h func(), keySpec string) bool {
	if !s.repo.ConfigureKey(keySpec) {
		return false
	}
	return s.startTap(h)
}

// Restart stops and restarts the event tap with a new hotkey.
func (s *HotkeyService) Restart(h func(), keySpec string) bool {
	s.Stop()
	time.Sleep(50 * time.Millisecond)
	return s.Start(h, keySpec)
}

// Stop terminates the event tap and all goroutines.
func (s *HotkeyService) Stop() {
	running.Store(false)
	s.repo.StopEventTap()
}

func (s *HotkeyService) startTap(h func()) bool {
	if !s.repo.HasPermission() {
		log.Println("hotkey: requesting Accessibility permission…")
		s.repo.RequestPermission()
		if !s.repo.HasPermission() {
			log.Println("hotkey: Accessibility not granted — check System Settings → Privacy & Security → Accessibility")
			return false
		}
	}
	log.Println("hotkey: Accessibility granted")
	if !s.repo.HasListenEventPermission() {
		log.Println("hotkey: requesting Input Monitoring permission…")
		if !s.repo.RequestListenEventPermission() || !s.repo.HasListenEventPermission() {
			log.Println("hotkey: Input Monitoring not granted — check System Settings → Privacy & Security → Input Monitoring")
			return false
		}
	}
	log.Println("hotkey: Input Monitoring granted")
	if !s.repo.StartEventTap() {
		log.Println("hotkey: CGEventTapCreate failed — Accessibility may need to be re-granted")
		return false
	}
	running.Store(true)
	log.Println("hotkey: CGEventTap active")

	go func() {
		for running.Load() {
			if s.repo.PollHotkey() && !s.recording.Load() {
				if s.TryExpandFocusedSnippet() {
					time.Sleep(20 * time.Millisecond)
					continue
				}
				h()
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()
	return true
}

// SnippetLookupFn looks up a snippet prefix and returns the content if found.
type SnippetLookupFn func(prefix string) *string

// StartSnippetExpand starts a goroutine that monitors typed characters
// and expands snippet prefixes.
func (s *HotkeyService) StartSnippetExpand(lookup SnippetLookupFn, getPrefixes func() []string) {
	s.snippetMu.Lock()
	s.snippetLookup = lookup
	s.snippetPrefixes = getPrefixes
	s.snippetMu.Unlock()

	lastRulesSignature := s.syncSnippetRules(lookup, getPrefixes, "")
	go func() {
		for running.Load() {
			time.Sleep(2 * time.Second)
			lastRulesSignature = s.syncSnippetRules(lookup, getPrefixes, lastRulesSignature)
		}
		log.Println("snippet: goroutine exited (running=false)")
	}()
}

func (s *HotkeyService) syncSnippetRules(lookup SnippetLookupFn, getPrefixes func() []string, lastSignature string) string {
	prefixes := getPrefixes()
	sort.Slice(prefixes, func(i, j int) bool {
		return utf8.RuneCountInString(prefixes[i]) > utf8.RuneCountInString(prefixes[j])
	})

	active := make([]string, 0, len(prefixes))
	contents := make(map[string]string, len(prefixes))
	var signature strings.Builder
	for _, rawPrefix := range prefixes {
		content := lookup(rawPrefix)
		prefix := repo.NormalizeSnippetTriggerText(rawPrefix)
		if content == nil && prefix != rawPrefix {
			content = lookup(prefix)
		}
		if prefix == "" || content == nil {
			continue
		}
		active = append(active, prefix)
		contents[prefix] = *content
		signature.WriteString(prefix)
		signature.WriteByte(0)
		signature.WriteString(*content)
		signature.WriteByte(0)
	}

	nextSignature := signature.String()
	if nextSignature == lastSignature {
		return lastSignature
	}

	s.repo.ClearSnippetRules()
	for _, prefix := range active {
		s.repo.AddSnippetRule(prefix, contents[prefix])
	}
	log.Printf("snippet: auto-expand active with %d prefix(es): %s", len(active), strings.Join(active, ", "))
	return nextSignature
}

// TryExpandFocusedSnippet expands a trigger already present in the focused input.
// This is a fallback for remote input paths that do not emit per-key CGEvents.
func (s *HotkeyService) TryExpandFocusedSnippet() bool {
	s.snippetMu.RLock()
	lookup := s.snippetLookup
	getPrefixes := s.snippetPrefixes
	s.snippetMu.RUnlock()
	if lookup == nil || getPrefixes == nil {
		return false
	}

	beforeCursor := s.repo.FocusedTextBeforeCursor(80)
	if beforeCursor == "" {
		return false
	}

	prefixes := getPrefixes()
	sort.Slice(prefixes, func(i, j int) bool {
		return utf8.RuneCountInString(prefixes[i]) > utf8.RuneCountInString(prefixes[j])
	})

	match, ok := repo.MatchSnippetTrigger(beforeCursor, prefixes)
	if !ok {
		return false
	}

	content := lookup(match.Prefix)
	if content == nil {
		for _, rawPrefix := range prefixes {
			if repo.NormalizeSnippetTriggerText(rawPrefix) == match.Prefix {
				content = lookup(rawPrefix)
				break
			}
		}
	}
	if content == nil {
		return false
	}

	replacement := *content + match.KeepText
	deleteChars := utf8.RuneCountInString(match.DeleteText)
	log.Printf("snippet: manually expanding focused prefix %q replacementLen=%d", match.Prefix, len(replacement))
	s.repo.ExpandSnippet(deleteChars, replacement)
	s.repo.ClearSnippetBuffer()
	return true
}

// SetRecording lets the shortcut recorder receive the current global chord
// without launching Void or expanding snippets.
func (s *HotkeyService) SetRecording(recording bool) {
	s.recording.Store(recording)
	s.repo.SetHotkeyRecording(recording)
	if recording {
		s.repo.PollHotkey()
	}
}
