//go:build darwin

package service

import (
	"log"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dyike/Void/internal/repo"
)

var running atomic.Bool

// HotkeyService manages global hotkey registration and snippet auto-expansion.
type HotkeyService struct {
	repo *repo.HotkeyRepo
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
	if !s.repo.StartEventTap() {
		log.Println("hotkey: CGEventTapCreate failed — Accessibility may need to be re-granted")
		return false
	}
	running.Store(true)
	log.Println("hotkey: CGEventTap active")

	go func() {
		for running.Load() {
			if s.repo.PollHotkey() {
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
	go func() {
		var lastBuf string
		for running.Load() {
			buf := s.repo.GetSnippetBuffer()
			if buf != lastBuf && len(buf) > 0 {
				lastBuf = buf
				prefixes := getPrefixes()
				for _, prefix := range prefixes {
					if strings.HasSuffix(buf, prefix) {
						content := lookup(prefix)
						if content != nil {
							log.Printf("snippet: MATCH! expanding prefix %q → %d chars", prefix, len(*content))
							s.expandSnippet(prefix, *content)
							s.repo.ClearSnippetBuffer()
							lastBuf = ""
							break
						}
					}
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		log.Println("snippet: goroutine exited (running=false)")
	}()
	log.Println("snippet: auto-expand active")
}

func (s *HotkeyService) expandSnippet(prefix string, content string) {
	// 1. Save current clipboard
	oldClip := s.repo.GetClipboardText()

	// 2. Send backspaces to delete the prefix
	s.repo.SendBackspaces(len(prefix))
	time.Sleep(50 * time.Millisecond)

	// 3. Set clipboard to snippet content
	s.repo.SetClipboardText(content)
	time.Sleep(30 * time.Millisecond)

	// 4. Paste (Cmd+V)
	s.repo.PasteClipboard()
	time.Sleep(100 * time.Millisecond)

	// 5. Restore original clipboard
	if oldClip != "" {
		s.repo.SetClipboardText(oldClip)
	}
}
