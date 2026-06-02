package clipboard

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// ClipboardItem represents a clipboard history entry.
type ClipboardItem struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // "text" | "url" | "image" | "file"
	Content   string                 `json:"content"`
	Preview   string                 `json:"preview,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

const maxHistory = 100

// Service watches the clipboard and maintains history.
type Service struct {
	mu          sync.RWMutex
	history     []ClipboardItem
	lastContent string
	dataDir     string
	stopCh      chan struct{}
}

// NewService creates a new clipboard service.
func NewService(dataDir string) *Service {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		// best effort
	}
	s := &Service{
		dataDir: dataDir,
		stopCh:  make(chan struct{}),
	}
	s.loadFromDisk()
	return s
}

// Start begins background clipboard polling. Call once.
func (s *Service) Start() {
	go s.poll()
}

// Stop terminates background polling.
func (s *Service) Stop() {
	close(s.stopCh)
}

// GetHistory returns all clipboard history items (newest first).
func (s *Service) GetHistory() []ClipboardItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ClipboardItem, len(s.history))
	copy(result, s.history)
	return result
}

// PasteItem copies the item to clipboard and returns its content for pasting.
func (s *Service) PasteItem(id string) error {
	item := s.findByID(id)
	if item == nil {
		return fmt.Errorf("item not found: %s", id)
	}
	return s.writeToClipboard(item)
}

// CopyItem copies the item to clipboard (without pasting).
func (s *Service) CopyItem(id string) error {
	return s.PasteItem(id)
}

// DeleteItem removes an item from history.
func (s *Service) DeleteItem(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, item := range s.history {
		if item.ID == id {
			s.history = append(s.history[:i], s.history[i+1:]...)
			s.saveToDisk()
			return nil
		}
	}
	return fmt.Errorf("item not found: %s", id)
}

// ClearHistory removes all clipboard history.
func (s *Service) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = nil
	s.saveToDisk()
	return nil
}

// ─── Polling ───────────────────────────────────────────────────────────────

func (s *Service) poll() {
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkClipboard()
		}
	}
}

func (s *Service) checkClipboard() {
	content, ctype := s.readClipboard()
	if content == "" || content == s.lastContent {
		return
	}
	s.lastContent = content
	s.addItem(content, ctype)
}

func (s *Service) readClipboard() (string, string) {
	// Prefer plain text. Without this, pbpaste can emit non-text payloads for
	// rich clipboard items, which then show up as unreadable history entries.
	out, err := exec.Command("pbpaste", "-Prefer", "txt").Output()
	if err != nil {
		out, err = exec.Command("pbpaste").Output()
	}
	if err == nil {
		if text, ok := normalizeClipboardText(out); ok {
			ctype := "text"
			if isURL(text) {
				ctype = "url"
			}
			return text, ctype
		}
	}
	return "", ""
}

func (s *Service) addItem(content, ctype string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := newID()
	preview := truncateRunes(content, 200)

	item := ClipboardItem{
		ID:        id,
		Type:      ctype,
		Content:   content,
		Preview:   preview,
		Timestamp: time.Now().UnixMilli(),
	}

	// Prepend (newest first), cap at maxHistory
	s.history = append([]ClipboardItem{item}, s.history...)
	if len(s.history) > maxHistory {
		s.history = s.history[:maxHistory]
	}
	s.saveToDisk()
}

func (s *Service) findByID(id string) *ClipboardItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.history {
		if item.ID == id {
			cp := item
			return &cp
		}
	}
	return nil
}

func (s *Service) writeToClipboard(item *ClipboardItem) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(item.Content)
	return cmd.Run()
}

// ─── Persistence ───────────────────────────────────────────────────────────

func (s *Service) historyPath() string {
	return filepath.Join(s.dataDir, "clipboard-history.json")
}

func (s *Service) loadFromDisk() {
	data, err := os.ReadFile(s.historyPath())
	if err != nil {
		return
	}
	var items []ClipboardItem
	if err := json.Unmarshal(data, &items); err != nil {
		return
	}
	s.history = sanitizeHistory(items)
	if len(s.history) > 0 {
		s.lastContent = s.history[0].Content
	}
	if len(s.history) != len(items) {
		s.saveToDisk()
	}
}

func (s *Service) saveToDisk() {
	data, err := json.MarshalIndent(s.history, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.historyPath(), data, 0644)
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func normalizeClipboardText(out []byte) (string, bool) {
	out = bytes.TrimRight(out, "\r\n")
	if len(out) == 0 || !utf8.Valid(out) {
		return "", false
	}

	text := string(out)
	if strings.TrimSpace(text) == "" || !isReadableText(text) {
		return "", false
	}
	return text, true
}

func isReadableText(text string) bool {
	for _, r := range text {
		if r == utf8.RuneError || r == '\uFFFD' {
			return false
		}
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func sanitizeHistory(items []ClipboardItem) []ClipboardItem {
	result := make([]ClipboardItem, 0, len(items))
	for _, item := range items {
		if item.Content == "" {
			continue
		}
		switch item.Type {
		case "text", "url", "":
			if !utf8.ValidString(item.Content) || !isReadableText(item.Content) {
				continue
			}
			preview := item.Preview
			if preview == "" || !utf8.ValidString(preview) || !isReadableText(preview) {
				preview = item.Content
			}
			item.Preview = truncateRunes(preview, 200)
		}
		result = append(result, item)
	}
	return result
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}

	var b strings.Builder
	limit := max - 3
	if limit < 0 {
		limit = 0
	}
	count := 0
	for _, r := range s {
		if count >= limit {
			break
		}
		b.WriteRune(r)
		count++
	}
	b.WriteString("...")
	return b.String()
}
