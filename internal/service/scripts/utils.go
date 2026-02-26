package scripts

import (
	"path/filepath"
	"strings"
)

// ─── Helpers ─────────────────────────────────────────────────────────────

func isScriptFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".sh" || ext == ".py" || ext == ".rb" || ext == ".js" ||
		ext == ".ts" || ext == ".zsh" || ext == ".fish" || ext == ".swift"
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return result
}
