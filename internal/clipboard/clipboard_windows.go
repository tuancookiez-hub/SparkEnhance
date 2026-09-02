package clipboard

import (
	"errors"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	openClipboard    = user32.NewProc("OpenClipboard")
	closeClipboard   = user32.NewProc("CloseClipboard")
	emptyClipboard   = user32.NewProc("EmptyClipboard")
	getClipboardData = user32.NewProc("GetClipboardData")
	setClipboardData = user32.NewProc("SetClipboardData")
	globalAlloc      = kernel32.NewProc("GlobalAlloc")
	globalFree       = kernel32.NewProc("GlobalFree")
	globalLock       = kernel32.NewProc("GlobalLock")
	globalUnlock     = kernel32.NewProc("GlobalUnlock")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

// ErrEmpty is returned when the clipboard holds no text.
var ErrEmpty = errors.New("clipboard is empty or contains no text")

// Read returns the current Unicode clipboard text.
func Read() (string, error) {
	ok, _, _ := openClipboard.Call(0)
	if ok == 0 {
		return "", errors.New("OpenClipboard failed")
	}
	defer closeClipboard.Call()

	h, _, _ := getClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", ErrEmpty
	}

	ptr, _, _ := globalLock.Call(h)
	if ptr == 0 {
		return "", errors.New("GlobalLock failed")
	}
	defer globalUnlock.Call(h)

	// Walk the null-terminated wide string.
	ws := (*[1 << 20]uint16)(unsafe.Pointer(ptr))
	var n int
	for n = 0; ws[n] != 0; n++ {
	}
	return syscall.UTF16ToString(ws[:n]), nil
}

// Write sets the clipboard to text. The previous clipboard contents are lost.
func Write(text string) error {
	ok, _, _ := openClipboard.Call(0)
	if ok == 0 {
		return errors.New("OpenClipboard failed")
	}
	defer closeClipboard.Call()

	emptyClipboard.Call()

	utf16 := syscall.StringToUTF16(text)
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&utf16[0])), len(utf16)*2)
	size := uintptr(len(bytes))

	h, _, _ := globalAlloc.Call(gmemMoveable, size)
	if h == 0 {
		return errors.New("GlobalAlloc failed")
	}
	defer globalFree.Call(h)

	ptr, _, _ := globalLock.Call(h)
	if ptr == 0 {
		return errors.New("GlobalLock failed")
	}
	defer globalUnlock.Call(h)

	dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size)
	copy(dst, bytes)

	if _, _, _ = setClipboardData.Call(cfUnicodeText, h); /* best effort */ false {
	}
	return nil
}
