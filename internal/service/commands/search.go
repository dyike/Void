package commands

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/Void/internal/service/filesearch"
)

// SearchFiles searches for files by name using fuzzy matching (pure Go, no Spotlight).
func (s *Service) SearchFiles(query string) ([]model.CommandInfo, error) {
	if query == "" {
		return nil, nil
	}
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		log.Println("filesearch: HOME env is empty")
		return nil, nil
	}
	log.Printf("filesearch: searching query=%q in %s", query, homeDir)
	start := time.Now()
	results := filesearch.SearchFiles(homeDir, query)
	elapsed := time.Since(start)
	log.Printf("filesearch: found %d results in %v", len(results), elapsed)
	if len(results) > 0 {
		log.Printf("filesearch: top result: %s (score=%d)", results[0].Path, results[0].Score)
	}
	var cmds []model.CommandInfo
	for _, r := range results {
		cmds = append(cmds, model.CommandInfo{
			ID:       "file-" + slugify(r.Path),
			Title:    r.Name,
			Subtitle: r.Dir,
			Icon:     fileIcon(r.Path),
			Category: "file",
			Path:     r.Path,
			Score:    float64(r.Score),
		})
	}
	return cmds, nil
}

// SearchFileContents searches inside file contents using grep (pure Go).
func (s *Service) SearchFileContents(query string) ([]model.CommandInfo, error) {
	if query == "" {
		return nil, nil
	}
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		return nil, nil
	}
	results := filesearch.SearchFileContents(homeDir, query)
	var cmds []model.CommandInfo
	for _, r := range results {
		subtitle := r.Dir
		if r.LineNum > 0 {
			subtitle = fmt.Sprintf("L%d: %s", r.LineNum, r.LineText)
		}
		cmds = append(cmds, model.CommandInfo{
			ID:       "file-" + slugify(r.Path),
			Title:    r.Name,
			Subtitle: subtitle,
			Icon:     fileIcon(r.Path),
			Category: "file",
			Path:     r.Path,
			Score:    float64(r.Score),
		})
	}
	return cmds, nil
}
