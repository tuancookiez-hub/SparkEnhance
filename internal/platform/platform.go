// Package platform wraps OS-level operations needed by SparkEnhance.
// No secrets, no network calls — pure platform APIs.
package platform

import (
	"runtime"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ─── Monitor / window geometry ───────────────────────────────────────────────

type Monitor struct {
	Width  int
	Height int
	X      int
	Y      int
}

type Point struct{ X, Y int }

// PrimaryMonitor returns the primary display.
func PrimaryMonitor() (Monitor, error) {
	return primaryMonitorImpl()
}

func MonitorFromPoint(px, py int) (Monitor, error) {
	return monitorFromPointImpl(px, py)
}

// WindowBounds returns (x, y, w, h, ok).
func WindowBounds(hwnd uintptr) (int, int, int, int, bool) {
	return windowBoundsImpl(hwnd)
}

// ─── Rounded corners (Windows only) ──────────────────────────────────────────

// SetRoundedWindowRegion clips the Win32 window to a rounded rectangle.
// Radius is in pixels. Idempotent — safe to call every position update.
func SetRoundedWindowRegion(hwnd uintptr, radius int) {
	if runtime.GOOS != "windows" {
		return
	}
	setRoundedRegionImpl(hwnd, radius)
}

// MoveWindowTopLeft moves and resizes the window in one call.
func MoveWindowTopLeft(hwnd uintptr, x, y, w, h int) {
	moveWindowTopLeftImpl(hwnd, x, y, w, h)
}

// ─── Clipboard (cross-platform) ──────────────────────────────────────────────

// SetClipboard writes text to the system clipboard.
func SetClipboard(text string) {
	setClipboardImpl(text)
}

// ─── Native window handle from Wails window ─────────────────────────────────

// NativeHandle returns the raw OS window handle as uintptr.
// On Windows: HWND cast to uintptr.
// On macOS:   NSWindow * cast to uintptr.
// On Linux:   GtkWindow * cast to uintptr.
func NativeHandle(win application.Window) uintptr {
	if win == nil {
		return 0
	}
	return uintptr(win.NativeWindow())
}

// ─── Platform implementations (stub for non-Windows; real impls in platform_*.go)

func primaryMonitorImpl() (Monitor, error) {
	return Monitor{Width: 1920, Height: 1080, X: 0, Y: 0}, nil
}

func monitorFromPointImpl(px, py int) (Monitor, error) {
	return primaryMonitorImpl()
}

func windowBoundsImpl(hwnd uintptr) (int, int, int, int, bool) {
	return 0, 0, 600, 56, false
}

func setRoundedRegionImpl(hwnd uintptr, radius int) {}

func moveWindowTopLeftImpl(hwnd uintptr, x, y, w, h int) {}

func setClipboardImpl(text string) {}

var _ = unsafe.Pointer // suppress unused import on non-Windows
