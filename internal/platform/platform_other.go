//go:build !windows

package platform

import (
	"context"
	"log"
)

// On non-Windows the platform package is a stub so the project still
// builds for development. Clipboard uses a temp file marker, hotkey is
// a no-op (since RegisterHotKey is Windows-only), tray is a no-op.

func ListenHotkey(ctx context.Context, _ string, onFire func()) {
	<-ctx.Done()
}

func ReadClipboard() (string, error) {
	return "", nil
}

func WriteClipboard(text string) error { return nil }
func GetSelection() (string, error)      { return "", nil }
func SimulatePaste(text string) error    { return nil }

func RunTray(ctx context.Context, _, _, _ func()) {
	log.Println("tray: stub (non-Windows build)")
	<-ctx.Done()
}
