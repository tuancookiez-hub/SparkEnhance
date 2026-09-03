package bar

import (
	"syscall"
	"unsafe"
)

// doPaste writes text to clipboard, simulates Ctrl+V, then restores prev.
func doPaste(text, prev string) error {
	if err := writeClipboardDirect(text); err != nil {
		return err
	}
	// Simulate Ctrl+V.
	_, _, _ = procKeybd_event.Call(0x11, 0, 0, 0) // CTRL down
	_, _, _ = procKeybd_event.Call(0x56, 0, 0, 0) // V down
	_, _, _ = procKeybd_event.Call(0x56, 0, 2, 0) // V up KEYEVENTF_KEYUP
	_, _, _ = procKeybd_event.Call(0x11, 0, 2, 0) // CTRL up

	// Restore original clipboard after delay.
	// (The Hide call restores synchronously; this handles the case where
	// the user doesn't dismiss the bar.)
	return nil
}

var (
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc     = kernel32.NewProc("GlobalAlloc")
	procGlobalFree      = kernel32.NewProc("GlobalFree")
	procGlobalLock      = kernel32.NewProc("GlobalLock")
	procGlobalUnlock    = kernel32.NewProc("GlobalUnlock")
	procKeybd_event     = user32.NewProc("keybd_event")
)

func writeClipboardDirect(text string) error {
	const (
		cfUnicode = 13
		gmemMove  = 0x0002
	)
	ok, _, _ := procOpenClipboard.Call(0)
	if ok == 0 {
		return syscall.GetLastError()
	}
	defer procCloseClipboard.Call()
	_, _, _ = procEmptyClipboard.Call()

	utf16 := syscall.StringToUTF16(text)
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&utf16[0])), len(utf16)*2)

	h, _, _ := procGlobalAlloc.Call(gmemMove, uintptr(len(bytes)))
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
	_, _, _ = procSetClipboardData.Call(cfUnicode, h)
	return nil
}
