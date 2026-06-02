//go:build darwin

package hotkey

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices -framework AppKit

#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <ApplicationServices/ApplicationServices.h>
#include <AppKit/AppKit.h>

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
static volatile int g_debug_keycount = 0;  // debug: count keyDown events that reach buffer code

static int getDebugKeyCount(void) { return g_debug_keycount; }

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

static char eventToASCIIChar(CGEventRef event) {
    UniChar chars[4];
    UniCharCount actualLength = 0;
    CGEventKeyboardGetUnicodeString(event, 4, &actualLength, chars);
    if (actualLength == 1 && chars[0] >= 32 && chars[0] <= 126) {
        return (char)chars[0];
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

static CFMachPortRef gTap;  // forward declaration for callback

static volatile int hotkey_fired = 0;
static volatile int g_all_events = 0;  // debug: count ALL events reaching callback

static int getAllEventCount(void) { return g_all_events; }

static CGEventRef eventTapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *userInfo) {
    // Re-enable tap if macOS disabled it due to timeout
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

        // Check hotkey
        if (keyCode == g_keyCode && masked == g_modifiers) {
            hotkey_fired = 1;
            return NULL;
        }

        // Buffer characters for snippet expansion (only no modifiers or Shift)
        int nonShiftMods = masked & ~kCGEventFlagMaskShift;
        if (nonShiftMods == 0) {
            g_debug_keycount++;
            if (keyCode == 53) {
                // Escape → clear buffer
                snippetBufClear();
            } else if (keyCode == 51) {
                // Backspace → remove last char
                if (g_snippet_len > 0) {
                    g_snippet_len--;
                    g_snippet_buf[g_snippet_len] = '\0';
                }
            } else if (keyCode == 36) {
                snippetBufAppend('\n');
            } else if (keyCode == 48) {
                snippetBufAppend('\t');
            } else {
                char c = eventToASCIIChar(event);
                if (c == 0) {
                    c = keycodeToChar(keyCode);
                }
                if (c != 0) {
                    int shifted = ((masked & kCGEventFlagMaskShift) != 0) && eventToASCIIChar(event) == 0;
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
	"log"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
	"unsafe"
)

// KeySpec describes a global hotkey combination.
type KeySpec struct {
	Key       string `json:"key"`
	Modifiers string `json:"modifiers"`
	Display   string `json:"display"`
}

// DefaultHotkey is the factory default: Option+Space.
var DefaultHotkey = KeySpec{Key: "space", Modifiers: "option", Display: "⌥Space"}

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

// ─── Parsing ──────────────────────────────────────────────────────────────

func ParseHotkey(spec string) (*KeySpec, error) {
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

	return &KeySpec{Key: key, Modifiers: strings.Join(mods, ","), Display: display}, nil
}

func FormatSpec(ks KeySpec) string {
	if ks.Modifiers == "" {
		return ks.Key
	}
	return strings.ReplaceAll(ks.Modifiers, ",", "+") + "+" + ks.Key
}

// ─── Service ─────────────────────────────────────────────────────────────

var running atomic.Bool

type Service struct{}

func NewService() *Service { return &Service{} }

func HasPermission() bool { return C.checkAccessibility(0) != 0 }
func RequestPermission()  { C.checkAccessibility(1) }
func HasListenEventPermission() bool {
	return C.checkListenEventAccess() != 0
}
func RequestListenEventPermission() bool {
	return C.requestListenEventAccess() != 0
}

func (s *Service) Start(h func(), keySpec string) bool {
	if !s.configureKey(keySpec) {
		return false
	}
	return s.startTap(h)
}

func (s *Service) Restart(h func(), keySpec string) bool {
	s.Stop()
	time.Sleep(50 * time.Millisecond)
	return s.Start(h, keySpec)
}

func (s *Service) configureKey(keySpec string) bool {
	ks, err := ParseHotkey(keySpec)
	if err != nil {
		log.Printf("hotkey: invalid spec %q: %v — using default", keySpec, err)
		ks = &DefaultHotkey
	}
	keyCode, ok := keyCodes[ks.Key]
	if !ok {
		log.Printf("hotkey: unknown key %q — using default", ks.Key)
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
	log.Printf("hotkey: configured as %s (keyCode=%d, modMask=0x%08x)", ks.Display, keyCode, modMask)
	return true
}

func (s *Service) startTap(h func()) bool {
	if !HasPermission() {
		log.Println("hotkey: requesting Accessibility permission…")
		RequestPermission()
		if !HasPermission() {
			log.Println("hotkey: Accessibility not granted — check System Settings → Privacy & Security → Accessibility")
			return false
		}
	}
	log.Println("hotkey: Accessibility granted")
	if !HasListenEventPermission() {
		log.Println("hotkey: requesting Input Monitoring permission…")
		if !RequestListenEventPermission() || !HasListenEventPermission() {
			log.Println("hotkey: Input Monitoring not granted — check System Settings → Privacy & Security → Input Monitoring")
			return false
		}
	}
	log.Println("hotkey: Input Monitoring granted")
	ret := C.startEventTap()
	if ret != 0 {
		log.Println("hotkey: CGEventTapCreate failed — Accessibility may need to be re-granted")
		return false
	}
	running.Store(true)
	log.Println("hotkey: CGEventTap active")

	go func() {
		for running.Load() {
			if C.pollHotkey() != 0 {
				h()
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()
	return true
}

func (s *Service) Stop() {
	running.Store(false)
	C.stopEventTap()
}

// ─── Snippet Auto-Expand ─────────────────────────────────────────────────

// SnippetLookupFn looks up a snippet prefix and returns the content if found.
type SnippetLookupFn func(prefix string) *string

// StartSnippetExpand starts a goroutine that monitors typed characters
// and expands snippet prefixes.
func (s *Service) StartSnippetExpand(lookup SnippetLookupFn, getPrefixes func() []string) {
	prefixes := getPrefixes()
	log.Printf("snippet: auto-expand active with %d prefix(es): %s", len(prefixes), strings.Join(prefixes, ", "))
	go func() {
		var lastBuf string
		for running.Load() {
			buf := C.GoString(C.getSnippetBuffer())

			if buf != lastBuf && len(buf) > 0 {
				lastBuf = buf
				prefixes := getPrefixes()
				sort.Slice(prefixes, func(i, j int) bool {
					return utf8.RuneCountInString(prefixes[i]) > utf8.RuneCountInString(prefixes[j])
				})
				if len(prefixes) > 0 && strings.Contains(buf, ";") {
					log.Printf("snippet: buffer=%q", buf)
				}
				if match, ok := matchSnippetTrigger(buf, prefixes); ok {
					content := lookup(match.prefix)
					if content != nil {
						log.Printf("snippet: MATCH! prefix=%q trigger=%q replacement=%d chars", match.prefix, match.deleteText, utf8.RuneCountInString(*content))
						s.expandSnippet(match.deleteText, *content+match.keepText)
						C.clearSnippetBuffer()
						lastBuf = ""
					}
				}
			}
			time.Sleep(50 * time.Millisecond)
		}
		log.Println("snippet: goroutine exited (running=false)")
	}()
}

type snippetTriggerMatch struct {
	prefix     string
	deleteText string
	keepText   string
}

func matchSnippetTrigger(buf string, prefixes []string) (snippetTriggerMatch, bool) {
	for _, prefix := range prefixes {
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

func (s *Service) expandSnippet(trigger string, content string) {
	// 1. Save current clipboard
	oldClip := C.getClipboardText()
	var oldClipStr string
	if oldClip != nil {
		oldClipStr = C.GoString(oldClip)
		C.free(unsafe.Pointer(oldClip))
	}

	// 2. Send backspaces to delete the prefix
	C.sendBackspaces(C.int(utf8.RuneCountInString(trigger)))
	time.Sleep(50 * time.Millisecond)

	// 3. Set clipboard to snippet content
	cContent := C.CString(content)
	C.setClipboardText(cContent)
	C.free(unsafe.Pointer(cContent))
	time.Sleep(30 * time.Millisecond)

	// 4. Paste (Cmd+V)
	C.pasteClipboard()
	time.Sleep(100 * time.Millisecond)

	// 5. Restore original clipboard
	if oldClipStr != "" {
		cOld := C.CString(oldClipStr)
		C.setClipboardText(cOld)
		C.free(unsafe.Pointer(cOld))
	}
}
