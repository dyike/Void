package service

import (
	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/repo"
)

// SnippetService manages text snippets.
type SnippetService struct {
	repo *repo.SnippetRepo
}

// NewSnippetService creates a new snippet service.
func NewSnippetService(r *repo.SnippetRepo) *SnippetService {
	return &SnippetService{repo: r}
}

// GetSnippets returns all snippets, optionally filtered by a search query.
func (s *SnippetService) GetSnippets(query string) []model.Snippet {
	return s.repo.GetAll(query)
}

// GetSnippet returns a snippet by ID.
func (s *SnippetService) GetSnippet(id string) (*model.Snippet, error) {
	return s.repo.GetByID(id), nil
}

// CreateSnippet adds a new snippet.
func (s *SnippetService) CreateSnippet(name, content, prefix string, tags []string) (*model.Snippet, error) {
	return s.repo.Create(name, content, prefix, tags), nil
}

// UpdateSnippet modifies an existing snippet.
func (s *SnippetService) UpdateSnippet(id, name, content, prefix string, tags []string) (*model.Snippet, error) {
	return s.repo.Update(id, name, content, prefix, tags), nil
}

// DeleteSnippet removes a snippet by ID.
func (s *SnippetService) DeleteSnippet(id string) error {
	s.repo.Delete(id)
	return nil
}

// FindByPrefix returns the first snippet whose prefix matches.
func (s *SnippetService) FindByPrefix(prefix string) *model.Snippet {
	return s.repo.FindByPrefix(prefix)
}

// GetAllPrefixes returns all non-empty snippet prefixes.
func (s *SnippetService) GetAllPrefixes() []string {
	return s.repo.GetAllPrefixes()
}

// ExpandSnippet returns the content for a matching prefix, or nil.
func (s *SnippetService) ExpandSnippet(prefix string) *string {
	sn := s.repo.FindByPrefix(prefix)
	if sn == nil {
		return nil
	}
	return &sn.Content
}

// CopyToClipboard copies the snippet content to the macOS clipboard.
func (s *SnippetService) CopyToClipboard(id string) error {
	return s.repo.CopyToClipboard(id)
}
