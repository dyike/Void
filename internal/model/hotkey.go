package model

// KeySpec describes a global hotkey combination.
type KeySpec struct {
	Key       string `json:"key"`
	Modifiers string `json:"modifiers"`
	Display   string `json:"display"`
}

// DefaultHotkey is the factory default: Option+Space.
var DefaultHotkey = KeySpec{Key: "space", Modifiers: "option", Display: "⌥Space"}
