// Package branding contains Void's editable artwork and generated runtime assets.
package branding

import _ "embed"

//go:generate go run ../../scripts/icons -root ../..

// AppIcon is full-bleed artwork; Keel adds the platform's icon mask and shadow.
//
//go:embed app.png
var AppIcon []byte

// TrayIcon is an 18pt @2x template with the app's offset circular opening.
//
//go:embed tray.png
var TrayIcon []byte
