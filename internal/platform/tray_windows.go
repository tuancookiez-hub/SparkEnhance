//go:build windows

package platform

import (
	"context"
	"log"
	"os/exec"
	"runtime"
	"syscall"
	"unsafe"
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")
	procCreateMenu   = user32.NewProc("CreatePopupMenu")
	procAppendMenu   = user32.NewProc("AppendMenuW")
	procTrackMenu    = user32.NewProc("TrackPopupMenu")
	procGetCursorPos = user32.NewProc("GetCursorPos")
	procSetForeground = user32.NewProc("SetForegroundWindow")
	procDestroyMenu  = user32.NewProc("DestroyMenu")
	procCreateWindow = user32.NewProc("CreateWindowExW")
	procRegisterClass = user32.NewProc("RegisterClassExW")
	procDefWindowProc = user32.NewProc("DefWindowProcW")
	procLoadIconW    = user32.NewProc("LoadIconW")
)

// startSystray creates a hidden message window, registers Shell_NotifyIcon,
// pumps messages, and dispatches menu actions to the callbacks in
// platform.go. Exits when ctx is cancelled.
func startSystray(ctx context.Context) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInst, _, _ := procGetModuleHandleW.Call(0)
	className := mustUTF16("SparkEnhanceTrayWnd")
	windowName := mustUTF16("SparkEnhanceTray")
	cb := syscall.NewCallback(trayWndProc)

	cls := wndclassex{
		CbSize:        uint32(unsafe.Sizeof(wndclassex{})),
		LpfnWndProc:   cb,
		HInstance:     hInst,
		LpszClassName: className,
	}
	_, _, _ = procRegisterClass.Call(uintptr(unsafe.Pointer(&cls)))

	hwnd, _, _ := procCreateWindow.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0, 0, 0, 0, 0,
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		log.Printf("tray: CreateWindow failed")
		return
	}

	nid := buildNotifyIcon(hwnd, hInst)
	if ok, _, _ := procShellNotifyIcon.Call(0 /* NIM_ADD */, uintptr(unsafe.Pointer(&nid))); ok == 0 {
		log.Printf("tray: Shell_NotifyIcon failed")
		return
	}

	// Pump messages until cancelled.
	msgCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		msgCh <- struct{}{}
	}()

	for {
		var m msg
		r, _, _ := procGetMessageW.Call(
			uintptr(unsafe.Pointer(&m)), 0, 0, 0,
		)
		if r == 0 {
			return
		}
		// Non-blocking check for cancel.
		select {
		case <-msgCh:
			procShellNotifyIcon.Call(2 /* NIM_DELETE */, uintptr(unsafe.Pointer(&nid)))
			return
		default:
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func trayWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
	const (
		WM_USER_TRAY   = 0x0400
		WM_RBUTTONUP   = 0x0205
		WM_LBUTTONUP   = 0x0202
		WM_COMMAND     = 0x0111
		IDM_ENHANCE    = 1001
		IDM_SETTINGS   = 1002
		IDM_QUIT       = 1003
	)
	switch uint32(msg) {
	case WM_USER_TRAY:
		switch uint32(lParam) {
		case WM_RBUTTONUP:
			showTrayMenu(hwnd)
		case WM_LBUTTONUP:
			if cbEnhance != nil {
				go cbEnhance()
			}
		}
	case WM_COMMAND:
		id := uint32(wParam & 0xFFFF)
		switch id {
		case IDM_ENHANCE:
			if cbEnhance != nil {
				go cbEnhance()
			}
		case IDM_SETTINGS:
			if cbSettings != nil {
				go cbSettings()
			}
		case IDM_QUIT:
			if cbQuit != nil {
				go cbQuit()
			}
		}
	}
	return 0
}

func showTrayMenu(hwnd uintptr) {
	hMenu, _, _ := procCreateMenu.Call(0)
	defer procDestroyMenu.Call(hMenu)

	append := func(id uint32, label string) {
		ptr, _ := syscall.UTF16PtrFromString(label)
		procAppendMenu.Call(hMenu, 0, uintptr(id), uintptr(unsafe.Pointer(ptr)))
	}
	append(1001, "Enhance (Ctrl+Shift+E)")
	append(1002, "Settings...")
	procAppendMenu.Call(hMenu, 0x0800, 0, 0) // separator
	append(1003, "Quit")

	var pt struct{ X, Y int32 }
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForeground.Call(hwnd)

	const TPM_RETURNCMD = 0x0100
	chosen, _, _ := procTrackMenu.Call(
		hMenu, TPM_RETURNCMD,
		uintptr(uint32(pt.X)), uintptr(uint32(pt.Y)),
		0, hwnd, 0,
	)
	if chosen != 0 {
		const WM_COMMAND = 0x0111
		procPostMessageW.Call(hwnd, WM_COMMAND, chosen, 0)
	}
}

var procPostMessageW = user32.NewProc("PostMessageW")
var procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
	LPriv   uint32
}

type wndclassex struct {
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

type notifyIconDataW struct {
	CbSize           uint32
	Wnd              uintptr
	ID               uint32
	Flags            uint32
	CallbackMessage  uint32
	Icon             uintptr
	Tip              [128]uint16
	State            uint32
	StateMask        uint32
	Info             [256]uint16
	Timeout          uint32
	Version          uint32
	InfoTitle        [64]uint16
	InfoFlags        uint32
	Guid             [16]byte
	IconBalloon      uintptr
}

func mustUTF16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func buildNotifyIcon(hwnd, hInst uintptr) notifyIconDataW {
	nid := notifyIconDataW{
		CbSize:          uint32(unsafe.Sizeof(notifyIconDataW{})),
		Wnd:             hwnd,
		ID:              1,
		Flags:           0x0001 | 0x0002 | 0x0004, // NIF_MESSAGE | NIF_ICON | NIF_TIP
		CallbackMessage: 0x0400,
		Icon:            loadDefaultIcon(),
	}
	tip, _ := syscall.UTF16PtrFromString("SparkEnhance — Ctrl+Shift+E")
	copyUTF16(nid.Tip[:], tip)
	return nid
}

func copyUTF16(dst []uint16, src *uint16) {
	for i := 0; i < len(dst)-1; i++ {
		v := *(*uint16)(unsafe.Add(unsafe.Pointer(src), uintptr(i)*2))
		if v == 0 {
			return
		}
		dst[i] = v
	}
}

func loadDefaultIcon() uintptr {
	// IDI_APPLICATION = 32512, IMAGE_ICON = 1, LR_DEFAULTSIZE = 0x40
	h, _, _ := procLoadIconW.Call(0, 32512)
	if h != 0 {
		return h
	}
	// Last-resort: load the exe icon
	exe, err := exec.LookPath("notepad.exe")
	if err == nil {
		h, _, _ = procLoadIconW.Call(
			uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(exe))),
			0,
		)
	}
	_ = syscall.StringToUTF16Ptr
	return h
}
