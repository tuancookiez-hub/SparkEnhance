//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

// Lazy imports for Windows native APIs.
var (
	user32          = syscall.NewLazyDLL("user32.dll")
	gdi32           = syscall.NewLazyDLL("gdi32.dll")
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procGetCursorPos = user32.NewProc("GetCursorPos")
	procMonitorFromPoint = user32.NewProc("MonitorFromPoint")
	procGetMonitorInfo = user32.NewProc("GetMonitorInfoW")
	procGetWindowRect = user32.NewProc("GetWindowRect")
	procCreateRoundRect = gdi32.NewProc("CreateRoundRectRgn")
	procSetWindowRgn = user32.NewProc("SetWindowRgn")
	procOpenClipboard = user32.NewProc("OpenClipboard")
	procCloseClipboard = user32.NewProc("CloseClipboard")
	procEmptyClipboard = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procGlobalAlloc = kernel32.NewProc("GlobalAlloc")
	procGlobalLock = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
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
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

// ─── Implementations (called from platform.go via the `Impl` suffix) ───────

func windowsCursorPos() (int, int) {
	type pt struct{ X, Y int32 }
	var p pt
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p)))
	if r == 0 {
		return 0, 0
	}
	return int(p.X), int(p.Y)
}

func windowsMonitorAt(px, py int) (Monitor, error) {
	hMon, _, _ := procMonitorFromPoint.Call(uintptr(px), uintptr(py))
	if hMon == 0 {
		return Monitor{X: 0, Y: 0, Width: 1920, Height: 1080}, nil
	}
	var mi monitorInfoEx
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	_, _, _ = procGetMonitorInfo.Call(hMon, uintptr(unsafe.Pointer(&mi)))
	return Monitor{
		X:      int(mi.RcMonitor.Left),
		Y:      int(mi.RcMonitor.Top),
		Width:  int(mi.RcMonitor.Right - mi.RcMonitor.Left),
		Height: int(mi.RcMonitor.Bottom - mi.RcMonitor.Top),
	}, nil
}

func windowsWorkAreaAt(x, y int) (Monitor, error) {
	mon, err := monitorAtImpl(x, y)
	if err != nil {
		return mon, err
	}
	hMon, _, _ := procMonitorFromPoint.Call(uintptr(x), uintptr(y))
	if hMon == 0 {
		return mon, nil
	}
	var mi monitorInfoEx
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	if _, _, _ = procGetMonitorInfo.Call(hMon, uintptr(unsafe.Pointer(&mi))); mi.CbSize == 0 {
		return mon, nil
	}
	return Monitor{
		X:      int(mi.RcWork.Left),
		Y:      int(mi.RcWork.Top),
		Width:  int(mi.RcWork.Right - mi.RcWork.Left),
		Height: int(mi.RcWork.Bottom - mi.RcWork.Top),
	}, nil
}

func windowsSetWindowRounded(hwnd uintptr, radius int) error {
	if hwnd == 0 || radius < 1 {
		return nil
	}
	var r winRect
	if _, _, _ = procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); r.Right == 0 && r.Bottom == 0 {
		return nil
	}
	w := int(r.Right - r.Left)
	h := int(r.Bottom - r.Top)
	hRgn, _, _ := procCreateRoundRect.Call(0, 0, uintptr(w), uintptr(h),
		uintptr(radius*2), uintptr(radius*2))
	if hRgn == 0 {
		return nil
	}
	// bRedraw=1; OS owns the region after this call (do NOT DeleteObject)
	_, _, _ = procSetWindowRgn.Call(hwnd, hRgn, 1)
	return nil
}

func windowsWindowBounds(hwnd uintptr) (int, int, int, int, bool) {
	if hwnd == 0 {
		return 0, 0, 0, 0, false
	}
	var r winRect
	if _, _, _ = procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); r.Right == 0 && r.Bottom == 0 {
		return 0, 0, 0, 0, false
	}
	return int(r.Left), int(r.Top), int(r.Right - r.Left), int(r.Bottom - r.Top), true
}

func windowsWriteClipboard(text string) error {
	utf16, _ := syscall.UTF16FromString(text)
	bufLen := len(utf16) * 2

	if r, _, _ := procOpenClipboard.Call(0); r == 0 {
		return nil
	}
	defer procCloseClipboard.Call()

	procEmptyClipboard.Call()
	hMem, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(bufLen+2))
	if hMem == 0 {
		return nil
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
	return nil
}

func windowsReadClipboard() (string, error) {
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

	for i := 0; ; i++ {
		if i > 524288 { // 1MB safety limit
			return "", nil
		}
		c := *(*uint16)(unsafe.Pointer(ptr + uintptr(i*2)))
		if c == 0 {
			buf := make([]uint16, i)
			for j := 0; j < i; j++ {
				buf[j] = *(*uint16)(unsafe.Pointer(ptr + uintptr(j*2)))
			}
			return syscall.UTF16ToString(buf), nil
		}
	}
}
