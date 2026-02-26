package commands

import "github.com/dyike/Void/internal/model"

// ─── System Commands ───────────────────────────────────────────────────────

func systemCommands() []model.CommandInfo {
	return []model.CommandInfo{
		// ── Void built-ins ──────────────────────────
		{
			ID: "system-clipboard-manager", Title: "Clipboard History",
			Subtitle: "Browse your clipboard history", Icon: "Clipboard", Category: "system",
		},
		{
			ID: "system-ai-chat", Title: "AI Chat",
			Subtitle: "Chat with AI assistant", Icon: "Sparkles", Category: "system",
		},
		{
			ID: "system-snippets", Title: "Snippets",
			Subtitle: "Manage text snippets", Icon: "FileText", Category: "system",
		},
		// ── macOS system actions ───────────────────
		{
			ID: "system-lock-screen", Title: "Lock Screen",
			Subtitle: "Turn off displays and lock", Icon: "Lock", Category: "system",
		},
		{
			ID: "system-sleep", Title: "Sleep",
			Subtitle: "Put Mac to sleep", Icon: "Moon", Category: "system",
		},
		{
			ID: "system-empty-trash", Title: "Empty Trash",
			Subtitle: "Permanently delete trashed files", Icon: "Trash", Category: "system",
		},
		{
			ID: "system-restart", Title: "Restart",
			Subtitle: "Restart your Mac", Icon: "RotateCw", Category: "system",
		},
		{
			ID: "system-shutdown", Title: "Shut Down",
			Subtitle: "Shut down your Mac", Icon: "Power", Category: "system",
		},
		{
			ID: "system-open-finder", Title: "Open Finder",
			Subtitle: "Open Finder", Icon: "Folder", Category: "system",
		},
		{
			ID: "system-open-terminal", Title: "Open Terminal",
			Subtitle: "Open Terminal", Icon: "Terminal", Category: "system",
		},
		// ── Window management ──────────────────────
		{
			ID: "window-left-half", Title: "Left Half",
			Subtitle: "Move window to left half", Icon: "ArrowLeft", Category: "system",
		},
		{
			ID: "window-right-half", Title: "Right Half",
			Subtitle: "Move window to right half", Icon: "ArrowRight", Category: "system",
		},
		{
			ID: "window-top-half", Title: "Top Half",
			Subtitle: "Move window to top half", Icon: "ArrowUp", Category: "system",
		},
		{
			ID: "window-bottom-half", Title: "Bottom Half",
			Subtitle: "Move window to bottom half", Icon: "ArrowDown", Category: "system",
		},
		{
			ID: "window-top-left", Title: "Top Left Quarter",
			Subtitle: "Move window to top-left quarter", Icon: "ArrowUpLeft", Category: "system",
		},
		{
			ID: "window-top-right", Title: "Top Right Quarter",
			Subtitle: "Move window to top-right quarter", Icon: "ArrowUpRight", Category: "system",
		},
		{
			ID: "window-bottom-left", Title: "Bottom Left Quarter",
			Subtitle: "Move window to bottom-left quarter", Icon: "ArrowDownLeft", Category: "system",
		},
		{
			ID: "window-bottom-right", Title: "Bottom Right Quarter",
			Subtitle: "Move window to bottom-right quarter", Icon: "ArrowDownRight", Category: "system",
		},
		{
			ID: "window-maximize", Title: "Maximize",
			Subtitle: "Maximize window to fill screen", Icon: "Maximize", Category: "system",
		},
		{
			ID: "window-center", Title: "Center Window",
			Subtitle: "Center window on screen", Icon: "Target", Category: "system",
		},
	}
}
