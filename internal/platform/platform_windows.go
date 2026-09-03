//go:build windows

package platform

import (
	"context"
	"log"
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterHotKey     = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey   = user32.NewProc("UnregisterHotKey")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procPeekMessageW       = user32.NewProc("PeekMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")

	procOpenClipboard     = user32.NewProc("OpenClipboard")
	procCloseClipboard    = user32.NewProc("CloseClipboard")
	procEmptyClipboard    = user32.NewProc("EmptyClipboard")
	procGetClipboardData  = user32.NewProc("GetClipboardData")
	procSetClipboardData  = user32.NewProc("SetClipboardData")
	procGlobalAlloc       = kernel32.NewProc("GlobalAlloc")
	procGlobalFree        = kernel32.NewProc("GlobalFree")
	procGlobalLock        = kernel32.NewProc("GlobalLock")
	procGlobalUnlock      = kernel32.NewProc("GlobalUnlock")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	prockeybd_event       = user32.NewProc("keybd_event")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
)

// parseHotkey parses "ctrl+shift+e" into a (mods, vk) pair.
func parseHotkey(s string) (mods, vk uint32, err error) {
	for _, part := range strings.Split(strings.ToLower(s), "+") {
		part = strings.TrimSpace(part)
		switch part {
		case "ctrl", "control":
			mods |= 0x0002
		case "shift":
			mods |= 0x0004
		case "alt":
			mods |= 0x0001
		case "win":
			mods |= 0x0008
		default:
			if len(part) == 1 && part[0] >= 'a' && part[0] <= 'z' {
				vk = uint32(part[0] - 'a' + 'A')
			} else {
				vk, err = nameVK(part)
			}
		}
	}
	if vk == 0 {
		err = errInvalidHotkey
	}
	return
}

var errInvalidHotkey = &hotkeyErr{"invalid hotkey chord"}

type hotkeyErr struct{ msg string }

func (e *hotkeyErr) Error() string { return e.msg }

func nameVK(name string) (uint32, error) {
	m := map[string]uint32{
		"f1": 0x70, "f2": 0x71, "f3": 0x72, "f4": 0x73,
		"f5": 0x74, "f6": 0x75, "f7": 0x76, "f8": 0x77,
		"f9": 0x78, "f10": 0x79, "f11": 0x7a, "f12": 0x7b,
		"space": 0x20, "enter": 0x0d, "return": 0x0d,
		"escape": 0x1b, "esc": 0x1b, "tab": 0x09,
	}
	if v, ok := m[name]; ok {
		return v, nil
	}
	return 0, &hotkeyErr{"unknown key: " + name}
}

// register returns a non-zero hotkey ID on success.
func register(mods, vk uint32) int {
	const id = 0xA001 // arbitrary unique id
	r, _, _ := procRegisterHotKey.Call(0, id, uintptr(mods), uintptr(vk))
	if r == 0 {
		return 0
	}
	return id
}

func unregister(id int) {
	_, _, _ = procUnregisterHotKey.Call(0, uintptr(id))
}

// pumpHotkeyMessages polls for WM_HOTKEY (0x0312) and signals msgCh.
func pumpHotkeyMessages(id int, msgCh chan<- struct{}) {
	var msg struct {
		Hwnd    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      struct{ X, Y int32 }
		LPriv   uint32
	}
	for {
		// PeekMessage with PM_NOREMOVE so we don't actually dispatch.
		r, _, _ := procPeekMessageW.Call(
			uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0x0001, // PM_NOREMOVE
		)
		if r == 0 {
			return
		}
		if msg.Message == 0x0312 && int(msg.WParam) == id {
			msgCh <- struct{}{}
			// Discard this message so we don't loop on it.
			r2, _, _ := procGetMessageW.Call(
				uintptr(unsafe.Pointer(&msg)), 0, 0, 0,
			)
			if r2 == 0 {
				return
			}
			_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		} else {
			// Brief sleep to avoid busy-looping.
			syscall.SyscallN(procSleepEx.Addr(), 100, 1)
		}
	}
}

var procSleepEx = kernel32.NewProc("SleepEx")

// ====== Clipboard ======

func ReadClipboard() (string, error) {
	ok, _, _ := procOpenClipboard.Call(0)
	if ok == 0 {
		return "", errClip
	}
	defer procCloseClipboard.Call()

	h, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if h == 0 {
		return "", errClipEmpty
	}
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return "", errClip
	}
	defer procGlobalUnlock.Call(h)

	ws := (*[1 << 20]uint16)(unsafe.Pointer(ptr))
	var n int
	for n = 0; ws[n] != 0; n++ {
	}
	return syscall.UTF16ToString(ws[:n]), nil
}

func WriteClipboard(text string) error {
	ok, _, _ := procOpenClipboard.Call(0)
	if ok == 0 {
		return errClip
	}
	defer procCloseClipboard.Call()
	_, _, _ = procEmptyClipboard.Call()

	utf16 := syscall.StringToUTF16(text)
	bytes := unsafe.Slice((*byte)(unsafe.Pointer(&utf16[0])), len(utf16)*2)

	h, _, _ := procGlobalAlloc.Call(gmemMoveable, uintptr(len(bytes)))
	if h == 0 {
		return errClip
	}
	defer procGlobalFree.Call(h)
	ptr, _, _ := procGlobalLock.Call(h)
	if ptr == 0 {
		return errClip
	}
	defer procGlobalUnlock.Call(h)
	dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(bytes))
	copy(dst, bytes)
	_, _, _ = procSetClipboardData.Call(cfUnicodeText, h)
	return nil
}

func simulateCopy() error {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return errNoForeground
	}
	_, _, _ = procSetForegroundWindow.Call(hwnd)
	// Ctrl down
	_, _, _ = prockeybd_event.Call(0x11, 0, 0, 0)
	_, _, _ = prockeybd_event.Call(0x43, 0, 0, 0) // C down
	_, _, _ = prockeybd_event.Call(0x43, 0, 2, 0) // C up
	_, _, _ = prockeybd_event.Call(0x11, 0, 2, 0) // Ctrl up
	syscall.SyscallN(procSleepEx.Addr(), 100, 0)
	return nil
}

func simulatePaste() error {
	_, _, _ = prockeybd_event.Call(0x11, 0, 0, 0) // Ctrl down
	_, _, _ = prockeybd_event.Call(0x56, 0, 0, 0) // V down
	_, _, _ = prockeybd_event.Call(0x56, 0, 2, 0) // V up
	_, _, _ = prockeybd_event.Call(0x11, 0, 2, 0) // Ctrl up
	return nil
}

// Error sentinels.
var (
	errClip       = &hotkeyErr{"clipboard unavailable"}
	errClipEmpty  = &hotkeyErr{"clipboard empty"}
	errNoForeground = &hotkeyErr{"no foreground window"}
)

// ====== Tray ======

func runTray(ctx context.Context, onEnhance, onSettings, onQuit func()) {
	cbEnhance, cbSettings, cbQuit = onEnhance, onSettings, onQuit
	go startSystray(ctx)
	<-ctx.Done()
	log.Println("tray: exiting")
}
