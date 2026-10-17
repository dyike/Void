package repo

import (
	"bytes"
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dyike/Void/internal/model"
)

func applicationRoots() []string {
	return []string{"/Applications", "/System/Applications", filepath.Join(os.Getenv("HOME"), "Applications")}
}

// Filter the displayed installation path, before resolving Nix/Home Manager
// symlinks. Their store targets are valid, but backup installations are not.
func installedApplication(path string, roots []string) bool {
	if !strings.HasSuffix(strings.ToLower(path), ".app") {
		return false
	}
	for _, root := range roots {
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		parts := strings.Split(rel, string(filepath.Separator))
		for i, part := range parts {
			p := strings.ToLower(part)
			if strings.HasPrefix(p, ".") || strings.Contains(p, ".backup") || strings.HasSuffix(p, ".bak") || p == "backup" || p == "backups" {
				return false
			}
			if i < len(parts)-1 && (strings.HasSuffix(p, ".app") || strings.HasSuffix(p, ".bundle")) {
				return false
			}
		}
		return true
	}
	return false
}

// Follow directory aliases used by package managers, never descend into an app
// bundle, and visit each real directory once to avoid symlink loops.
func scanApplicationPaths(roots []string) []string {
	var paths []string
	seen := map[string]bool{}
	var scan func(string, int)
	scan = func(dir string, depth int) {
		if depth > 8 {
			return
		}
		real, err := filepath.EvalSymlinks(dir)
		if err != nil || seen[real] {
			return
		}
		seen[real] = true
		entries, _ := os.ReadDir(dir)
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())
			if strings.HasSuffix(strings.ToLower(entry.Name()), ".app") {
				paths = append(paths, path)
			} else if installedApplication(filepath.Join(path, "candidate.app"), roots) {
				scan(path, depth+1)
			}
		}
	}
	for _, root := range roots {
		scan(root, 0)
	}
	return paths
}

type appMetadata struct {
	bundleID   string
	background bool
}

func readAppMetadata(path string) (appMetadata, bool) {
	file := filepath.Join(path, "Contents", "Info.plist")
	data, err := os.ReadFile(file)
	if err != nil {
		return appMetadata{}, false // stale Spotlight entries and broken symlinks
	}
	if bytes.HasPrefix(data, []byte("bplist")) {
		data, err = exec.Command("/usr/bin/plutil", "-convert", "xml1", "-o", "-", file).Output()
		if err != nil {
			return appMetadata{}, false
		}
	}
	var result appMetadata
	d := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := d.Token()
		if err == io.EOF {
			return result, true
		}
		if err != nil {
			return appMetadata{}, false
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "key" {
			continue
		}
		var key string
		if d.DecodeElement(&key, &start) != nil {
			return appMetadata{}, false
		}
		for {
			token, err = d.Token()
			if err != nil {
				return appMetadata{}, false
			}
			if start, ok = token.(xml.StartElement); ok {
				break
			}
		}
		var value string
		if d.DecodeElement(&value, &start) != nil {
			return appMetadata{}, false
		}
		switch key {
		case "CFBundleIdentifier":
			result.bundleID = strings.ToLower(value)
		case "LSBackgroundOnly":
			result.background = start.Name.Local == "true" || value == "1" || strings.EqualFold(value, "true")
		}
	}
}

func (r *CommandRepo) appsFromPaths(paths, roots []string) []model.CommandInfo {
	// Choose a stable installation: shared Applications before user-local
	// copies, then shorter paths before deeper aliases, independent of Spotlight order.
	sort.Slice(paths, func(i, j int) bool {
		rank := func(path string) int {
			for i, root := range roots {
				if strings.HasPrefix(path, root+string(filepath.Separator)) {
					return i
				}
			}
			return len(roots)
		}
		if a, b := rank(paths[i]), rank(paths[j]); a != b {
			return a < b
		}
		if len(paths[i]) != len(paths[j]) {
			return len(paths[i]) < len(paths[j])
		}
		return paths[i] < paths[j]
	})
	seenPath, seenBundle, seenID := map[string]bool{}, map[string]bool{}, map[string]bool{}
	var result []model.CommandInfo
	for _, path := range paths {
		path = filepath.Clean(strings.TrimSpace(path))
		if !installedApplication(path, roots) {
			continue
		}
		real, err := filepath.EvalSymlinks(path)
		if err != nil || seenPath[real] {
			continue
		}
		metadata, ok := readAppMetadata(path)
		if !ok || metadata.background || metadata.bundleID != "" && seenBundle[metadata.bundleID] {
			continue
		}
		seenPath[real] = true
		if metadata.bundleID != "" {
			seenBundle[metadata.bundleID] = true
		}
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		id := "app-" + slugify(name) // retain existing pins and usage history
		if seenID[id] {
			id += fmtAppSuffix(metadata.bundleID, real)
		}
		seenID[id] = true
		result = append(result, model.CommandInfo{ID: id, Title: name, Subtitle: filepath.Dir(path), Category: "app", Path: path, IconDataURL: r.iconForApp(path)})
	}
	sort.SliceStable(result, func(i, j int) bool { return strings.ToLower(result[i].Title) < strings.ToLower(result[j].Title) })
	return result
}

func fmtAppSuffix(bundleID, real string) string {
	if bundleID == "" {
		bundleID = real
	}
	return fmt.Sprintf("-%x", md5.Sum([]byte(bundleID)))[:9]
}
