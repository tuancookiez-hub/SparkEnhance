// Package platform contains OS glue for SparkEnhance:
//
//   - ReadClipboard / WriteClipboard / GetSelection / SimulatePaste
//   - ListenHotkey (global Win32 RegisterHotKey in a goroutine)
//   - RunTray (system tray icon + menu)
//
// On non-Windows platforms the same functions are stubbed for development
// convenience (clipboard uses a temp file, hotkey is a no-op, tray is a stub).
package platform

import (
	"context"
	"log"
	"sync"
)

// ListenHotkey registers a global hotkey and calls onFire whenever it is
// pressed. The listener runs in a background goroutine and exits when
// ctx is cancelled.
func ListenHotkey(ctx context.Context, hotkey string, onFire func()) {
	go listenHotkey(ctx, hotkey, onFire)
}

func listenHotkey(ctx context.Context, hotkey string, onFire func()) {
	if hotkey == "" {
		hotkey = "ctrl+shift+e"
	}
	hmod, vk, err := parseHotkey(hotkey)
	if err != nil {
		log.Printf("hotkey parse %q: %v", hotkey, err)
		return
	}
	hk := register(hmod, vk)
	if hk == 0 {
		log.Printf("RegisterHotKey failed for %s", hotkey)
		return
	}
	defer unregister(hk)

	msgCh := make(chan struct{}, 1)
	go pumpHotkeyMessages(hk, msgCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-msgCh:
			if onFire != nil {
				go onFire()
			}
		}
	}
}

// SetSelection reads the user's current text selection by sending Ctrl+C
// and reading the clipboard. Restores the previous clipboard contents.
func GetSelection() (string, error) {
	prev, _ := ReadClipboard()
	if err := simulateCopy(); err != nil {
		return "", err
	}
	sel, err := ReadClipboard()
	if err != nil || sel == "" {
		if prev != "" {
			_ = WriteClipboard(prev)
		}
		return "", err
	}
	if prev != "" && prev != sel {
		_ = WriteClipboard(prev)
	}
	return sel, nil
}

// SimulatePaste writes text to the clipboard and sends Ctrl+V.
func SimulatePaste(text string) error {
	if err := WriteClipboard(text); err != nil {
		return err
	}
	return simulatePaste()
}

// RunTray starts a system tray icon with three items: Enhance, Settings, Quit.
func RunTray(ctx context.Context, onEnhance, onSettings, onQuit func()) {
	runTray(ctx, onEnhance, onSettings, onQuit)
}

var (
	cbEnhance   func()
	cbSettings  func()
	cbQuit      func()
	trayOnce    sync.Once
)
