package service

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

// ClipboardService watches the clipboard and maintains history.
type ClipboardService struct {
	mu          sync.RWMutex
	history     []model.ClipboardItem
	lastContent string
	dataDir     string
	stopCh      chan struct{}
}

// NewClipboardService creates a new clipboard service.
func NewClipboardService(dataDir string) *ClipboardService {
	_ = os.MkdirAll(dataDir, 0755)
	s := &ClipboardService{
		dataDir: dataDir,
		stopCh:  make(chan struct{}),
	}
	s.loadFromDisk()
	return s
}

// Start begins background clipboard polling.
func (s *ClipboardService) Start() {
	go s.poll()
}

// Stop terminates background polling.
func (s *ClipboardService) Stop() {
	close(s.stopCh)
}

// GetHistory returns all clipboard history items (newest first).
func (s *ClipboardService) GetHistory() []model.ClipboardItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]model.ClipboardItem, len(s.history))
	copy(result, s.history)
	return result
}

// PasteItem copies the item to clipboard.
func (s *ClipboardService) PasteItem(id string) error {
	item := s.findByID(id)
	if item == nil {
		return fmt.Errorf("item not found: %s", id)
	}
	return s.writeToClipboard(item)
}

// CopyItem copies the item to clipboard.
func (s *ClipboardService) CopyItem(id string) error {
	return s.PasteItem(id)
}

// DeleteItem removes an item from history.
func (s *ClipboardService) DeleteItem(id string) error {
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
func (s *ClipboardService) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = nil
	s.saveToDisk()
	return nil
}

func (s *ClipboardService) poll() {
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

func (s *ClipboardService) checkClipboard() {
	content, ctype := s.readClipboard()
	if content == "" || content == s.lastContent {
		return
	}
	s.lastContent = content
	s.addItem(content, ctype)
}

func (s *ClipboardService) readClipboard() (string, string) {
	out, err := exec.Command("pbpaste").Output()
	if err == nil {
		text := strings.TrimRight(string(out), "\n")
		if text != "" {
			ctype := "text"
			if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
				ctype = "url"
			}
			return text, ctype
		}
	}
	return "", ""
}

func (s *ClipboardService) addItem(content, ctype string) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	s.history = append([]model.ClipboardItem{item}, s.history...)
	if len(s.history) > maxClipboardHistory {
		s.history = s.history[:maxClipboardHistory]
	}
	s.saveToDisk()
}

func (s *ClipboardService) findByID(id string) *model.ClipboardItem {
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

func (s *ClipboardService) writeToClipboard(item *model.ClipboardItem) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(item.Content)
	return cmd.Run()
}

func (s *ClipboardService) historyPath() string {
	return filepath.Join(s.dataDir, "clipboard-history.json")
}

func (s *ClipboardService) loadFromDisk() {
	data, err := os.ReadFile(s.historyPath())
	if err != nil {
		return
	}
	var items []model.ClipboardItem
	if err := json.Unmarshal(data, &items); err != nil {
		return
	}
	s.history = items
	if len(s.history) > 0 {
		s.lastContent = s.history[0].Content
	}
}

func (s *ClipboardService) saveToDisk() {
	data, err := json.MarshalIndent(s.history, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.historyPath(), data, 0644)
}

func clipboardNewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
