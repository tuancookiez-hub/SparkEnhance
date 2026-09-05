//go:build windows

package platform

// On Windows, the actual implementations live in platform_windows.go
// as `windowsXxx`. These dispatcher functions bridge the shared
// `*Impl` suffix names called from platform.go to the windows-prefixed
// implementations. The suffix dispatch pattern keeps platform.go
// build-tag-free.

func cursorPosImpl() (int, int) { return windowsCursorPos() }
func workAreaAtImpl(x, y int) (Monitor, error) { return windowsWorkAreaAt(x, y) }
func monitorAtImpl(x, y int) (Monitor, error) { return windowsMonitorAt(x, y) }
func writeClipboardImpl(text string) error { return windowsWriteClipboard(text) }
func readClipboardImpl() (string, error) { return windowsReadClipboard() }
func setWindowRoundedImpl(hwnd uintptr, radius int) error { return windowsSetWindowRounded(hwnd, radius) }
func windowBoundsImpl(hwnd uintptr) (int, int, int, int, bool) { return windowsWindowBounds(hwnd) }
