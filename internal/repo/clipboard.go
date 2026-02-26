package repo

import (
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

	preview := content
	if len(preview) > 200 {
		preview = preview[:197] + "..."
	}
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
	out, err := exec.Command("pbpaste").Output()
	if err == nil {
		text := strings.TrimRight(string(out), "\n")
		if text != "" {
			t := "text"
			if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
				t = "url"
			}
			return text, t
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
	r.history = items
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
