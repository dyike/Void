//go:build darwin

package repo

/*
#cgo CFLAGS: -x objective-c -fblocks
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices -framework AppKit

#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <ApplicationServices/ApplicationServices.h>
#include <AppKit/AppKit.h>
#include <dispatch/dispatch.h>
#include <fcntl.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/file.h>
#include <sys/time.h>
#include <unistd.h>

// ── Configurable hotkey ───────────────────────────────────────────────────

static CGKeyCode  g_keyCode   = 49;                          // Space
static int        g_modifiers = kCGEventFlagMaskAlternate;   // Option

void setHotkeyConfig(int keyCode, int modifiers) {
    g_keyCode   = (CGKeyCode)keyCode;
    g_modifiers = modifiers;
}

// ── Snippet character buffer ──────────────────────────────────────────────

#define SNIPPET_BUF_SIZE 64
static char g_snippet_buf[SNIPPET_BUF_SIZE];
static int  g_snippet_len = 0;
static volatile int g_debug_keycount = 0;

#define MAX_SNIPPET_RULES 128
static char* g_snippet_prefixes[MAX_SNIPPET_RULES];
static char* g_snippet_contents[MAX_SNIPPET_RULES];
static int   g_snippet_rule_count = 0;
static pthread_mutex_t g_snippet_mutex = PTHREAD_MUTEX_INITIALIZER;

static int getDebugKeyCount(void) { return g_debug_keycount; }

static void sendBackspaces(int n);
static void pasteClipboard(void);
static void setClipboardText(const char *text);
static char* getClipboardText(void);

static char keycodeToChar(CGKeyCode kc) {
    switch (kc) {
        case  0: return 'a'; case  1: return 's'; case  2: return 'd';
        case  3: return 'f'; case  4: return 'h'; case  5: return 'g';
        case  6: return 'z'; case  7: return 'x'; case  8: return 'c';
        case  9: return 'v'; case 11: return 'b'; case 12: return 'q';
        case 13: return 'w'; case 14: return 'e'; case 15: return 'r';
        case 16: return 'y'; case 17: return 't'; case 18: return '1';
        case 19: return '2'; case 20: return '3'; case 21: return '4';
        case 22: return '6'; case 23: return '5'; case 24: return '=';
        case 25: return '9'; case 26: return '7'; case 27: return '-';
        case 28: return '8'; case 29: return '0'; case 30: return ']';
        case 31: return 'o'; case 32: return 'u'; case 33: return '[';
        case 34: return 'i'; case 35: return 'p'; case 37: return 'l';
        case 38: return 'j'; case 39: return '\'';case 40: return 'k';
        case 41: return ';'; case 42: return '\\';case 43: return ',';
        case 44: return '/'; case 45: return 'n'; case 46: return 'm';
        case 47: return '.'; case 50: return '`';
        case 49: return ' ';
        default: return 0;
    }
}

static char unicodeToSnippetChar(UniChar ch) {
    if (ch >= 32 && ch <= 126) {
        return (char)ch;
    }
    // Chinese input sources may emit full-width ASCII such as "；addr".
    // Keep snippet triggers comparable with user-configured half-width prefixes.
    if (ch >= 0xFF01 && ch <= 0xFF5E) {
        return (char)(ch - 0xFEE0);
    }
    if (ch == 0x3000) {
        return ' ';
    }
    return 0;
}

static char eventToSnippetChar(CGEventRef event) {
    UniChar chars[4];
    UniCharCount actualLength = 0;
    CGEventKeyboardGetUnicodeString(event, 4, &actualLength, chars);
    if (actualLength == 1) {
        return unicodeToSnippetChar(chars[0]);
    }
    return 0;
}

static void snippetBufAppend(char c) {
    if (g_snippet_len < SNIPPET_BUF_SIZE - 1) {
        g_snippet_buf[g_snippet_len++] = c;
    } else {
        for (int i = 0; i < SNIPPET_BUF_SIZE - 2; i++) {
            g_snippet_buf[i] = g_snippet_buf[i + 1];
        }
        g_snippet_buf[SNIPPET_BUF_SIZE - 2] = c;
    }
    g_snippet_buf[g_snippet_len] = '\0';
}

static void snippetBufClear(void) {
    g_snippet_len = 0;
    g_snippet_buf[0] = '\0';
}

static const char* getSnippetBuffer(void) {
    return g_snippet_buf;
}

static void clearSnippetBuffer(void) {
    snippetBufClear();
}

static void clearSnippetRules(void) {
    pthread_mutex_lock(&g_snippet_mutex);
    for (int i = 0; i < g_snippet_rule_count; i++) {
        if (g_snippet_prefixes[i]) free(g_snippet_prefixes[i]);
        if (g_snippet_contents[i]) free(g_snippet_contents[i]);
        g_snippet_prefixes[i] = NULL;
        g_snippet_contents[i] = NULL;
    }
    g_snippet_rule_count = 0;
    pthread_mutex_unlock(&g_snippet_mutex);
}

static void addSnippetRule(const char *prefix, const char *content) {
    if (prefix == NULL || content == NULL || prefix[0] == '\0') return;

    pthread_mutex_lock(&g_snippet_mutex);
    if (g_snippet_rule_count < MAX_SNIPPET_RULES) {
        g_snippet_prefixes[g_snippet_rule_count] = strdup(prefix);
        g_snippet_contents[g_snippet_rule_count] = strdup(content);
        g_snippet_rule_count++;
    }
    pthread_mutex_unlock(&g_snippet_mutex);
}

static int snippetBufHasSuffix(const char *suffix) {
    size_t bufLen = strlen(g_snippet_buf);
    size_t suffixLen = strlen(suffix);
    if (suffixLen == 0 || suffixLen > bufLen) return 0;
    return strcmp(g_snippet_buf + (bufLen - suffixLen), suffix) == 0;
}

static void performSnippetExpansion(int triggerLen, const char *replacement) {
    if (triggerLen <= 0 || replacement == NULL) return;

    char *replacementCopy = strdup(replacement);
    dispatch_after(dispatch_time(DISPATCH_TIME_NOW, 30 * NSEC_PER_MSEC), dispatch_get_main_queue(), ^{
        char *oldClip = getClipboardText();

        sendBackspaces(triggerLen);
        usleep(50000);

        setClipboardText(replacementCopy);
        usleep(30000);

        pasteClipboard();
        usleep(2000000);

        if (oldClip != NULL) {
            setClipboardText(oldClip);
            free(oldClip);
        }
        free(replacementCopy);
    });
}

static long long nowMillis(void) {
    struct timeval tv;
    gettimeofday(&tv, NULL);
    return ((long long)tv.tv_sec * 1000LL) + ((long long)tv.tv_usec / 1000LL);
}

static int claimSnippetExpansion(const char *prefix) {
    if (prefix == NULL) return 0;

    int fd = open("/tmp/void-snippet-expand.lock", O_CREAT | O_RDWR, 0600);
    if (fd < 0) return 1;
    if (flock(fd, LOCK_EX | LOCK_NB) != 0) {
        close(fd);
        return 0;
    }

    char buf[256] = {0};
    ssize_t n = read(fd, buf, sizeof(buf) - 1);
    if (n < 0) n = 0;
    buf[n] = '\0';

    long long lastMs = 0;
    char lastPrefix[128] = {0};
    sscanf(buf, "%lld %127s", &lastMs, lastPrefix);

    long long now = nowMillis();
    int duplicate = (strcmp(lastPrefix, prefix) == 0 && lastMs > 0 && (now - lastMs) < 1500);
    if (!duplicate) {
        ftruncate(fd, 0);
        lseek(fd, 0, SEEK_SET);
        dprintf(fd, "%lld %s\n", now, prefix);
    }

    flock(fd, LOCK_UN);
    close(fd);
    return duplicate ? 0 : 1;
}

static void tryExpandSnippetRules(void) {
    pthread_mutex_lock(&g_snippet_mutex);
    for (int i = 0; i < g_snippet_rule_count; i++) {
        const char *prefix = g_snippet_prefixes[i];
        const char *content = g_snippet_contents[i];
        if (prefix == NULL || content == NULL) continue;

        if (snippetBufHasSuffix(prefix)) {
            char *replacement = strdup(content);
            int triggerLen = (int)strlen(prefix);
            int claimed = claimSnippetExpansion(prefix);
            fprintf(stderr, "snippet: matched prefix \"%s\" replacementLen=%zu pid=%d%s\n", prefix, strlen(content), getpid(), claimed ? "" : " (duplicate suppressed)");
            pthread_mutex_unlock(&g_snippet_mutex);
            if (claimed) {
                performSnippetExpansion(triggerLen, replacement);
            }
            free(replacement);
            snippetBufClear();
            return;
        }

        const char delimiters[] = {' ', '\t', '\n'};
        for (int d = 0; d < 3; d++) {
            char delimiter = delimiters[d];
            size_t prefixLen = strlen(prefix);
            char trigger[SNIPPET_BUF_SIZE];
            if (prefixLen + 1 >= sizeof(trigger)) continue;
            memcpy(trigger, prefix, prefixLen);
            trigger[prefixLen] = delimiter;
            trigger[prefixLen + 1] = '\0';

            if (snippetBufHasSuffix(trigger)) {
                size_t contentLen = strlen(content);
                char *replacement = (char*)malloc(contentLen + 2);
                if (replacement == NULL) continue;
                memcpy(replacement, content, contentLen);
                replacement[contentLen] = delimiter;
                replacement[contentLen + 1] = '\0';
                int triggerLen = (int)(prefixLen + 1);
                int claimed = claimSnippetExpansion(prefix);
                fprintf(stderr, "snippet: matched prefix \"%s\" with delimiter replacementLen=%zu pid=%d%s\n", prefix, strlen(content), getpid(), claimed ? "" : " (duplicate suppressed)");
                pthread_mutex_unlock(&g_snippet_mutex);
                if (claimed) {
                    performSnippetExpansion(triggerLen, replacement);
                }
                free(replacement);
                snippetBufClear();
                return;
            }
        }
    }
    pthread_mutex_unlock(&g_snippet_mutex);
}

// ── Synthetic key events ──────────────────────────────────────────────────

static void sendBackspaces(int n) {
    for (int i = 0; i < n; i++) {
        CGEventRef down = CGEventCreateKeyboardEvent(NULL, 51, true);
        CGEventRef up   = CGEventCreateKeyboardEvent(NULL, 51, false);
        CGEventPost(kCGAnnotatedSessionEventTap, down);
        CGEventPost(kCGAnnotatedSessionEventTap, up);
        CFRelease(down);
        CFRelease(up);
        usleep(5000);
    }
}

static void pasteClipboard(void) {
    CGEventRef down = CGEventCreateKeyboardEvent(NULL, 9, true);
    CGEventRef up   = CGEventCreateKeyboardEvent(NULL, 9, false);
    CGEventSetFlags(down, kCGEventFlagMaskCommand);
    CGEventSetFlags(up, kCGEventFlagMaskCommand);
    CGEventPost(kCGAnnotatedSessionEventTap, down);
    CGEventPost(kCGAnnotatedSessionEventTap, up);
    CFRelease(down);
    CFRelease(up);
}

// ── Event tap ─────────────────────────────────────────────────────────────

static CFMachPortRef gTap;

static volatile int hotkey_fired = 0;
static volatile int g_all_events = 0;

static int getAllEventCount(void) { return g_all_events; }

static CGEventRef eventTapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *userInfo) {
    if (type == kCGEventTapDisabledByTimeout || type == kCGEventTapDisabledByUserInput) {
        if (gTap) CGEventTapEnable(gTap, true);
        return event;
    }

    if (type == kCGEventKeyDown) {
        g_all_events++;
        CGKeyCode keyCode = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
        CGEventFlags flags = CGEventGetFlags(event);
        int masked = (int)(flags & (kCGEventFlagMaskAlternate | kCGEventFlagMaskCommand |
                                    kCGEventFlagMaskControl   | kCGEventFlagMaskShift));

        if (keyCode == g_keyCode && masked == g_modifiers) {
            hotkey_fired = 1;
            return NULL;
        }

        int nonShiftMods = masked & ~kCGEventFlagMaskShift;
        if (nonShiftMods == 0) {
            g_debug_keycount++;
            if (keyCode == 53) {
                snippetBufClear();
            } else if (keyCode == 51) {
                if (g_snippet_len > 0) {
                    g_snippet_len--;
                    g_snippet_buf[g_snippet_len] = '\0';
                }
            } else if (keyCode == 36) {
                snippetBufAppend('\n');
                tryExpandSnippetRules();
            } else if (keyCode == 48) {
                snippetBufAppend('\t');
                tryExpandSnippetRules();
            } else {
                char c = eventToSnippetChar(event);
                if (c == 0) {
                    c = keycodeToChar(keyCode);
                }
                if (c != 0) {
                    int shifted = ((masked & kCGEventFlagMaskShift) != 0) && eventToSnippetChar(event) == 0;
                    if (shifted) {
                        switch (c) {
                            case ';': c = ':'; break;
                            case '\'': c = '"'; break;
                            case ',': c = '<'; break;
                            case '.': c = '>'; break;
                            case '/': c = '?'; break;
                            case '1': c = '!'; break;
                            case '2': c = '@'; break;
                            case '3': c = '#'; break;
                            case '4': c = '$'; break;
                            case '5': c = '%'; break;
                            case '6': c = '^'; break;
                            case '7': c = '&'; break;
                            case '8': c = '*'; break;
                            case '9': c = '('; break;
                            case '0': c = ')'; break;
                            case '-': c = '_'; break;
                            case '=': c = '+'; break;
                            case '[': c = '{'; break;
                            case ']': c = '}'; break;
                            case '\\': c = '|'; break;
                            case '`': c = '~'; break;
                            default:
                                if (c >= 'a' && c <= 'z') c -= 32;
                                break;
                        }
                    }
                    snippetBufAppend(c);
                    tryExpandSnippetRules();
                }
            }
        } else {
            snippetBufClear();
        }
    }
    return event;
}

static CFRunLoopSourceRef gRunLoopSrc = NULL;

static int startEventTap(void) {
    CGEventMask mask = CGEventMaskBit(kCGEventKeyDown);
    gTap = CGEventTapCreate(
        kCGHIDEventTap,
        kCGHeadInsertEventTap,
        kCGEventTapOptionDefault,
        mask,
        eventTapCallback,
        NULL
    );
    if (!gTap) return -1;
    gRunLoopSrc = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, gTap, 0);
    CFRunLoopAddSource(CFRunLoopGetMain(), gRunLoopSrc, kCFRunLoopCommonModes);
    CGEventTapEnable(gTap, true);
    return 0;
}

static void stopEventTap(void) {
    if (gTap) { CGEventTapEnable(gTap, false); CFRelease(gTap); gTap = NULL; }
    if (gRunLoopSrc) { CFRelease(gRunLoopSrc); gRunLoopSrc = NULL; }
}

static int pollHotkey(void) {
    if (hotkey_fired) { hotkey_fired = 0; return 1; }
    return 0;
}

// ── Clipboard helpers ─────────────────────────────────────────────────────

static void setClipboardText(const char *text) {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    [pb clearContents];
    [pb setString:[NSString stringWithUTF8String:text] forType:NSPasteboardTypeString];
}

static char* getClipboardText(void) {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    NSString *s = [pb stringForType:NSPasteboardTypeString];
    if (s == nil) return NULL;
    return strdup([s UTF8String]);
}

// ── Accessibility ─────────────────────────────────────────────────────────

static int checkAccessibility(int prompt) {
    CFStringRef key = CFSTR("AXTrustedCheckOptionPrompt");
    CFBooleanRef val = prompt ? kCFBooleanTrue : kCFBooleanFalse;
    CFDictionaryRef opts = CFDictionaryCreate(NULL,
        (const void **)&key, (const void **)&val,
        1, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    int trusted = AXIsProcessTrustedWithOptions(opts) ? 1 : 0;
    CFRelease(opts);
    return trusted;
}

static int checkListenEventAccess(void) {
    return CGPreflightListenEventAccess() ? 1 : 0;
}

static int requestListenEventAccess(void) {
    return CGRequestListenEventAccess() ? 1 : 0;
}
*/
import "C"
import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/dyike/Void/internal/model"
)

// ─── Key code table ──────────────────────────────────────────────────────

var keyCodes = map[string]int{
	"space": 49, "return": 36, "enter": 36, "tab": 48,
	"delete": 51, "backspace": 51, "escape": 53, "esc": 53,
	"capslock": 57, "home": 115, "end": 119,
	"pageup": 116, "pagedown": 121,
	"left": 123, "right": 124, "down": 125, "up": 126,
	"f1": 122, "f2": 120, "f3": 99, "f4": 118,
	"f5": 96, "f6": 97, "f7": 98, "f8": 100,
	"f9": 101, "f10": 109, "f11": 103, "f12": 111,
	"a": 0, "b": 11, "c": 8, "d": 2, "e": 14,
	"f": 3, "g": 5, "h": 4, "i": 34, "j": 38,
	"k": 40, "l": 37, "m": 46, "n": 45, "o": 31,
	"p": 35, "q": 12, "r": 15, "s": 1, "t": 17,
	"u": 32, "v": 9, "w": 13, "x": 7, "y": 16, "z": 6,
	"0": 29, "1": 18, "2": 19, "3": 20, "4": 21,
	"5": 23, "6": 22, "7": 26, "8": 28, "9": 25,
	"`": 50, "-": 27, "=": 24, "[": 33, "]": 30,
	"\\": 42, ";": 41, "'": 39, ",": 43, ".": 47, "/": 44,
}

var modifierFlags = map[string]int{
	"option": 0x00080000, "alt": 0x00080000,
	"cmd": 0x00100000, "command": 0x00100000,
	"ctrl": 0x00040000, "control": 0x00040000,
	"shift": 0x00020000,
}

var modifierSymbols = map[string]string{
	"option": "⌥", "alt": "⌥",
	"cmd": "⌘", "command": "⌘",
	"ctrl": "⌃", "control": "⌃",
	"shift": "⇧",
}

var keyDisplayNames = map[string]string{
	"space": "Space", "return": "↩", "enter": "↩",
	"tab": "⇥", "delete": "⌫", "backspace": "⌫",
	"escape": "⎋", "esc": "⎋",
	"up": "↑", "down": "↓", "left": "←", "right": "→",
	"pageup": "⇞", "pagedown": "⇟", "home": "↖", "end": "↘",
}

// ─── HotkeyRepo ──────────────────────────────────────────────────────────

// HotkeyRepo provides low-level hotkey, event tap, and snippet buffer operations.
type HotkeyRepo struct{}

// NewHotkeyRepo creates a new hotkey repo.
func NewHotkeyRepo() *HotkeyRepo { return &HotkeyRepo{} }

// ParseHotkey parses a hotkey spec string into a KeySpec.
func (r *HotkeyRepo) ParseHotkey(spec string) (*model.KeySpec, error) {
	spec = strings.ToLower(strings.TrimSpace(spec))
	parts := strings.Split(spec, "+")

	var mods []string
	var key string
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if i == len(parts)-1 {
			key = p
		} else {
			mods = append(mods, p)
		}
	}
	if key == "\u00a0" || key == " " {
		key = "space"
	}
	if key == "" {
		return nil, fmt.Errorf("no key specified")
	}
	if _, ok := keyCodes[key]; !ok {
		return nil, fmt.Errorf("unknown key: %q", key)
	}

	var display string
	for _, m := range mods {
		if sym, ok := modifierSymbols[m]; ok {
			display += sym
		} else {
			return nil, fmt.Errorf("unknown modifier: %q", m)
		}
	}
	if disp, ok := keyDisplayNames[key]; ok {
		display += disp
	} else {
		display += strings.ToUpper(key)
	}

	return &model.KeySpec{Key: key, Modifiers: strings.Join(mods, ","), Display: display}, nil
}

// FormatSpec formats a KeySpec back into a spec string.
func (r *HotkeyRepo) FormatSpec(ks model.KeySpec) string {
	if ks.Modifiers == "" {
		return ks.Key
	}
	return strings.ReplaceAll(ks.Modifiers, ",", "+") + "+" + ks.Key
}

// HasPermission checks if the app has Accessibility permission.
func (r *HotkeyRepo) HasPermission() bool { return C.checkAccessibility(0) != 0 }

// RequestPermission prompts the user to grant Accessibility permission.
func (r *HotkeyRepo) RequestPermission() { C.checkAccessibility(1) }

// HasListenEventPermission checks if the app has Input Monitoring permission.
func (r *HotkeyRepo) HasListenEventPermission() bool {
	return C.checkListenEventAccess() != 0
}

// RequestListenEventPermission prompts the user to grant Input Monitoring permission.
func (r *HotkeyRepo) RequestListenEventPermission() bool {
	return C.requestListenEventAccess() != 0
}

// ConfigureKey sets the hotkey in the C event tap.
func (r *HotkeyRepo) ConfigureKey(keySpec string) bool {
	ks, err := r.ParseHotkey(keySpec)
	if err != nil {
		ks = &model.DefaultHotkey
	}
	keyCode, ok := keyCodes[ks.Key]
	if !ok {
		keyCode = keyCodes["space"]
	}
	modMask := 0
	for _, m := range strings.Split(ks.Modifiers, ",") {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if f, ok := modifierFlags[m]; ok {
			modMask |= f
		}
	}
	if modMask == 0 {
		modMask = modifierFlags["option"]
	}
	C.setHotkeyConfig(C.int(keyCode), C.int(modMask))
	return true
}

// StartEventTap installs the CGEventTap. Returns false on failure.
func (r *HotkeyRepo) StartEventTap() bool {
	return C.startEventTap() == 0
}

// StopEventTap removes the CGEventTap.
func (r *HotkeyRepo) StopEventTap() {
	C.stopEventTap()
}

// PollHotkey returns true if the hotkey was fired since the last poll.
func (r *HotkeyRepo) PollHotkey() bool {
	return C.pollHotkey() != 0
}

// ClearSnippetBuffer resets the snippet character buffer.
func (r *HotkeyRepo) ClearSnippetBuffer() {
	C.clearSnippetBuffer()
}

// ClearSnippetRules removes all active auto-expansion rules from the event tap.
func (r *HotkeyRepo) ClearSnippetRules() {
	C.clearSnippetRules()
}

// AddSnippetRule registers one auto-expansion rule in the event tap.
func (r *HotkeyRepo) AddSnippetRule(prefix, content string) {
	cPrefix := C.CString(prefix)
	cContent := C.CString(content)
	C.addSnippetRule(cPrefix, cContent)
	C.free(unsafe.Pointer(cPrefix))
	C.free(unsafe.Pointer(cContent))
}

// NormalizeSnippetTriggerText maps full-width ASCII to half-width text.
func NormalizeSnippetTriggerText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E:
			b.WriteRune(r - 0xFEE0)
		case r == 0x3000:
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

type snippetTriggerMatch struct {
	prefix     string
	deleteText string
	keepText   string
}

func matchSnippetTrigger(buf string, prefixes []string) (snippetTriggerMatch, bool) {
	buf = NormalizeSnippetTriggerText(buf)
	for _, prefix := range prefixes {
		prefix = NormalizeSnippetTriggerText(prefix)
		if prefix == "" {
			continue
		}
		if strings.HasSuffix(buf, prefix) {
			return snippetTriggerMatch{
				prefix:     prefix,
				deleteText: prefix,
			}, true
		}

		for _, delimiter := range []string{" ", "\t", "\n"} {
			trigger := prefix + delimiter
			if strings.HasSuffix(buf, trigger) {
				return snippetTriggerMatch{
					prefix:     prefix,
					deleteText: trigger,
					keepText:   delimiter,
				}, true
			}
		}
	}
	return snippetTriggerMatch{}, false
}

// GetClipboardText returns the current clipboard text via AppKit.
func (r *HotkeyRepo) GetClipboardText() string {
	p := C.getClipboardText()
	if p == nil {
		return ""
	}
	s := C.GoString(p)
	C.free(unsafe.Pointer(p))
	return s
}

// SetClipboardText sets the clipboard text via AppKit.
func (r *HotkeyRepo) SetClipboardText(text string) {
	cs := C.CString(text)
	C.setClipboardText(cs)
	C.free(unsafe.Pointer(cs))
}

// PasteClipboard sends a Cmd+V key event to paste clipboard content.
func (r *HotkeyRepo) PasteClipboard() {
	C.pasteClipboard()
}
