package commands

import (
	"path/filepath"
	"strings"
)

// ─── Helpers ───────────────────────────────────────────────────────────────

func fileIcon(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return "FileText"
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg":
		return "Image"
	case ".mp3", ".wav", ".flac", ".aac":
		return "Music"
	case ".mp4", ".mov", ".avi", ".mkv":
		return "Video"
	case ".zip", ".tar", ".gz", ".rar":
		return "Archive"
	case ".go", ".py", ".js", ".ts", ".rs", ".swift":
		return "FileCode"
	case ".md", ".txt":
		return "FileText"
	case ".xlsx", ".csv":
		return "FileSpreadsheet"
	case ".app":
		return "Monitor"
	default:
		return "File"
	}
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
