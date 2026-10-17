package repo

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework AppKit

#include <AppKit/AppKit.h>
#include <stdlib.h>

static long long clipboardChangeCount(void) {
    @autoreleasepool {
        return (long long)[[NSPasteboard generalPasteboard] changeCount];
    }
}

static char* clipboardFilePaths(void) {
    @autoreleasepool {
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        NSDictionary *options = @{NSPasteboardURLReadingFileURLsOnlyKey: @YES};
        NSArray *urls = [pb readObjectsForClasses:@[[NSURL class]] options:options];
        if (urls == nil || [urls count] == 0) return NULL;

        NSMutableArray *paths = [NSMutableArray arrayWithCapacity:[urls count]];
        for (NSURL *url in urls) {
            if ([url isFileURL] && [url path] != nil) {
                [paths addObject:[url path]];
            }
        }
        if ([paths count] == 0) return NULL;

        NSString *joined = [paths componentsJoinedByString:@"\n"];
        return strdup([joined UTF8String]);
    }
}

static int clipboardPNGToFile(const char *path) {
    @autoreleasepool {
        if (path == NULL) return 0;
        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        NSData *pngData = [pb dataForType:NSPasteboardTypePNG];

        if (pngData == nil) {
            NSData *tiffData = [pb dataForType:NSPasteboardTypeTIFF];
            if (tiffData != nil) {
                NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:tiffData];
                if (rep != nil) {
                    pngData = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
                }
            }
        }
        if (pngData == nil || [pngData length] == 0) return 0;

        NSString *dest = [NSString stringWithUTF8String:path];
        return [pngData writeToFile:dest atomically:YES] ? 1 : 0;
    }
}

static int setClipboardFilePaths(const char *pathsText) {
    @autoreleasepool {
        if (pathsText == NULL) return 0;
        NSString *raw = [NSString stringWithUTF8String:pathsText];
        NSArray *lines = [raw componentsSeparatedByCharactersInSet:[NSCharacterSet newlineCharacterSet]];
        NSMutableArray *urls = [NSMutableArray array];

        for (NSString *line in lines) {
            if ([line length] == 0) continue;
            NSURL *url = [NSURL fileURLWithPath:line];
            if (url != nil) [urls addObject:url];
        }
        if ([urls count] == 0) return 0;

        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        [pb clearContents];
        return [pb writeObjects:urls] ? 1 : 0;
    }
}

static int setClipboardPNGFile(const char *path) {
    @autoreleasepool {
        if (path == NULL) return 0;
        NSString *src = [NSString stringWithUTF8String:path];
        NSData *data = [NSData dataWithContentsOfFile:src];
        if (data == nil || [data length] == 0) return 0;

        NSPasteboard *pb = [NSPasteboard generalPasteboard];
        [pb clearContents];
        return [pb setData:data forType:NSPasteboardTypePNG] ? 1 : 0;
    }
}
*/
import "C"
import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/dyike/Void/internal/model"
)

const maxClipboardHistory = 100

// ClipboardRepo manages clipboard history persistence and read/write operations.
type ClipboardRepo struct {
	readMu        sync.Mutex
	clipboardRead clipboardReadState
	mu            sync.RWMutex
	history       []model.ClipboardItem
	dataDir       string
}

// NewClipboardRepo creates a new clipboard repo.
func NewClipboardRepo(dataDir string) *ClipboardRepo {
	_ = os.MkdirAll(dataDir, 0755)
	r := &ClipboardRepo{dataDir: dataDir}
	r.loadFromDisk()
	return r
}

// LastContent returns the content of the most recently added item (for dedup).
func (r *ClipboardRepo) LastContent() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.history) > 0 {
		return r.history[0].Content
	}
	return ""
}

// LastFingerprint returns the most recently added clipboard fingerprint.
func (r *ClipboardRepo) LastFingerprint() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.history) > 0 {
		return clipboardFingerprint(r.history[0])
	}
	return ""
}

// GetHistory returns all clipboard history items (newest first).
func (r *ClipboardRepo) GetHistory() []model.ClipboardItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]model.ClipboardItem, len(r.history))
	copy(result, r.history)
	return result
}

// AddItem inserts a new clipboard item into history.
func (r *ClipboardRepo) AddItem(content, ctype string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	content = normalizeClipboardContent(content, ctype)
	if content == "" {
		return
	}
	preview := clipboardPreview(content, ctype)
	item := model.ClipboardItem{
		ID:        clipboardNewID(),
		Type:      ctype,
		Content:   content,
		Preview:   preview,
		Timestamp: time.Now().UnixMilli(),
	}

	itemKey := clipboardFingerprint(item)
	deduped := r.history[:0]
	for _, existing := range r.history {
		if clipboardFingerprint(existing) == itemKey {
			continue
		}
		deduped = append(deduped, existing)
	}
	r.history = deduped
	r.history = append([]model.ClipboardItem{item}, r.history...)
	if len(r.history) > maxClipboardHistory {
		r.history = r.history[:maxClipboardHistory]
	}
	r.saveToDisk()
}

// FindByID returns a clipboard item by ID.
func (r *ClipboardRepo) FindByID(id string) *model.ClipboardItem {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.history {
		if item.ID == id {
			cp := item
			return &cp
		}
	}
	return nil
}

// Delete removes a clipboard item from history.
func (r *ClipboardRepo) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, item := range r.history {
		if item.ID == id {
			r.history = append(r.history[:i], r.history[i+1:]...)
			r.saveToDisk()
			return nil
		}
	}
	return fmt.Errorf("item not found: %s", id)
}

// Clear removes all clipboard history.
func (r *ClipboardRepo) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history = nil
	r.saveToDisk()
}

// ReadClipboardIfChanged avoids decoding images and writing assets again while
// NSPasteboard's change count is unchanged. File/text/image priority is retained.
func (r *ClipboardRepo) ReadClipboardIfChanged() (content, ctype string) {
	r.readMu.Lock()
	defer r.readMu.Unlock()
	return r.clipboardRead.read(func() int64 { return int64(C.clipboardChangeCount()) }, r.ReadClipboard)
}

// ReadClipboard reads the current system clipboard content.
func (r *ClipboardRepo) ReadClipboard() (content, ctype string) {
	if paths := readClipboardFilePaths(); paths != "" {
		return paths, "file"
	}
	if imagePath := r.readClipboardImage(); imagePath != "" {
		return imagePath, "image"
	}

	out, err := exec.Command("pbpaste", "-Prefer", "txt").Output()
	if err != nil {
		out, err = exec.Command("pbpaste").Output()
	}
	if err == nil {
		if text, ok := normalizeClipboardText(out); ok {
			ctype := "text"
			if isURL(text) {
				ctype = "url"
			}
			return text, ctype
		}
	}
	return "", ""
}

// WriteToClipboard writes content to the system clipboard via pbcopy.
func (r *ClipboardRepo) WriteToClipboard(content string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(content)
	return cmd.Run()
}

// WriteItemToClipboard writes a typed history item back to the system clipboard.
func (r *ClipboardRepo) WriteItemToClipboard(item model.ClipboardItem) error {
	switch item.Type {
	case "file":
		if writeClipboardFilePaths(item.Content) {
			return nil
		}
	case "image":
		if writeClipboardPNGFile(item.Content) {
			return nil
		}
	}
	return r.WriteToClipboard(item.Content)
}

func (r *ClipboardRepo) historyPath() string {
	return filepath.Join(r.dataDir, "clipboard-history.json")
}

func (r *ClipboardRepo) assetsDir() string {
	return filepath.Join(r.dataDir, "clipboard-assets")
}

func (r *ClipboardRepo) loadFromDisk() {
	data, err := os.ReadFile(r.historyPath())
	if err != nil {
		return
	}
	var items []model.ClipboardItem
	if err := json.Unmarshal(data, &items); err != nil {
		return
	}
	r.history = sanitizeHistory(items)
	if len(r.history) != len(items) {
		r.saveToDisk()
	}
}

func (r *ClipboardRepo) saveToDisk() {
	data, err := json.MarshalIndent(r.history, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(r.historyPath(), data, 0644)
}

func clipboardNewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func readClipboardFilePaths() string {
	p := C.clipboardFilePaths()
	if p == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(p))
	return normalizeFilePaths(C.GoString(p))
}

func (r *ClipboardRepo) readClipboardImage() string {
	if err := os.MkdirAll(r.assetsDir(), 0755); err != nil {
		return ""
	}
	tmp, err := os.CreateTemp(r.assetsDir(), "clipboard-image-*.png")
	if err != nil {
		return ""
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()

	cPath := C.CString(tmpPath)
	ok := C.clipboardPNGToFile(cPath) != 0
	C.free(unsafe.Pointer(cPath))
	if !ok {
		_ = os.Remove(tmpPath)
		return ""
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil || len(data) == 0 {
		_ = os.Remove(tmpPath)
		return ""
	}
	sum := sha256.Sum256(data)
	finalPath := filepath.Join(r.assetsDir(), hex.EncodeToString(sum[:])+".png")
	if err := os.Rename(tmpPath, finalPath); err != nil {
		if _, statErr := os.Stat(finalPath); statErr == nil {
			_ = os.Remove(tmpPath)
		} else {
			_ = os.Remove(tmpPath)
			return ""
		}
	}
	return finalPath
}

func writeClipboardFilePaths(paths string) bool {
	cPaths := C.CString(paths)
	defer C.free(unsafe.Pointer(cPaths))
	return C.setClipboardFilePaths(cPaths) != 0
}

func writeClipboardPNGFile(path string) bool {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))
	return C.setClipboardPNGFile(cPath) != 0
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func normalizeClipboardText(out []byte) (string, bool) {
	out = bytes.TrimRight(out, "\r\n")
	if len(out) == 0 || !utf8.Valid(out) {
		return "", false
	}

	text := string(out)
	if strings.TrimSpace(text) == "" || !isReadableText(text) {
		return "", false
	}
	return text, true
}

func isReadableText(text string) bool {
	for _, r := range text {
		if r == utf8.RuneError || r == '\uFFFD' {
			return false
		}
		if r == '\n' || r == '\r' || r == '\t' {
			continue
		}
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func sanitizeHistory(items []model.ClipboardItem) []model.ClipboardItem {
	result := make([]model.ClipboardItem, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		item.Content = normalizeClipboardContent(item.Content, item.Type)
		if item.Content == "" {
			continue
		}
		switch item.Type {
		case "text", "url", "":
			if !utf8.ValidString(item.Content) || !isReadableText(item.Content) {
				continue
			}
			preview := item.Preview
			if preview == "" || !utf8.ValidString(preview) || !isReadableText(preview) {
				preview = item.Content
			}
			item.Preview = truncateRunes(preview, 200)
		case "file":
			item.Preview = clipboardPreview(item.Content, item.Type)
		case "image":
			if _, err := os.Stat(item.Content); err != nil {
				continue
			}
			item.Preview = clipboardPreview(item.Content, item.Type)
		}
		key := clipboardFingerprint(item)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, item)
	}
	return result
}

func normalizeClipboardContent(content, ctype string) string {
	switch ctype {
	case "file":
		return normalizeFilePaths(content)
	default:
		return strings.TrimRight(content, "\r\n")
	}
}

func normalizeFilePaths(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	paths := make([]string, 0, len(lines))
	for _, line := range lines {
		path := strings.TrimSpace(line)
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	return strings.Join(paths, "\n")
}

func clipboardPreview(content, ctype string) string {
	switch ctype {
	case "file":
		paths := strings.Split(content, "\n")
		if len(paths) == 1 {
			return filepath.Base(paths[0])
		}
		return fmt.Sprintf("%s and %d more", filepath.Base(paths[0]), len(paths)-1)
	case "image":
		return filepath.Base(content)
	default:
		return truncateRunes(content, 200)
	}
}

func clipboardFingerprint(item model.ClipboardItem) string {
	content := normalizeClipboardContent(item.Content, item.Type)
	if item.Type == "url" {
		return "text:" + content
	}
	return item.Type + ":" + content
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}

	var b strings.Builder
	limit := max - 3
	if limit < 0 {
		limit = 0
	}
	count := 0
	for _, r := range s {
		if count >= limit {
			break
		}
		b.WriteRune(r)
		count++
	}
	b.WriteString("...")
	return b.String()
}
