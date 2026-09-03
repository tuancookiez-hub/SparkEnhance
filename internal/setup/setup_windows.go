// Package setup implements the first-run dialog: collect the GMI API key.
// The dialog is a native Win32 window with a text input and OK / Cancel.
package setup

import (
	"log"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const className = "SparkEnhanceSetup"

var (
	user32 = syscall.NewLazyDLL("user32.dll")

	procGetModuleHandleW   = user32.NewProc("GetModuleHandleW")
	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procGetMessageW        = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessageW   = user32.NewProc("DispatchMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procSetWindowTextW     = user32.NewProc("SetWindowTextW")
	procGetWindowTextW     = user32.NewProc("GetWindowTextW")
	procGetDlgItem         = user32.NewProc("GetDlgItem")
	procSendMessageW       = user32.NewProc("SendMessageW")
	procSetFocus           = user32.NewProc("SetFocus")
	procMessageBeep        = user32.NewProc("MessageBeep")
	procEndDialog          = user32.NewProc("EndDialog")
	procDestroyWindow      = user32.NewProc("DestroyWindow")
	procShowWindow         = user32.NewProc("ShowWindow")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
)

// IDs for child controls.
const (
	idLabel   = 1001
	idLabel2  = 1002
	idInput   = 1003
	idGetKey  = 1004
	idOK      = 1005
	idCancel  = 1006
)

var (
	inputHwnd uintptr
	result    string
	ok        bool
)

// PromptAPIKey runs the setup dialog modally and returns the entered key
// and whether OK was pressed. The function is synchronous.
func PromptAPIKey() (string, bool) {
	classNamePtr, _ := syscall.UTF16PtrFromString(className)
	hInst, _, _ := procGetModuleHandleW.Call(0)

	cls := wndclassexw{
		CbSize:        uint32(unsafe.Sizeof(wndclassexw{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInst,
		HbrBackground: 15, // COLOR_3DFACE
		LpszClassName: classNamePtr,
	}
	_, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&cls)))

	const (
		WS_OVERLAPPED = 0x00C00000
		WS_CAPTION    = 0x00C00000
		WS_SYSMENU    = 0x00080000
		WS_VISIBLE    = 0x10000000
		WS_CHILD      = 0x40000000
		WS_TABSTOP    = 0x00010000
		WS_BORDER     = 0x00800000
		BS_PUSHBUTTON = 0x00000000
		BS_DEFPUSHBUTTON = 0x00000001
		ES_AUTOHSCROLL = 0x00000004
		ES_PASSWORD    = 0x00000020
	)

	titlePtr, _ := syscall.UTF16PtrFromString("SparkEnhance — Setup")
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(WS_OVERLAPPED|WS_CAPTION|WS_SYSMENU|WS_VISIBLE),
		200, 150, 520, 240,
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		log.Printf("setup: CreateWindowExW failed")
		return "", false
	}
	defer procDestroyWindow.Call(hwnd)

	// Label 1
	labelPtr, _ := syscall.UTF16PtrFromString("GMI Cloud API key")
	hLabel, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(mustPtr("STATIC"))), uintptr(unsafe.Pointer(labelPtr)),
		uintptr(WS_CHILD|WS_VISIBLE), 16, 16, 460, 20,
		hwnd, idLabel, hInst, 0,
	)
	_ = hLabel

	// Label 2
	hintPtr, _ := syscall.UTF16PtrFromString("Get a free key at https://console.gmicloud.ai")
	hHint, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(mustPtr("STATIC"))), uintptr(unsafe.Pointer(hintPtr)),
		uintptr(WS_CHILD|WS_VISIBLE), 16, 36, 460, 18,
		hwnd, idLabel2, hInst, 0,
	)
	_ = hHint

	// API key input
	inputHwnd, _, _ = procCreateWindowExW.Call(
		0x00000200, // WS_EX_CLIENTEDGE
		uintptr(unsafe.Pointer(mustPtr("EDIT"))), 0,
		uintptr(WS_CHILD|WS_VISIBLE|WS_TABSTOP|WS_BORDER|ES_AUTOHSCROLL|ES_PASSWORD),
		16, 60, 460, 24,
		hwnd, idInput, hInst, 0,
	)

	// Buttons
	okPtr, _ := syscall.UTF16PtrFromString("Save and start")
	hOK, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(mustPtr("BUTTON"))), uintptr(unsafe.Pointer(okPtr)),
		uintptr(WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_DEFPUSHBUTTON),
		260, 160, 100, 28,
		hwnd, idOK, hInst, 0,
	)
	_ = hOK

	cancelPtr, _ := syscall.UTF16PtrFromString("Cancel")
	hCancel, _, _ := procCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(mustPtr("BUTTON"))), uintptr(unsafe.Pointer(cancelPtr)),
		uintptr(WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_PUSHBUTTON),
		376, 160, 100, 28,
		hwnd, idCancel, hInst, 0,
	)
	_ = hCancel

	// Focus the input
	_, _, _ = procSetFocus.Call(inputHwnd)

	// Run the message loop until PostQuitMessage.
	runMessageLoop()

	return result, ok
}

func runMessageLoop() {
	for {
		var m msg
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 {
			return
		}
		if r == 0xFFFFFFFF {
			return
		}
		_, _, _ = procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		_, _, _ = procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
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
	Pt       struct{ X, Y int32 }
	LPrivate uint32
}

func mustPtr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		log.Panicf("UTF16PtrFromString: %v", err)
	}
	return p
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case 0x0111: // WM_COMMAND
		id := uint32(wParam & 0xFFFF)
		switch id {
		case idOK:
			text := readEdit(inputHwnd)
			if text == "" {
				_, _, _ = procMessageBeep.Call(0)
				return 0
			}
			result = text
			ok = true
			_, _, _ = procPostQuitMessage.Call(0)
			return 0
		case idCancel:
			ok = false
			_, _, _ = procPostQuitMessage.Call(0)
			return 0
		}
	case 0x0002: // WM_DESTROY
		_, _, _ = procPostQuitMessage.Call(0)
		return 0
	case 0x0100: // WM_KEYDOWN
		if wParam == 0x1B { // ESC
			ok = false
			_, _, _ = procPostQuitMessage.Call(0)
		}
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func readEdit(hwnd uintptr) string {
	buf := make([]uint16, 512)
	r, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 512)
	if r == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:r])
}
