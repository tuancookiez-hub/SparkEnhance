package bar

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procKeybd_event      = user32.NewProc("keybd_event")
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

// pasteEnhancedWindows writes text to the clipboard and simulates Ctrl+V,
// then waits briefly and restores prev to the clipboard.
func pasteEnhancedWindows(text, prev string) error {
	if err := writeClipboard(text); err != nil {
		return err
	}
	// Simulate Ctrl+V (VK_CONTROL=0x11, VK_V=0x56, both 0-down)
	_, _, _ = procKeybd_event.Call(0x11, 0, 0, 0) // CTRL down
	time.Sleep(20 * time.Millisecond)
	_, _, _ = procKeybd_event.Call(0x56, 0, 0, 0) // V down
	time.Sleep(20 * time.Millisecond)
	_, _, _ = procKeybd_event.Call(0x56, 0, 2, 0) // V up (KEYEVENTF_KEYUP=2)
	_, _, _ = procKeybd_event.Call(0x11, 0, 2, 0) // CTRL up

	// Wait then restore.
	time.Sleep(500 * time.Millisecond)
	if prev != "" {
		_ = writeClipboard(prev)
	}
	return nil
}

func writeClipboard(text string) error {
	const (
		cfUnicodeText = 13
		gmemMoveable  = 0x0002
	)
	ok, _, _ := procOpenClipboard.Call(0)
	if ok == 0 {
		return syscall.GetLastError()
	}
	defer procCloseClipboard.Call()

	_, _, _ = procEmptyClipboard.Call()

	utf16 := syscall.StringToUTF16(text)
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&utf16[0])), len(utf16)*2)
	h, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(len(bytes)))
	if h == 0 {
		return syscall.GetLastError()
	}
	defer procGlobalFree.Call(h)

	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return syscall.GetLastError()
	}
	defer procGlobalUnlock.Call(h)
	dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(bytes))
	copy(dst, bytes)
	_, _, _ = procSetClipboardData.Call(cfUnicodeText, h)
	return nil
}
