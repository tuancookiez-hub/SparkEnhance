// Package platform wraps OS-level operations SparkEnhance needs.
// No secrets, no network calls — pure platform APIs.
//
// Cross-platform:
//   - CursorPos / WorkAreaAt / MonitorAt  — read monitor geometry
//   - SetWindowRounded                    — round the bar's OS window corners
//   - WriteClipboard / CaptureSelection  — clipboard read/write
//
// Windows-only: SetWindowRounded uses SetWindowRgn + CreateRoundRectRgn
// (since CSS border-radius alone doesn't clip the OS window on Win10/11).
// macOS / Linux: no-op — CSS border-radius handles the rounding via
// WebView2 / WebKit2GTK.
package platform

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ─── Public types ────────────────────────────────────────────────────────────

// Monitor is a logical display (not necessarily physical).
type Monitor struct {
	X      int
	Y      int
	Width  int
	Height int
}

// ─── Public API (cross-platform) ─────────────────────────────────────────────

// CursorPos returns the current cursor position in screen coords.
func CursorPos() (int, int) {
	return cursorPosImpl()
}

// WorkAreaAt returns the work area of the monitor containing (x, y).
// "Work area" excludes the taskbar / dock — the area a floating bar
// can safely use.
func WorkAreaAt(x, y int) (Monitor, error) {
	return workAreaAtImpl(x, y)
}

// MonitorAt returns the monitor containing (x, y).
func MonitorAt(x, y int) (Monitor, error) {
	return monitorAtImpl(x, y)
}

// SetWindowRounded clips the OS window to a rounded rectangle. On Windows
// this calls SetWindowRgn + CreateRoundRectRgn. On macOS / Linux the
// WebView2 / WebKit2GTK CSS border-radius handles the visual rounding,
// so this is a no-op.
func SetWindowRounded(win application.Window) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	hwnd := uintptr(win.NativeWindow())
	if hwnd == 0 {
		return nil
	}
	return setWindowRoundedImpl(hwnd, 15)
}

// WriteClipboard writes text to the system clipboard.
func WriteClipboard(text string) error {
	return writeClipboardImpl(text)
}

// CaptureSelection returns the current selection (assumed to be on
// the clipboard after the frontend triggers Ctrl+C).
func CaptureSelection() (string, error) {
	return readClipboardImpl()
}

// WindowBounds returns the current screen-space bounds of the bar's
// native window: (x, y, w, h, ok).
func WindowBounds(hwnd uintptr) (int, int, int, int, bool) {
	return windowBoundsImpl(hwnd)
}
