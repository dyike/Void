package service

import (
	"fmt"
	"time"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/repo"
)

// ClipboardService watches the clipboard and maintains history.
type ClipboardService struct {
	repo   *repo.ClipboardRepo
	stopCh chan struct{}
}

// NewClipboardService creates a new clipboard service.
func NewClipboardService(r *repo.ClipboardRepo) *ClipboardService {
	return &ClipboardService{
		repo:   r,
		stopCh: make(chan struct{}),
	}
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
	return s.repo.GetHistory()
}

// PasteItem copies the item back to the clipboard.
func (s *ClipboardService) PasteItem(id string) error {
	item := s.repo.FindByID(id)
	if item == nil {
		return fmt.Errorf("item not found: %s", id)
	}
	return s.repo.WriteItemToClipboard(*item)
}

// CopyItem copies the item to clipboard (alias for PasteItem).
func (s *ClipboardService) CopyItem(id string) error {
	return s.PasteItem(id)
}

// DeleteItem removes an item from history.
func (s *ClipboardService) DeleteItem(id string) error {
	return s.repo.Delete(id)
}

// ClearHistory removes all clipboard history.
func (s *ClipboardService) ClearHistory() error {
	s.repo.Clear()
	return nil
}

func (s *ClipboardService) poll() {
	lastFingerprint := s.repo.LastFingerprint()
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			content, ctype := s.repo.ReadClipboardIfChanged()
			if content == "" {
				continue
			}
			fingerprint := ctype + ":" + content
			if ctype == "url" {
				fingerprint = "text:" + content
			}
			if fingerprint == lastFingerprint {
				continue
			}
			lastFingerprint = fingerprint
			s.repo.AddItem(content, ctype)
		}
	}
}
