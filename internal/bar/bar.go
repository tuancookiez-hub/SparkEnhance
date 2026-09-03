// Package bar implements the floating "enhance" window using raw Win32.
package bar

import (
	"log"
	"sync"
	"syscall"
	"unsafe"
)

const className = "SparkEnhanceBar"

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	gdi32                  = syscall.NewLazyDLL("gdi32.dll")
	procGetModuleHandleW   = user32.NewProc("GetModuleHandleW")
	procDestroyWindow      = user32.NewProc("DestroyWindow")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procShowWindow         = user32.NewProc("ShowWindow")
	procInvalidateRect     = user32.NewProc("InvalidateRect")
	procBeginPaint         = user32.NewProc("BeginPaint")
	procEndPaint           = user32.NewProc("EndPaint")
	procDrawTextW          = user32.NewProc("DrawTextW")
	procGetClientRect      = user32.NewProc("GetClientRect")
	procFillRect           = user32.NewProc("FillRect")
	procSetBkMode          = user32.NewProc("SetBkMode")
	procSetTextColor       = user32.NewProc("SetTextColor")
	procDefWindowProcW     = user32.NewProc("DefWindowProcW")
	procRegisterClassExW   = user32.NewProc("RegisterClassExW")
	procCreateWindowExW    = user32.NewProc("CreateWindowExW")
	procPostMessageW       = user32.NewProc("PostMessageW")
)

// Window is the floating bar.
type Window struct {
	hwnd   uintptr
	mu     sync.Mutex
	state  State
	lastSelection string
	lastEnhanced  string
	prevClipboard string
	autoPaste     bool
	errorText     string
}

// State describes what the bar is currently showing.
type State int

const (
	StateHidden State = iota
	StateEnhancing
	StateResult
	StateError
)

type rect struct {
	Left, Top, Right, Bottom int32
}

type paintstruct struct {
	Hdc         uintptr
	FErase      int32
	Rect        rect
	FRestore    int32
	FIncUpdate  int32
	ReservedB   [32]byte
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

// New creates the bar window (initially hidden). autoPaste is the configured
// value at construction time; changing it requires recreating the bar.
func New(autoPaste bool) (*Window, error) {
	w := &Window{autoPaste: autoPaste}

	classNamePtr, _ := syscall.UTF16PtrFromString(className)
	hInst, _, _ := procGetModuleHandleW.Call(0)

	const CS_HREDRAW = 0x0002
	cls := wndclassexw{
		CbSize:        uint32(unsafe.Sizeof(wndclassexw{})),
		Style:         CS_HREDRAW,
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInst,
		HbrBackground: 6, // COLOR_3DFACE
		LpszClassName: classNamePtr,
	}
	_, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&cls)))

	const (
		WS_POPUP         = 0x80000000
		WS_BORDER        = 0x00800000
		WS_VISIBLE       = 0x10000000
		WS_EX_NOACTIVATE = 0x08000000
		WS_EX_TOOLWINDOW = 0x00000080
		WS_EX_TOPMOST    = 0x00000008
	)

	hwnd, _, _ := procCreateWindowExW.Call(
		uintptr(WS_EX_NOACTIVATE|WS_EX_TOOLWINDOW|WS_EX_TOPMOST),
		uintptr(unsafe.Pointer(classNamePtr)),
		0,
		uintptr(WS_POPUP|WS_BORDER),
		0, 0, 360, 80,
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		return nil, syscall.GetLastError()
	}
	w.hwnd = hwnd
	return w, nil
}

// Destroy tears down the bar window.
func (w *Window) Destroy() {
	if w.hwnd == 0 {
		return
	}
	_, _, _ = procDestroyWindow.Call(w.hwnd)
	w.hwnd = 0
}

// Show places the bar at (x, y) in screen coords and displays it.
func (w *Window) Show(x, y int32, st State) {
	w.mu.Lock()
	w.state = st
	w.errorText = ""
	w.mu.Unlock()
	pos := struct{ X, Y int32 }{x + 16, y + 16}
	_, _, _ = procSetWindowPos.Call(
		w.hwnd, 0,
		uintptr(uint32(pos.X)), uintptr(uint32(pos.Y)),
		360, 80,
		0x0040, // SWP_SHOWWINDOW
	)
	repaint(w.hwnd)
}

// Hide hides the bar.
func (w *Window) Hide() {
	w.mu.Lock()
	w.state = StateHidden
	w.mu.Unlock()
	_, _, _ = procShowWindow.Call(w.hwnd, 0)
}

// ShowError shows the bar with an error message.
func (w *Window) ShowError(msg string) {
	w.mu.Lock()
	w.state = StateError
	w.errorText = msg
	w.mu.Unlock()
	_, _, _ = procShowWindow.Call(w.hwnd, 5) // SW_SHOW
	repaint(w.hwnd)
}

// SetSelection remembers the original selection for the result.
func (w *Window) SetSelection(text string, score int) {
	w.mu.Lock()
	w.lastSelection = text
	w.mu.Unlock()
}

// ShowResult displays the enhanced text and the score, and remembers the
// previous clipboard contents to restore on dismiss.
func (w *Window) ShowResult(selection, enhanced string, score int, prevClipboard string, autoPaste bool) {
	w.mu.Lock()
	w.state = StateResult
	w.lastSelection = selection
	w.lastEnhanced = enhanced
	w.prevClipboard = prevClipboard
	w.autoPaste = autoPaste
	w.mu.Unlock()
	_, _, _ = procShowWindow.Call(w.hwnd, 5)
	repaint(w.hwnd)
	if autoPaste {
		go w.doPaste()
	}
}

func (w *Window) doPaste() {
	if w.lastEnhanced == "" {
		return
	}
	if err := pasteEnhanced(w.lastEnhanced, w.prevClipboard); err != nil {
		log.Printf("paste error: %v", err)
	}
}

func repaint(hwnd uintptr) {
	_, _, _ = procInvalidateRect.Call(hwnd, 0, 1)
}
