//go:build windows

package platform

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	// Hotkey / hook
	procSetWindowsHookExW   = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx     = user32.NewProc("CallNextHookEx")
	procGetAsyncKeyState    = user32.NewProc("GetAsyncKeyState")
	procGetMessageW         = user32.NewProc("GetMessageW")
	procTranslateMessage    = user32.NewProc("TranslateMessage")
	procDispatchMessageW    = user32.NewProc("DispatchMessageW")
	procPeekMessageW        = user32.NewProc("PeekMessageW")

	// Clipboard
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procGetClipboardData = user32.NewProc("GetClipboardData")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	prockeybd_event      = user32.NewProc("keybd_event")

	// Tray
	procShellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")
	procCreateMenu      = user32.NewProc("CreatePopupMenu")
	procAppendMenu      = user32.NewProc("AppendMenuW")
	procTrackMenu       = user32.NewProc("TrackPopupMenu")
	procGetCursorPos    = user32.NewProc("GetCursorPos")
	procSetForeground   = user32.NewProc("SetForegroundWindow")
	procDestroyMenu     = user32.NewProc("DestroyMenu")
	procCreateWindow    = user32.NewProc("CreateWindowExW")
	procRegisterClass   = user32.NewProc("RegisterClassExW")
	procDefWindowProc   = user32.NewProc("DefWindowProcW")
	procLoadImageW      = user32.NewProc("LoadImageW")
	procLoadIconW       = user32.NewProc("LoadIconW")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procPostMessageW    = user32.NewProc("PostMessageW")

	// Misc
	procSleepEx = kernel32.NewProc("SleepEx")
)

const (
	WH_KEYBOARD_LL = 13

	WM_HOTKEY      = 0x0312
	WM_USER_TRAY   = 0x0400
	WM_RBUTTONUP   = 0x0205
	WM_LBUTTONUP   = 0x0202
	WM_COMMAND     = 0x0111
	WM_MOUSEMOVE   = 0x0200
	WM_LBUTTONDOWN = 0x0201
	CF_UNICODETEXT = 13
	gmemMoveable   = 0x0002

	NIM_ADD    = 0x00000000
	NIM_DELETE = 0x00000002
	NIM_MODIFY = 0x00000001

	IDM_ENHANCE  = 1001
	IDM_SETTINGS = 1002
	IDM_QUIT     = 1003
)

// ===== Errors =====

type clipErr struct{ msg string }

func (e *clipErr) Error() string { return e.msg }

var (
	errClip         = &clipErr{"clipboard unavailable"}
	errClipEmpty    = &clipErr{"clipboard empty"}
	errNoForeground = &clipErr{"no foreground window"}
	errInvalidHotkey = &clipErr{"invalid hotkey chord"}
)

// ===== Messages =====

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      struct{ X, Y int32 }
}

func mustUTF16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

// ===== Hotkey parsing =====

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
	return 0, &clipErr{"unknown key: " + name}
}

// ===== Low-level keyboard hook (the Handy's way) =====

var (
	hookCurrentChord atomic.Value // *keyChord
	hookOnFire       atomic.Value // func()
	hookCtx          context.Context
	hookCancel       context.CancelFunc
	hookHandle       uintptr
)

type keyChord struct {
	mods uint32
	vk   uint32
}

// hookProc handles WH_KEYBOARD_LL notifications. Runs in the OS thread
// that called SetWindowsHookExW.
func hookProc(code int32, wParam uintptr, lParam uintptr) uintptr {
	if code >= 0 {
		kbd := (*kbHookStruct)(unsafe.Pointer(lParam))
		if kbd != nil {
			// WM_KEYDOWN = 0x0100
			if uint32(wParam) == 0x0100 {
				chordPtr := hookCurrentChord.Load()
				if chordPtr != nil {
					chord := chordPtr.(*keyChord)
					// Use GetAsyncKeyState to check modifier state
					ctrlHeld := (getAsyncKey(0x11) & 0x8000) != 0
					shiftHeld := (getAsyncKey(0x10) & 0x8000) != 0
					altHeld := (getAsyncKey(0x12) & 0x8000) != 0
					winHeld := (getAsyncKey(0x5B) & 0x8000) != 0

					ctrl := (chord.mods & 0x0002) != 0
					shift := (chord.mods & 0x0004) != 0
					alt := (chord.mods & 0x0001) != 0
					win := (chord.mods & 0x0008) != 0

					if ctrlHeld == ctrl && shiftHeld == shift && altHeld == alt && winHeld == win &&
						uint32(kbd.VkCode) == chord.vk {
						// Suppress the chord so the focused app doesn't see it
						onFire := hookOnFire.Load()
						if onFire != nil {
							go onFire.(func())()
						}
						// Return 1 to swallow the keystroke
						return 1
					}
				}
			}
		}
	}
	r, _, _ := procCallNextHookEx.Call(0, uintptr(code), wParam, lParam)
	return r
}

type kbHookStruct struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

func getAsyncKey(vk int) uint16 {
	r, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	return uint16(r)
}

// listenHotkeyLL installs a global low-level keyboard hook on a dedicated thread.
// The hook is fired by the OS thread that pumped the message; we use a goroutine
// to call onFire so the hook callback returns immediately.
func listenHotkeyLL(ctx context.Context, hotkey string, onFire func()) {
	if hotkey == "" {
		hotkey = "ctrl+shift+e"
	}
	mods, vk, err := parseHotkey(hotkey)
	if err != nil {
		log.Printf("hotkey parse %q: %v", hotkey, err)
		return
	}

	// Cancel any previous hook.
	if hookCancel != nil {
		hookCancel()
		hookCancel = nil
	}

	ctx, cancel := context.WithCancel(ctx)
	hookCtx = ctx
	hookCancel = cancel
	hookCurrentChord.Store(&keyChord{mods: mods, vk: vk})
	hookOnFire.Store(onFire)

	go runHookThread(ctx, mods, vk, onFire)
}

// runHookThread creates a thread with a message loop and installs the hook.
func runHookThread(ctx context.Context, mods, vk uint32, onFire func()) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	cb := syscall.NewCallbackCDecl(func(code int32, wParam, lParam uintptr) uintptr {
		return hookProc(code, wParam, lParam)
	})

	hHook, _, _ := procSetWindowsHookExW.Call(
		uintptr(WH_KEYBOARD_LL),
		cb,
		0, 0,
	)
	if hHook == 0 {
		log.Printf("SetWindowsHookExW failed")
		return
	}
	hookHandle = hHook
	log.Printf("WH_KEYBOARD_LL hook installed: mods=0x%x vk=0x%x", mods, vk)

	// Release thread periodically so the Go scheduler can stop us.
	go func() {
		<-ctx.Done()
		// Post a WM_QUIT so the message loop returns.
		// Use PostThreadMessage-style hack: send a message to ourselves via PeekMessage.
		// Simpler: wait briefly then unhook.
		time.Sleep(200 * time.Millisecond)
		if hHook != 0 {
			procUnhookWindowsHookEx.Call(hHook)
			hookHandle = 0
			log.Printf("WH_KEYBOARD_LL hook removed")
		}
	}()

	// Message loop is REQUIRED for the hook to receive callbacks.
	for {
		var m msg
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 || r == 0xFFFFFFFF {
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// ListenHotkey is the Windows implementation. The cross-platform wrapper
// in platform.go calls listenHotkeyImpl which is this function.
func listenHotkeyImpl(ctx context.Context, hotkey string, onFire func()) {
	go listenHotkeyLL(ctx, hotkey, onFire)
}

// ===== Clipboard =====

func readClipboardImpl() (string, error) {
	ok, _, _ := procOpenClipboard.Call(0)
	if ok == 0 {
		return "", errClip
	}
	defer procCloseClipboard.Call()

	h, _, _ := procGetClipboardData.Call(CF_UNICODETEXT)
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
	for n = 0; n < len(ws) && ws[n] != 0; n++ {
	}
	return syscall.UTF16ToString(ws[:n]), nil
}

func writeClipboardImpl(text string) error {
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
	_, _, _ = procSetClipboardData.Call(CF_UNICODETEXT, h)
	return nil
}

func simulateCopy() error {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return errNoForeground
	}
	_, _, _ = procSetForegroundWindow.Call(hwnd)
	_, _, _ = prockeybd_event.Call(0x11, 0, 0, 0)
	_, _, _ = prockeybd_event.Call(0x43, 0, 0, 0)
	_, _, _ = prockeybd_event.Call(0x43, 0, 2, 0)
	_, _, _ = prockeybd_event.Call(0x11, 0, 2, 0)
	syscall.SyscallN(procSleepEx.Addr(), 150, 0)
	return nil
}

func simulatePaste() error {
	_, _, _ = prockeybd_event.Call(0x11, 0, 0, 0)
	_, _, _ = prockeybd_event.Call(0x56, 0, 0, 0)
	_, _, _ = prockeybd_event.Call(0x56, 0, 2, 0)
	_, _, _ = prockeybd_event.Call(0x11, 0, 2, 0)
	return nil
}

func getSelectionImpl() (string, error) {
	prev, _ := readClipboardImpl()
	if err := simulateCopy(); err != nil {
		return "", err
	}
	sel, err := readClipboardImpl()
	if err != nil || sel == "" {
		if prev != "" {
			_ = writeClipboardImpl(prev)
		}
		return "", err
	}
	if prev != "" && prev != sel {
		_ = writeClipboardImpl(prev)
	}
	return sel, nil
}

func simulatePasteImpl(text string) error {
	if err := writeClipboardImpl(text); err != nil {
		return err
	}
	return simulatePaste()
}

// ===== Tray =====

var (
	cbEnhance  func()
	cbSettings func()
	cbQuit     func()
)

func runTrayImpl(ctx context.Context, onEnhance, onSettings, onQuit func()) {
	cbEnhance, cbSettings, cbQuit = onEnhance, onSettings, onQuit
	go startSystray(ctx)
	<-ctx.Done()
	log.Println("tray: exiting")
}

func startSystray(ctx context.Context) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hInst, _, _ := procGetModuleHandleW.Call(0)
	className := mustUTF16("SparkEnhanceTrayWnd")
	windowName := mustUTF16("SparkEnhanceTray")
	cb := syscall.NewCallbackCDecl(trayWndProc)

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
		le, _, _ := kernel32.NewProc("GetLastError").Call()
		log.Printf("tray: CreateWindow failed (lastErr=%d hInst=0x%x class=%q) — falling back to no-tray mode", le, hInst, className)
		return
	}

	iconPath, _ := iconFilePath()
	hIcon := loadIconFromFile(iconPath)
	if hIcon == 0 {
		hIcon = loadDefaultIcon()
	}
	if hIcon == 0 {
		log.Printf("tray: failed to load icon (file: %s)", iconPath)
		return
	}
	log.Printf("tray: icon loaded from %s (hIcon=0x%x)", iconPath, hIcon)

	nid := buildNotifyIcon(hwnd, hIcon)
	if ok, _, _ := procShellNotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&nid))); ok == 0 {
		log.Printf("tray: Shell_NotifyIcon NIM_ADD failed")
		return
	}
	log.Printf("tray: icon added successfully")

	msgCh := make(chan struct{})
	go func() {
		<-ctx.Done()
		msgCh <- struct{}{}
	}()

	for {
		var m msg
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 {
			return
		}
		select {
		case <-msgCh:
			procShellNotifyIcon.Call(NIM_DELETE, uintptr(unsafe.Pointer(&nid)))
			return
		default:
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func trayWndProc(hwnd, msg, wParam, lParam uintptr) uintptr {
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
	append(IDM_ENHANCE, "Enhance (Ctrl+Shift+E)")
	append(IDM_SETTINGS, "Settings…")
	procAppendMenu.Call(hMenu, 0x0800, 0, 0)
	append(IDM_QUIT, "Quit")

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
		procPostMessageW.Call(hwnd, WM_COMMAND, chosen, 0)
	}
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

// NOTIFYICONDATAW with V3 (NOTIFYICON_VERSION_4) fields.
type notifyIconDataW struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	UState           uint32
	UStateMask       uint32
	SzInfo           [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GUIDItem         [16]byte
	HIconBalloon     uintptr
}

func buildNotifyIcon(hwnd uintptr, hIcon uintptr) notifyIconDataW {
	nid := notifyIconDataW{
		CbSize:            uint32(unsafe.Sizeof(notifyIconDataW{})),
		HWnd:              hwnd,
		UID:               1,
		UFlags:            0x01 | 0x02 | 0x04, // NIF_MESSAGE | NIF_ICON | NIF_TIP
		UCallbackMessage:  WM_USER_TRAY,
		HIcon:             hIcon,
		UTimeoutOrVersion: 4, // NOTIFYICON_VERSION_4 — required for modern Windows
	}
	tip, _ := syscall.UTF16PtrFromString("SparkEnhance — Ctrl+Shift+E")
	copyStr16(nid.SzTip[:], tip)
	return nid
}

func copyStr16(dst []uint16, src *uint16) {
	for i := 0; i < len(dst)-1; i++ {
		v := *(*uint16)(unsafe.Add(unsafe.Pointer(src), uintptr(i)*2))
		if v == 0 {
			return
		}
		dst[i] = v
	}
}

func iconFilePath() (string, string) {
	var exePath [261]uint16
	procGetModuleFileNameW := kernel32.NewProc("GetModuleFileNameW")
	n, _, _ := procGetModuleFileNameW.Call(0, uintptr(unsafe.Pointer(&exePath[0])), 261)
	if n == 0 {
		return "", ""
	}
	full := syscall.UTF16ToString(exePath[:n])
	dir := filepath.Dir(full)
	// Try icon.ico next to exe
	cand := filepath.Join(dir, "icon.ico")
	if _, err := os.Stat(cand); err == nil {
		return cand, dir
	}
	// Try ../windows/icon.ico (Wails build layout)
	cand = filepath.Join(dir, "..", "windows", "icon.ico")
	if _, err := os.Stat(cand); err == nil {
		return cand, dir
	}
	// Last resort
	return cand, dir
}


func loadIconFromFile(path string) uintptr {
	if path == "" {
		return 0
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return 0
	}
	if _, err := os.Stat(abs); err != nil {
		return 0
	}
	absPtr, _ := syscall.UTF16PtrFromString(abs)
	// IMAGE_ICON=1, LR_LOADFROMFILE=0x00000010
	h, _, _ := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(absPtr)),
		1, 0, 0,
		0x00000010,
	)
	return h
}

func loadDefaultIcon() uintptr {
	h, _, _ := procLoadIconW.Call(0, 32512) // IDI_APPLICATION
	return h
}
