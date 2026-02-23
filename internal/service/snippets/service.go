package snippets

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Snippet is a reusable text template.
type Snippet struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Content   string   `json:"content"`
	Prefix    string   `json:"prefix,omitempty"` // auto-expand trigger, e.g. ";addr"
	Tags      []string `json:"tags,omitempty"`
	CreatedAt int64    `json:"createdAt"`
	UpdatedAt int64    `json:"updatedAt"`
}

// Service manages text snippets.
type Service struct {
	mu       sync.RWMutex
	snippets []Snippet
	path     string
}

// NewService creates a new snippet service.
func NewService(dataDir string) *Service {
	s := &Service{
		path: filepath.Join(dataDir, "snippets.json"),
	}
	s.load()
	return s
}

// GetSnippets returns all snippets, optionally filtered by a search query.
func (s *Service) GetSnippets(query string) []Snippet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query == "" {
		result := make([]Snippet, len(s.snippets))
		copy(result, s.snippets)
		return result
	}

	q := strings.ToLower(query)
	var result []Snippet
	for _, sn := range s.snippets {
		if strings.Contains(strings.ToLower(sn.Name), q) ||
			strings.Contains(strings.ToLower(sn.Content), q) ||
			strings.Contains(strings.ToLower(sn.Prefix), q) {
			result = append(result, sn)
		}
	}
	return result
}

// GetSnippet returns a snippet by ID.
func (s *Service) GetSnippet(id string) (*Snippet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sn := range s.snippets {
		if sn.ID == id {
			cp := sn
			return &cp, nil
		}
	}
	return nil, nil
}

// CreateSnippet adds a new snippet.
func (s *Service) CreateSnippet(name, content, prefix string, tags []string) (*Snippet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()
	sn := Snippet{
		ID:        newID(),
		Name:      name,
		Content:   content,
		Prefix:    prefix,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.snippets = append([]Snippet{sn}, s.snippets...)
	s.save()
	return &sn, nil
}

// UpdateSnippet modifies an existing snippet.
func (s *Service) UpdateSnippet(id, name, content, prefix string, tags []string) (*Snippet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sn := range s.snippets {
		if sn.ID == id {
			s.snippets[i].Name = name
			s.snippets[i].Content = content
			s.snippets[i].Prefix = prefix
			s.snippets[i].Tags = tags
			s.snippets[i].UpdatedAt = time.Now().UnixMilli()
			s.save()
			cp := s.snippets[i]
			return &cp, nil
		}
	}
	return nil, nil
}

// DeleteSnippet removes a snippet by ID.
func (s *Service) DeleteSnippet(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sn := range s.snippets {
		if sn.ID == id {
			s.snippets = append(s.snippets[:i], s.snippets[i+1:]...)
			s.save()
			return nil
		}
	}
	return nil
}

// FindByPrefix returns the first snippet whose prefix matches.
func (s *Service) FindByPrefix(prefix string) *Snippet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sn := range s.snippets {
		if sn.Prefix != "" && sn.Prefix == prefix {
			cp := sn
			return &cp
		}
	}
	return nil
}

// CopyToClipboard copies the snippet content to the macOS clipboard.
func (s *Service) CopyToClipboard(id string) error {
	sn, err := s.GetSnippet(id)
	if err != nil || sn == nil {
		return nil
	}
	// Use pbcopy
	cmd := strings.NewReader(sn.Content)
	c := exec.Command("pbcopy")
	c.Stdin = cmd
	return c.Run()
}

// ─── Persistence ───────────────────────────────────────────────────

func (s *Service) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var snippets []Snippet
	if err := json.Unmarshal(data, &snippets); err != nil {
		return
	}
	// Sort by updatedAt desc
	sort.Slice(snippets, func(i, j int) bool {
		return snippets[i].UpdatedAt > snippets[j].UpdatedAt
	})
	s.snippets = snippets
}

func (s *Service) save() {
	data, err := json.MarshalIndent(s.snippets, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.path, data, 0644)
}

// ─── Helpers ───────────────────────────────────────────────────────

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
