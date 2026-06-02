package repo

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

	"github.com/dyike/Void/internal/model"
)

const maxClipboardHistory = 100

// ClipboardRepo manages clipboard history persistence and read/write operations.
type ClipboardRepo struct {
	mu      sync.RWMutex
	history []model.ClipboardItem
	dataDir string
}

// NewClipboardRepo creates a new clipboard repo.
func NewClipboardRepo(dataDir string) *ClipboardRepo {
	_ = os.MkdirAll(dataDir, 0755)
	r := &ClipboardRepo{dataDir: dataDir}
	r.loadFromDisk()
	return r
}

// LastContent returns the content of the most recently added item (for dedup).
func (r *ClipboardRepo) LastContent() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.history) > 0 {
		return r.history[0].Content
	}
	return ""
}

// GetHistory returns all clipboard history items (newest first).
func (r *ClipboardRepo) GetHistory() []model.ClipboardItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]model.ClipboardItem, len(r.history))
	copy(result, r.history)
	return result
}

// AddItem inserts a new clipboard item into history.
func (r *ClipboardRepo) AddItem(content, ctype string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	preview := truncateRunes(content, 200)
	item := model.ClipboardItem{
		ID:        clipboardNewID(),
		Type:      ctype,
		Content:   content,
		Preview:   preview,
		Timestamp: time.Now().UnixMilli(),
	}
	r.history = append([]model.ClipboardItem{item}, r.history...)
	if len(r.history) > maxClipboardHistory {
		r.history = r.history[:maxClipboardHistory]
	}
	r.saveToDisk()
}

// FindByID returns a clipboard item by ID.
func (r *ClipboardRepo) FindByID(id string) *model.ClipboardItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.history {
		if item.ID == id {
			cp := item
			return &cp
		}
	}
	return nil
}

// Delete removes a clipboard item from history.
func (r *ClipboardRepo) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, item := range r.history {
		if item.ID == id {
			r.history = append(r.history[:i], r.history[i+1:]...)
			r.saveToDisk()
			return nil
		}
	}
	return fmt.Errorf("item not found: %s", id)
}

// Clear removes all clipboard history.
func (r *ClipboardRepo) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history = nil
	r.saveToDisk()
}

// ReadClipboard reads the current system clipboard content via pbpaste.
func (r *ClipboardRepo) ReadClipboard() (content, ctype string) {
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

// WriteToClipboard writes content to the system clipboard via pbcopy.
func (r *ClipboardRepo) WriteToClipboard(content string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}

func (r *ClipboardRepo) historyPath() string {
	return filepath.Join(r.dataDir, "clipboard-history.json")
}

func (r *ClipboardRepo) loadFromDisk() {
	data, err := os.ReadFile(r.historyPath())
	if err != nil {
		return
	}
	var items []model.ClipboardItem
	if err := json.Unmarshal(data, &items); err != nil {
		return
	}
	r.history = sanitizeHistory(items)
	if len(r.history) != len(items) {
		r.saveToDisk()
	}
}

func (r *ClipboardRepo) saveToDisk() {
	data, err := json.MarshalIndent(r.history, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(r.historyPath(), data, 0644)
}

func clipboardNewID() string {
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

func sanitizeHistory(items []model.ClipboardItem) []model.ClipboardItem {
	result := make([]model.ClipboardItem, 0, len(items))
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
