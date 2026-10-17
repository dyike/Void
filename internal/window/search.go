package window

import (
	"slices"
	"strings"

	"github.com/dyike/Void/internal/model"
	"github.com/sahilm/fuzzy"
)

// Rank titles before descriptions. Fuzzy-matching a long filesystem path makes
// unrelated system helpers appear for almost every application query.
func searchCommands(commands []model.CommandInfo, query string) []model.CommandInfo {
	query = strings.TrimSpace(query)
	if query == "" {
		return commands
	}
	titles := make([]string, len(commands))
	for i, c := range commands {
		titles[i] = c.Title
	}
	matches := fuzzy.Find(query, titles)
	q := strings.ToLower(query)
	priority := func(title string) int {
		title = strings.ToLower(title)
		if title == q {
			return 0
		}
		if strings.HasPrefix(title, q) {
			return 1
		}
		if strings.Contains(title, q) {
			return 2
		}
		return 3
	}
	slices.SortStableFunc(matches, func(a, b fuzzy.Match) int { return priority(a.Str) - priority(b.Str) })
	results := make([]model.CommandInfo, 0, len(matches))
	seen := make(map[int]bool, len(matches))
	for _, match := range matches {
		results = append(results, commands[match.Index])
		seen[match.Index] = true
	}
	for i, c := range commands {
		if !seen[i] && strings.Contains(strings.ToLower(c.Subtitle), q) {
			results = append(results, c)
		}
	}
	return results
}
