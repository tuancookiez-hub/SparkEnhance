//go:build !windows

package platform

import (
	"context"
	"log"
)

// On non-Windows the platform package is a stub so the project still
// builds for development.

func listenHotkeyImpl(ctx context.Context, _ string, onFire func()) {
	<-ctx.Done()
}

func readClipboardImpl() (string, error)       { return "", nil }
func writeClipboardImpl(text string) error     { return nil }
func getSelectionImpl() (string, error)        { return "", nil }
func simulatePasteImpl(text string) error      { return nil }

func runTrayImpl(ctx context.Context, _, _, _ func()) {
	log.Println("tray: stub (non-Windows build)")
	<-ctx.Done()
}
