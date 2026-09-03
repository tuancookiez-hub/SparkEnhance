// Package platform contains OS glue for SparkEnhance:
//
//   - ReadClipboard / WriteClipboard / GetSelection / SimulatePaste
//   - ListenHotkey (global WH_KEYBOARD_LL low-level keyboard hook)
//   - RunTray (system tray icon + menu)
//
// On non-Windows platforms the same functions are stubbed for development
// convenience.
package platform

import (
	"context"
)

// ListenHotkey registers a global hotkey. On Windows it installs a
// WH_KEYBOARD_LL low-level keyboard hook that fires from any app
// (including WebView2). Safe to call multiple times — re-registers.
func ListenHotkey(ctx context.Context, hotkey string, onFire func()) {
	listenHotkeyImpl(ctx, hotkey, onFire)
}

// ReadClipboard returns the current clipboard text.
func ReadClipboard() (string, error) {
	return readClipboardImpl()
}

// WriteClipboard sets the clipboard to text.
func WriteClipboard(text string) error {
	return writeClipboardImpl(text)
}

// GetSelection reads the user's current text selection by sending Ctrl+C.
func GetSelection() (string, error) {
	return getSelectionImpl()
}

// SimulatePaste writes text to the clipboard and sends Ctrl+V.
func SimulatePaste(text string) error {
	return simulatePasteImpl(text)
}

// RunTray starts a system tray icon with three items: Enhance, Settings, Quit.
func RunTray(ctx context.Context, onEnhance, onSettings, onQuit func()) {
	runTrayImpl(ctx, onEnhance, onSettings, onQuit)
}
