package repo

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

	"github.com/dyike/Void/internal/model"
)

// SnippetRepo manages text snippet persistence and lookup.
type SnippetRepo struct {
	mu       sync.RWMutex
	snippets []model.Snippet
	path     string
}

// NewSnippetRepo creates a new snippet repo.
func NewSnippetRepo(dataDir string) *SnippetRepo {
	r := &SnippetRepo{
		path: filepath.Join(dataDir, "snippets.json"),
	}
	r.load()
	return r
}

// GetAll returns all snippets, optionally filtered by a search query.
func (r *SnippetRepo) GetAll(query string) []model.Snippet {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if query == "" {
		result := make([]model.Snippet, len(r.snippets))
		copy(result, r.snippets)
		return result
	}

	q := strings.ToLower(query)
	var result []model.Snippet
	for _, sn := range r.snippets {
		if strings.Contains(strings.ToLower(sn.Name), q) ||
			strings.Contains(strings.ToLower(sn.Content), q) ||
			strings.Contains(strings.ToLower(sn.Prefix), q) {
			result = append(result, sn)
		}
	}
	return result
}

// GetByID returns a snippet by ID.
func (r *SnippetRepo) GetByID(id string) *model.Snippet {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, sn := range r.snippets {
		if sn.ID == id {
			cp := sn
			return &cp
		}
	}
	return nil
}

// Create adds a new snippet.
func (r *SnippetRepo) Create(name, content, prefix string, tags []string) *model.Snippet {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UnixMilli()
	sn := model.Snippet{
		ID:        snippetNewID(),
		Name:      name,
		Content:   content,
		Prefix:    prefix,
		Tags:      tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	r.snippets = append([]model.Snippet{sn}, r.snippets...)
	r.save()
	return &sn
}

// Update modifies an existing snippet.
func (r *SnippetRepo) Update(id, name, content, prefix string, tags []string) *model.Snippet {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, sn := range r.snippets {
		if sn.ID == id {
			r.snippets[i].Name = name
			r.snippets[i].Content = content
			r.snippets[i].Prefix = prefix
			r.snippets[i].Tags = tags
			r.snippets[i].UpdatedAt = time.Now().UnixMilli()
			r.save()
			cp := r.snippets[i]
			return &cp
		}
	}
	return nil
}

// Delete removes a snippet by ID.
func (r *SnippetRepo) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, sn := range r.snippets {
		if sn.ID == id {
			r.snippets = append(r.snippets[:i], r.snippets[i+1:]...)
			r.save()
			return
		}
	}
}

// FindByPrefix returns the first snippet whose prefix matches.
func (r *SnippetRepo) FindByPrefix(prefix string) *model.Snippet {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, sn := range r.snippets {
		if sn.Prefix != "" && sn.Prefix == prefix {
			cp := sn
			return &cp
		}
	}
	return nil
}

// GetAllPrefixes returns all non-empty snippet prefixes.
func (r *SnippetRepo) GetAllPrefixes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var prefixes []string
	for _, sn := range r.snippets {
		if sn.Prefix != "" {
			prefixes = append(prefixes, sn.Prefix)
		}
	}
	return prefixes
}

// CopyToClipboard copies snippet content to the macOS clipboard via pbcopy.
func (r *SnippetRepo) CopyToClipboard(id string) error {
	sn := r.GetByID(id)
	if sn == nil {
		return nil
	}
	c := exec.Command("pbcopy")
	c.Stdin = strings.NewReader(sn.Content)
	return c.Run()
}

func (r *SnippetRepo) load() {
	data, err := os.ReadFile(r.path)
	if err != nil {
		return
	}
	var snippets []model.Snippet
	if err := json.Unmarshal(data, &snippets); err != nil {
		return
	}
	sort.Slice(snippets, func(i, j int) bool {
		return snippets[i].UpdatedAt > snippets[j].UpdatedAt
	})
	r.snippets = snippets
}

func (r *SnippetRepo) save() {
	data, err := json.MarshalIndent(r.snippets, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(r.path, data, 0644)
}

func snippetNewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
