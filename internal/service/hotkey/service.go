//go:build darwin

package hotkey

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices

#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <ApplicationServices/ApplicationServices.h>

// ── Configurable hotkey ───────────────────────────────────────────────────

// Current hotkey configuration (defaults to Option+Space)
static CGKeyCode  g_keyCode   = 49;                          // Space
static int        g_modifiers = kCGEventFlagMaskAlternate;   // Option

void setHotkeyConfig(int keyCode, int modifiers) {
    g_keyCode   = (CGKeyCode)keyCode;
    g_modifiers = modifiers;
}

// ── Event tap ─────────────────────────────────────────────────────────────

static volatile int hotkey_fired = 0;

static CGEventRef eventTapCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *userInfo) {
    if (type == kCGEventKeyDown) {
        CGKeyCode keyCode = (CGKeyCode)CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
        CGEventFlags flags = CGEventGetFlags(event);
        int masked = (int)(flags & (kCGEventFlagMaskAlternate | kCGEventFlagMaskCommand |
                                    kCGEventFlagMaskControl   | kCGEventFlagMaskShift));
        if (keyCode == g_keyCode && masked == g_modifiers) {
            hotkey_fired = 1;
            return NULL; // consume event
        }
    }
    return event;
}

static CFMachPortRef gTap = NULL;
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
    if (gTap) {
        CGEventTapEnable(gTap, false);
        CFRelease(gTap);
        gTap = NULL;
    }
    if (gRunLoopSrc) {
        CFRelease(gRunLoopSrc);
        gRunLoopSrc = NULL;
    }
}

static int pollHotkey(void) {
    if (hotkey_fired) { hotkey_fired = 0; return 1; }
    return 0;
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
*/
import "C"
import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"
)

// KeySpec describes a global hotkey combination.
type KeySpec struct {
	Key       string `json:"key"`       // e.g. "space", "a", "f1"
	Modifiers string `json:"modifiers"` // comma-separated: "option", "cmd", "ctrl", "shift"
	// Human-readable display string e.g. "⌥Space"
	Display string `json:"display"`
}

// DefaultHotkey is the factory default: Option+Space.
var DefaultHotkey = KeySpec{Key: "space", Modifiers: "option", Display: "⌥Space"}

// ─── Key code table ──────────────────────────────────────────────────────

// keyCodes maps key name → macOS virtual keyCode.
// Source: <Carbon/Events.h> kVK_* constants.
var keyCodes = map[string]int{
	// Special keys
	"space":     49,
	"return":    36,
	"enter":     36,
	"tab":       48,
	"delete":    51,
	"backspace": 51,
	"escape":    53,
	"esc":       53,
	"capslock":  57,
	"home":      115,
	"end":       119,
	"pageup":    116,
	"pagedown":  121,
	"left":      123,
	"right":     124,
	"down":      125,
	"up":        126,
	// Function keys
	"f1": 122, "f2": 120, "f3": 99, "f4": 118,
	"f5": 96, "f6": 97, "f7": 98, "f8": 100,
	"f9": 101, "f10": 109, "f11": 103, "f12": 111,
	// Letters (US QWERTY)
	"a": 0, "b": 11, "c": 8, "d": 2, "e": 14,
	"f": 3, "g": 5, "h": 4, "i": 34, "j": 38,
	"k": 40, "l": 37, "m": 46, "n": 45, "o": 31,
	"p": 35, "q": 12, "r": 15, "s": 1, "t": 17,
	"u": 32, "v": 9, "w": 13, "x": 7, "y": 16,
	"z": 6,
	// Numbers
	"0": 29, "1": 18, "2": 19, "3": 20,
	"4": 21, "5": 23, "6": 22, "7": 26,
	"8": 28, "9": 25,
	// Symbols
	"`": 50, "-": 27, "=": 24, "[": 33, "]": 30,
	"\\": 42, ";": 41, "'": 39, ",": 43, ".": 47, "/": 44,
}

// modifierFlags maps modifier name → CGEventFlags mask value.
var modifierFlags = map[string]int{
	"option":  0x00080000, // kCGEventFlagMaskAlternate
	"alt":     0x00080000,
	"cmd":     0x00100000, // kCGEventFlagMaskCommand
	"command": 0x00100000,
	"ctrl":    0x00040000, // kCGEventFlagMaskControl
	"control": 0x00040000,
	"shift":   0x00020000, // kCGEventFlagMaskShift
}

// modifierSymbols maps modifier name → display symbol.
var modifierSymbols = map[string]string{
	"option": "⌥", "alt": "⌥",
	"cmd": "⌘", "command": "⌘",
	"ctrl": "⌃", "control": "⌃",
	"shift": "⇧",
}

// keyDisplayNames maps key name → display string.
var keyDisplayNames = map[string]string{
	"space": "Space", "return": "↩", "enter": "↩",
	"tab": "⇥", "delete": "⌫", "backspace": "⌫",
	"escape": "⎋", "esc": "⎋",
	"up": "↑", "down": "↓", "left": "←", "right": "→",
	"pageup": "⇞", "pagedown": "⇟", "home": "↖", "end": "↘",
}

// ─── Parsing ──────────────────────────────────────────────────────────────

// ParseHotkey parses a hotkey string like "option+space" or "cmd+shift+a".
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

	if key == "" {
		return nil, fmt.Errorf("no key specified")
	}
	if _, ok := keyCodes[key]; !ok {
		return nil, fmt.Errorf("unknown key: %q", key)
	}

	// Build display string
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

	return &KeySpec{
		Key:       key,
		Modifiers: strings.Join(mods, ","),
		Display:   display,
	}, nil
}

// FormatSpec returns the "mod+mod+key" string for a KeySpec.
func FormatSpec(ks KeySpec) string {
	if ks.Modifiers == "" {
		return ks.Key
	}
	return strings.ReplaceAll(ks.Modifiers, ",", "+") + "+" + ks.Key
}

// ─── Service ─────────────────────────────────────────────────────────────

var running atomic.Bool

// Service manages the global CGEventTap hotkey.
type Service struct{}

// NewService creates a new hotkey service.
func NewService() *Service { return &Service{} }

// HasPermission reports whether Accessibility is granted.
func HasPermission() bool { return C.checkAccessibility(0) != 0 }

// RequestPermission opens the Accessibility permission dialog.
func RequestPermission() { C.checkAccessibility(1) }

// Start registers a global hotkey using CGEventTap.
// keySpec is a string like "option+space" or "cmd+a". Falls back to DefaultHotkey.
// Returns true if the tap was successfully installed.
func (s *Service) Start(h func(), keySpec string) bool {
	if !s.configureKey(keySpec) {
		return false
	}
	return s.startTap(h)
}

// Restart stops the current tap and starts a new one with the given keySpec.
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

// Stop removes the event tap.
func (s *Service) Stop() {
	running.Store(false)
	C.stopEventTap()
}
