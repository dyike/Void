package filesearch

import (
	"bufio"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sahilm/fuzzy"
)

// Result represents a file search result.
type Result struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	Dir      string `json:"dir"`
	Score    int    `json:"score"`
	LineNum  int    `json:"lineNum"`
	LineText string `json:"lineText"`
}

// Directories to skip during walk.
var skipDirs = map[string]bool{
	".git": true, ".svn": true, ".hg": true,
	"node_modules": true, "__pycache__": true,
	".cache": true, ".Trash": true, "Library": true,
	".npm": true, ".cargo": true, ".rustup": true,
	".local": true, ".docker": true, ".gradle": true,
	".m2": true, ".gem": true, ".cpan": true,
	"Pictures": true, "Music": true, "Movies": true,
}

const maxResults = 30
const searchTimeout = 5 * time.Second

// SearchFiles searches for files by name using sahilm/fuzzy.
func SearchFiles(rootDir string, query string) []Result {
	if query == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	defer cancel()

	// Collect file paths
	start := time.Now()
	paths := collectFiles(ctx, rootDir)
	collectTime := time.Since(start)
	log.Printf("filesearch: collected %d files in %v", len(paths), collectTime)

	if len(paths) == 0 {
		return nil
	}

	// Build display names (relative paths for better matching)
	displayNames := make([]string, len(paths))
	for i, p := range paths {
		rel, err := filepath.Rel(rootDir, p)
		if err != nil {
			rel = filepath.Base(p)
		}
		displayNames[i] = rel
	}

	// Fuzzy match
	matchStart := time.Now()
	matches := fuzzy.Find(query, displayNames)
	matchTime := time.Since(matchStart)
	log.Printf("filesearch: fuzzy matched %d results in %v", len(matches), matchTime)

	var results []Result
	for _, m := range matches {
		if len(results) >= maxResults {
			break
		}
		p := paths[m.Index]
		results = append(results, Result{
			Path:  p,
			Name:  filepath.Base(p),
			Dir:   filepath.Dir(p),
			Score: m.Score,
		})
	}
	return results
}

// collectFiles walks the directory tree concurrently.
func collectFiles(ctx context.Context, rootDir string) []string {
	var (
		mu    sync.Mutex
		paths []string
		count atomic.Int32
		wg    sync.WaitGroup
		sem   = make(chan struct{}, 16)
	)

	walkDir(ctx, rootDir, &mu, &paths, &count, &wg, sem, 0)
	wg.Wait()

	return paths
}

func walkDir(ctx context.Context, dir string, mu *sync.Mutex, paths *[]string, count *atomic.Int32, wg *sync.WaitGroup, sem chan struct{}, depth int) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if count.Load() > 100000 || depth > 10 {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		default:
		}

		name := entry.Name()

		if entry.IsDir() {
			if (len(name) > 1 && name[0] == '.') || skipDirs[name] {
				continue
			}
			subPath := filepath.Join(dir, name)
			// Non-blocking: if sem full, walk synchronously (prevents deadlock)
			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func(p string, d int) {
					defer wg.Done()
					defer func() { <-sem }()
					walkDir(ctx, p, mu, paths, count, wg, sem, d)
				}(subPath, depth+1)
			default:
				walkDir(ctx, subPath, mu, paths, count, wg, sem, depth+1)
			}
			continue
		}

		// Skip hidden files
		if len(name) > 1 && name[0] == '.' {
			continue
		}

		fullPath := filepath.Join(dir, name)
		mu.Lock()
		*paths = append(*paths, fullPath)
		mu.Unlock()
		count.Add(1)
	}
}

// SearchFileContents searches inside files for a text pattern.
func SearchFileContents(rootDir string, query string) []Result {
	if query == "" || len(query) < 2 {
		return nil
	}
	queryLower := strings.ToLower(query)

	ctx, cancel := context.WithTimeout(context.Background(), searchTimeout)
	defer cancel()

	var (
		mu      sync.Mutex
		results []Result
		found   atomic.Int32
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 16)
	)

	grepDir(ctx, rootDir, queryLower, &mu, &results, &found, &wg, sem, 0)
	wg.Wait()

	sortResults(results)
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results
}

func grepDir(ctx context.Context, dir string, query string, mu *sync.Mutex, results *[]Result, found *atomic.Int32, wg *sync.WaitGroup, sem chan struct{}, depth int) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if found.Load() >= maxResults*2 || depth > 10 {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if found.Load() >= maxResults*2 {
			return
		}

		name := entry.Name()
		if entry.IsDir() {
			if (len(name) > 1 && name[0] == '.') || skipDirs[name] {
				continue
			}
			subPath := filepath.Join(dir, name)
			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func(p string, d int) {
					defer wg.Done()
					defer func() { <-sem }()
					grepDir(ctx, p, query, mu, results, found, wg, sem, d)
				}(subPath, depth+1)
			default:
				grepDir(ctx, subPath, query, mu, results, found, wg, sem, depth+1)
			}
			continue
		}

		if (len(name) > 1 && name[0] == '.') || !isTextFile(name) {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Size() > 1<<20 {
			continue
		}

		fullPath := filepath.Join(dir, name)
		if lineNum, lineText := grepFile(fullPath, query); lineNum > 0 {
			mu.Lock()
			*results = append(*results, Result{
				Path:     fullPath,
				Name:     name,
				Dir:      dir,
				Score:    100,
				LineNum:  lineNum,
				LineText: lineText,
			})
			mu.Unlock()
			found.Add(1)
		}
	}
}

func grepFile(path string, query string) (int, string) {
	f, err := os.Open(path)
	if err != nil {
		return 0, ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), query) {
			text := strings.TrimSpace(line)
			if len(text) > 120 {
				text = text[:120] + "…"
			}
			return lineNum, text
		}
		if lineNum > 5000 {
			break
		}
	}
	return 0, ""
}

// ─── Helpers ─────────────────────────────────────────────────────────────

var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".go": true, ".py": true, ".js": true,
	".ts": true, ".tsx": true, ".jsx": true, ".json": true, ".yaml": true,
	".yml": true, ".toml": true, ".xml": true, ".html": true, ".css": true,
	".scss": true, ".less": true, ".sh": true, ".bash": true, ".zsh": true,
	".fish": true, ".rb": true, ".rs": true, ".c": true, ".h": true,
	".cpp": true, ".hpp": true, ".java": true, ".kt": true, ".swift": true,
	".sql": true, ".csv": true, ".log": true, ".env": true, ".cfg": true,
	".conf": true, ".ini": true, ".properties": true,
	".vue": true, ".svelte": true, ".astro": true, ".php": true,
	".mod": true, ".sum": true,
}

func isTextFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		lower := strings.ToLower(name)
		return lower == "makefile" || lower == "dockerfile" || lower == "readme" || lower == "license"
	}
	return textExtensions[ext]
}

func sortResults(results []Result) {
	for i := 1; i < len(results); i++ {
		key := results[i]
		j := i - 1
		for j >= 0 && results[j].Score < key.Score {
			results[j+1] = results[j]
			j--
		}
		results[j+1] = key
	}
}
