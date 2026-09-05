//go:build windows

package platform

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	user32              = windows.NewLazySystemDLL("user32.dll")
	gdi32               = windows.NewLazySystemDLL("gdi32.dll")
	kernel32            = windows.NewLazySystemDLL("kernel32.dll")
	procGetWindowRect   = user32.NewProc("GetWindowRect")
	procGetSystemMetrics= user32.NewProc("GetSystemMetrics")
	procMonitorFromPoint= user32.NewProc("MonitorFromPoint")
	procGetMonitorInfo  = user32.NewProc("GetMonitorInfoW")
	procCreateRoundRect = gdi32.NewProc("CreateRoundRectRgn")
	procSetWindowRgn    = user32.NewProc("SetWindowRgn")
	procSetWindowPos    = user32.NewProc("SetWindowPos")
	procOpenClipboard   = user32.NewProc("OpenClipboard")
	procCloseClipboard  = user32.NewProc("CloseClipboard")
	procGetClipboardData= user32.NewProc("GetClipboardData")
	procEmptyClipboard  = user32.NewProc("EmptyClipboard")
	procSetClipboardData= user32.NewProc("SetClipboardData")
	procGlobalAlloc    = kernel32.NewProc("GlobalAlloc")
	procGlobalLock     = kernel32.NewProc("GlobalLock")
	procGlobalUnlock   = kernel32.NewProc("GlobalUnlock")
	procSendInput      = user32.NewProc("SendInputW")
)

type winRect struct{ Left, Top, Right, Bottom int32 }

type monitorInfoEx struct {
	CbSize    uint32
	RcMonitor winRect
	RcWork    winRect
	DwFlags   uint32
	SzDevice  [32]uint16
}

const (
	cfUnicodeText    = 13
	gmemMoveable    = 0x0002
	hwndTop         = 0
	swpNoActivate   = 0x0010
	swpShowWindow   = 0x0040
	inputKeyboard   = 1
	keyeventfKeyup  = 0x0002
)

func primaryMonitorImpl() (Monitor, error) {
	w, _, _ := procGetSystemMetrics.Call(0)
	h, _, _ := procGetSystemMetrics.Call(1)
	return Monitor{Width: int(w), Height: int(h), X: 0, Y: 0}, nil
}

func monitorFromPointImpl(px, py int) (Monitor, error) {
	hMon, _, _ := procMonitorFromPoint.Call(uintptr(px), uintptr(py))
	if hMon == 0 {
		return primaryMonitorImpl()
	}
	var mi monitorInfoEx
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfo.Call(hMon, uintptr(unsafe.Pointer(&mi)))
	return Monitor{
		X:      int(mi.RcMonitor.Left),
		Y:      int(mi.RcMonitor.Top),
		Width:  int(mi.RcMonitor.Right - mi.RcMonitor.Left),
		Height: int(mi.RcMonitor.Bottom - mi.RcMonitor.Top),
	}, nil
}

func windowBoundsImpl(hwnd uintptr) (int, int, int, int, bool) {
	if hwnd == 0 {
		return 0, 0, 0, 0, false
	}
	var r winRect
	if _, _, _ = procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); r.Right == 0 && r.Bottom == 0 {
		return 0, 0, 0, 0, false
	}
	return int(r.Left), int(r.Top), int(r.Right - r.Left), int(r.Bottom - r.Top), true
}

func setRoundedRegionImpl(hwnd uintptr, radius int) {
	if hwnd == 0 || radius < 1 {
		return
	}
	_, _, w, h, ok := windowBoundsImpl(hwnd)
	if !ok {
		return
	}
	hRgn, _, _ := procCreateRoundRect.Call(0, 0, uintptr(w), uintptr(h), uintptr(radius*2), uintptr(radius*2))
	if hRgn == 0 {
		return
	}
	procSetWindowRgn.Call(hwnd, hRgn, 1) // OS owns region after this call
}

func moveWindowTopLeftImpl(hwnd uintptr, x, y, w, h int) {
	if hwnd == 0 {
		return
	}
	procSetWindowPos.Call(hwnd, hwndTop, uintptr(x), uintptr(y), uintptr(w), uintptr(h), swpNoActivate|swpShowWindow)
}

func setClipboardImpl(text string) {
	utf16, _ := syscall.UTF16FromString(text)
	bufLen := len(utf16) * 2

	if r, _, _ := procOpenClipboard.Call(0); r == 0 {
		return
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()
	hMem, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(bufLen+2))
	if hMem == 0 {
		return
	}
	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr != 0 {
		for i, v := range utf16 {
			*(*uint16)(unsafe.Pointer(ptr + uintptr(i*2))) = v
		}
		*(*uint16)(unsafe.Pointer(ptr + uintptr(bufLen))) = 0
		procGlobalUnlock.Call(hMem)
	}
	procSetClipboardData.Call(uintptr(cfUnicodeText), hMem)
}

// CaptureSelection reads the current clipboard text. The frontend triggers Ctrl+C
// before calling this, so the clipboard is pre-populated.
func CaptureSelection() (string, error) {
	if r, _, _ := procOpenClipboard.Call(0); r == 0 {
		return "", nil
	}
	defer procCloseClipboard.Call()

	hMem, _, _ := procGetClipboardData.Call(uintptr(cfUnicodeText))
	if hMem == 0 {
		return "", nil
	}
	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr == 0 {
		return "", nil
	}
	defer procGlobalUnlock.Call(hMem)

	var buf []uint16
	for i := 0; ; i++ {
		if i > 524288 { // 1MB safety limit
			break
		}
		c := *(*uint16)(unsafe.Pointer(ptr + uintptr(i*2)))
		if c == 0 {
			buf = make([]uint16, i)
			for j := 0; j < i; j++ {
				buf[j] = *(*uint16)(unsafe.Pointer(ptr + uintptr(j*2)))
			}
			return syscall.UTF16ToString(buf), nil
		}
	}
	return "", nil
}
