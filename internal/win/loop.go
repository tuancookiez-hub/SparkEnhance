package win

import (
	"log"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetModuleHandleW   = kernel32.NewProc("GetModuleHandleW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procRegisterHotKey     = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey   = user32.NewProc("UnregisterHotKey")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procGetCursorPos       = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procCreateMutexW       = kernel32.NewProc("CreateMutexW")
	procGetLastError       = kernel32.NewProc("GetLastError")
	procCloseHandle        = kernel32.NewProc("CloseHandle")
	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
)

type point struct {
	X, Y int32
}

type wndclassexw struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type msg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

func mustUTF16(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		log.Panicf("UTF16PtrFromString(%q): %v", s, err)
	}
	return p
}

func currentCursorPos() (x, y int32) {
	var pt point
	r, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if r == 0 {
		return 0, 0
	}
	return pt.X, pt.Y
}

// runMessageLoop pumps Windows messages until WM_QUIT.
func runMessageLoop() {
	for {
		var m msg
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 {
			return
		}
		if r == 0xFFFFFFFF {
			log.Printf("GetMessageW failed")
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// errSingleInstance signals the mutex is already held.
var errSingleInstance = syscall.Errno(0xB7)

func acquireSingleInstance() error {
	const name = "Global\\SparkEnhanceSingleInstance_v1"
	namePtr, _ := syscall.UTF16PtrFromString(name)
	h, _, _ := procCreateMutexW.Call(0, 1, uintptr(unsafe.Pointer(namePtr)))
	if h == 0 {
		return syscall.GetLastError()
	}
	lastErr, _, _ := procGetLastError.Call()
	if lastErr == 0xB7 {
		// already exists — leak handle intentionally
		return errSingleInstance
	}
	return nil
}
