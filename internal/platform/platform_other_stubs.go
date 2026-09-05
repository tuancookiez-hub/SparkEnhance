//go:build !windows

package platform

// Non-Windows stubs. macOS and Linux rely on:
//   - CSS border-radius for visual rounding
//   - Wails built-in clipboard APIs (not yet wired through here)
//
// Returning zeroed/empty values is the right default — the bar still
// shows, the hotkey still fires, the clipboard still works in
// applications that aren't the SparkEnhance app, just not yet via
// our helpers.

func cursorPosImpl() (int, int) { return 0, 0 }
func workAreaAtImpl(x, y int) (Monitor, error) {
	return Monitor{X: 0, Y: 0, Width: 1920, Height: 1080}, nil
}
func monitorAtImpl(x, y int) (Monitor, error) {
	return Monitor{X: 0, Y: 0, Width: 1920, Height: 1080}, nil
}
func writeClipboardImpl(text string) error { return nil }
func readClipboardImpl() (string, error) { return "", nil }
func setWindowRoundedImpl(hwnd uintptr, radius int) error { return nil }
func windowBoundsImpl(hwnd uintptr) (int, int, int, int, bool) {
	return 0, 0, 0, 0, false
}
