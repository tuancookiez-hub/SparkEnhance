// Package bar implements the floating "enhance" result window using raw Win32.
// No external DLL dependencies — all via golang.org/x/sys.
package bar

import (
	"log"
	"sync"
	"syscall"
	"unsafe"
)

const (
	className = "SparkEnhanceBar"
	barWidth  = 400
	barHeight = 130
	btnH      = 24
	btnY      = 96

	closeW = 70
	closeX = barWidth - 20 - closeW
	copyW = 70
	copyX = closeX - 8 - copyW
)

var (
	user32  = syscall.NewLazyDLL("user32.dll")
	gdi32   = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetModuleHandleW     = kernel32.NewProc("GetModuleHandleW")
	procDestroyWindow        = user32.NewProc("DestroyWindow")
	procSetWindowPos         = user32.NewProc("SetWindowPos")
	procShowWindow           = user32.NewProc("ShowWindow")
	procInvalidateRect      = user32.NewProc("InvalidateRect")
	procBeginPaint           = user32.NewProc("BeginPaint")
	procEndPaint             = user32.NewProc("EndPaint")
	procDrawTextW            = user32.NewProc("DrawTextW")
	procGetClientRect        = user32.NewProc("GetClientRect")
	procFillRect             = user32.NewProc("FillRect")
	procSetBkMode            = user32.NewProc("SetBkMode")
	procSetTextColor         = user32.NewProc("SetTextColor")
	procDefWindowProcW       = user32.NewProc("DefWindowProcW")
	procRegisterClassExW     = user32.NewProc("RegisterClassExW")
	procCreateWindowExW      = user32.NewProc("CreateWindowExW")
	procSetCursor            = user32.NewProc("SetCursor")
	procLoadCursorW          = user32.NewProc("LoadCursorW")
	procGetSysColorBrush     = user32.NewProc("GetSysColorBrush")
	procCreateRoundRectRgn   = gdi32.NewProc("CreateRoundRectRgn")
	procSetWindowRgn         = user32.NewProc("SetWindowRgn")
)

type rect struct {
	Left, Top, Right, Bottom int32
}

type paintstruct struct {
	Hdc        uintptr
	FErase     int32
	Rect       rect
	FRestore   int32
	FIncUpdate int32
	ReservedB  [32]byte
}

type wndclassexw struct {
	CbSize         uint32
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

var hCursorHand uintptr

// Window is the floating bar.
type Window struct {
	hwnd            uintptr
	mu              sync.Mutex
	state           State
	lastSelection   string
	lastEnhanced    string
	prevClipboard   string
	autoPaste       bool
	errorText       string
	score           int
}

// State describes what the bar is currently showing.
type State int

const (
	StateHidden State = iota
	StateEnhancing
	StateResult
	StateError
)

var barInstances = map[uintptr]*Window{}

func init() {
	hCursorHand, _, _ = procLoadCursorW.Call(0, uintptr(32649)) // IDC_HAND
}

// New creates the bar window (initially hidden).
func New(autoPaste bool) (*Window, error) {
	w := &Window{autoPaste: autoPaste}

	classPtr, _ := syscall.UTF16PtrFromString(className)
	hInst, _, _ := procGetModuleHandleW.Call(0)

	cls := wndclassexw{
		CbSize:        uint32(unsafe.Sizeof(wndclassexw{})),
		Style:         0x0001 | 0x0002,
		LpfnWndProc:   syscall.NewCallback(barWndProc),
		HInstance:     hInst,
		HbrBackground: 6,
		LpszClassName: classPtr,
	}
	_, _, _ = procRegisterClassExW.Call(uintptr(unsafe.Pointer(&cls)))

	hwnd, _, _ := procCreateWindowExW.Call(
		uintptr(0x08000000|0x00000080|0x00000008), // NOACTIVATE|TOOLWINDOW|TOPMOST
		uintptr(unsafe.Pointer(classPtr)), 0,
		uintptr(0x80000000|0x00800000), // POPUP|BORDER
		0, 0, barWidth, barHeight,
		0, 0, hInst, 0,
	)
	if hwnd == 0 {
		return nil, syscall.GetLastError()
	}
	w.hwnd = hwnd
	barInstances[hwnd] = w

	rgn, _, _ := procCreateRoundRectRgn.Call(0, 0, uintptr(barWidth), uintptr(barHeight), 8, 8)
	if rgn != 0 {
		_, _, _ = procSetWindowRgn.Call(hwnd, rgn, 1)
	}

	return w, nil
}

// Destroy tears down the bar window.
func (w *Window) Destroy() {
	if w.hwnd != 0 {
		delete(barInstances, w.hwnd)
		_, _, _ = procDestroyWindow.Call(w.hwnd)
		w.hwnd = 0
	}
}

// Show places the bar at (x, y) in screen coords and displays it.
func (w *Window) Show(x, y int32, st State) {
	w.mu.Lock()
	w.state = st
	w.errorText = ""
	w.mu.Unlock()

	_, _, _ = procSetWindowPos.Call(
		w.hwnd, 0,
		uintptr(uint32(x+16)), uintptr(uint32(y+16)),
		barWidth, barHeight,
		0x0040, // SWP_SHOWWINDOW
	)
	repaint(w.hwnd)
}

// Hide hides the bar and restores the original clipboard.
func (w *Window) Hide() {
	w.mu.Lock()
	w.state = StateHidden
	w.mu.Unlock()
	_, _, _ = procShowWindow.Call(w.hwnd, 0)

	// Restore original clipboard.
	w.mu.Lock()
	prev := w.prevClipboard
	w.mu.Unlock()
	if prev != "" {
		_ = writeClipboardDirect(prev)
	}
}

// ShowError shows the bar with an error message.
func (w *Window) ShowError(msg string) {
	w.mu.Lock()
	w.state = StateError
	w.errorText = msg
	w.mu.Unlock()
	_, _, _ = procShowWindow.Call(w.hwnd, 5)
	repaint(w.hwnd)
}

// SetSelection stores the original selection text.
func (w *Window) SetSelection(text string, score int) {
	w.mu.Lock()
	w.lastSelection = text
	w.score = score
	w.mu.Unlock()
}

// ShowResult displays the enhanced text and score, remembers prev clipboard.
func (w *Window) ShowResult(selection, enhanced string, score int, prevClipboard string, autoPaste bool) {
	w.mu.Lock()
	w.state = StateResult
	w.lastSelection = selection
	w.lastEnhanced = enhanced
	w.prevClipboard = prevClipboard
	w.autoPaste = autoPaste
	w.score = score
	w.mu.Unlock()
	_, _, _ = procShowWindow.Call(w.hwnd, 5)
	repaint(w.hwnd)
	if autoPaste {
		go w.doPaste()
	}
}

func (w *Window) doPaste() {
	w.mu.Lock()
	text := w.lastEnhanced
	prev := w.prevClipboard
	w.mu.Unlock()
	if text == "" {
		return
	}
	if err := pasteEnhanced(text, prev); err != nil {
		log.Printf("paste error: %v", err)
	}
}

// pasteEnhanced is implemented in bar_paste_windows.go.
func pasteEnhanced(text, prev string) error {
	return doPaste(text, prev)
}

func repaint(hwnd uintptr) {
	_, _, _ = procInvalidateRect.Call(hwnd, 0, 1)
}

// hitTest returns which region (close=1, copy=2, none=0) at (x,y).
func (w *Window) hitTest(x, y int32) int {
	if y >= btnY && y <= btnY+btnH {
		if x >= closeX && x <= closeX+closeW {
			return 1
		}
		if x >= copyX && x <= copyX+copyW {
			return 2
		}
	}
	return 0
}
