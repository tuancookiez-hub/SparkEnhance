// Package tray implements a Win32 system tray icon using raw syscalls.
package tray

import (
	"log"
	"syscall"
	"unsafe"
)

var (
	user32  = syscall.NewLazyDLL("user32.dll")
	shell32 = syscall.NewLazyDLL("shell32.dll")

	procCreatePopupMenu      = user32.NewProc("CreatePopupMenu")
	procDestroyMenu          = user32.NewProc("DestroyMenu")
	procAppendMenuW          = user32.NewProc("AppendMenuW")
	procTrackPopupMenu       = user32.NewProc("TrackPopupMenu")
	procGetCursorPos         = user32.NewProc("GetCursorPos")
	procSetForegroundWindow  = user32.NewProc("SetForegroundWindow")
	procPostMessageW         = user32.NewProc("PostMessageW")
	procShell_NotifyIconW    = shell32.NewProc("Shell_NotifyIconW")
	procLoadImageW           = user32.NewProc("LoadImageW")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
)

// Icon is a system tray icon with a right-click menu.
type Icon struct {
	hwnd      uintptr
	iconData  []byte
	callback  uint32
	cfg       configGetter
	quitFn    func()
	nid       *notifyIconDataW
	registered bool
}

type configGetter interface {
	GetAPIKey() string
}

// New adds an icon to the system tray.
func New(hwnd uintptr, callback uint32, iconData []byte, cfg configGetter, quit func()) (*Icon, error) {
	ic := &Icon{hwnd: hwnd, iconData: iconData, callback: callback, cfg: cfg, quitFn: quit}
	if err := ic.add(); err != nil {
		return nil, err
	}
	return ic, nil
}

// Remove deletes the icon from the tray.
func (i *Icon) Remove() {
	if !i.registered || i.nid == nil {
		return
	}
	_, _, _ = procShell_NotifyIconW.Call(0, uintptr(unsafe.Pointer(i.nid)))
	i.registered = false
}

// ShowMenu shows the right-click context menu.
func (i *Icon) ShowMenu() {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	const (idEnhance = 1001; idQuit = 1004)
	appendItem := func(id uint, label string) {
		ptr, _ := syscall.UTF16PtrFromString(label)
		procAppendMenuW.Call(hMenu, 0, uintptr(id), uintptr(unsafe.Pointer(ptr)))
	}
	appendItem(idEnhance, "Enhance (Ctrl+Shift+E)")
	appendItem(idQuit, "Quit")

	var pt point
	_, _, _ = procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	_, _, _ = procSetForegroundWindow.Call(i.hwnd)
	const TPM_RETURNCMD = 0x0100
	chosen, _, _ := procTrackPopupMenu.Call(
		hMenu, TPM_RETURNCMD,
		uintptr(uint32(pt.X)), uintptr(uint32(pt.Y)),
		0, i.hwnd, 0,
	)
	if chosen == 0 {
		return
	}
	_, _, _ = procPostMessageW.Call(i.hwnd, 0x0111, chosen, 0) // WM_COMMAND
}

// HandleCommand dispatches a menu item click.
func (i *Icon) HandleCommand(id uint32) {
	if id == 1004 { // Quit
		i.quitFn()
	}
}

func (i *Icon) add() error {
	hicon, err := iconToHICON(i.iconData)
	if err != nil {
		log.Printf("iconToHICON failed, using default: %v", err)
		hicon = defaultIcon()
	}
	nid := &notifyIconDataW{
		Size:             uint32(unsafe.Sizeof(notifyIconDataW{})),
		Wnd:              i.hwnd,
		ID:               1,
		Flags:            0x0001 | 0x0002 | 0x0004, // NIF_MESSAGE | NIF_ICON | NIF_TIP
		CallbackMessage:  i.callback,
		Icon:             hicon,
	}
	tip, _ := syscall.UTF16PtrFromString("SparkEnhance")
	copyStr16(nid.Tip[:], tip)

	r, _, _ := procShell_NotifyIconW.Call(0, uintptr(unsafe.Pointer(nid)))
	if r == 0 {
		return syscall.GetLastError()
	}
	i.nid = nid
	i.registered = true
	return nil
}

type point struct {
	X, Y int32
}

// defaultIcon returns the system default app icon.
func defaultIcon() uintptr {
	const IMAGE_ICON = 1
	h, _, _ := procLoadImageW.Call(0, uintptr(32512), IMAGE_ICON, 16, 16, 2) // LR_DEFAULTSIZE=2
	return h
}

// copyStr16 copies a null-terminated UTF-16 string into a fixed buffer.
func copyStr16(dst []uint16, src *uint16) {
	for i := 0; i < len(dst)-1; i++ {
		v := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(src)) + uintptr(i)*2))
		if v == 0 {
			return
		}
		dst[i] = v
	}
}
