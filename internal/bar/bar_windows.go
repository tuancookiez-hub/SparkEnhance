package bar

import (
	"syscall"
	"unsafe"
)

// wndProc handles the bar's messages.
func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case 0x000F: // WM_PAINT
		onPaint(hwnd)
		return 0
	case 0x0010: // WM_CLOSE
		_, _, _ = procShowWindow.Call(hwnd, 0)
		return 0
	case 0x0200: // WM_MOUSEMOVE — hide when mouse leaves the bar after a delay
		// (not yet implemented: tracking + auto-hide timer)
	}
	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

// getBar returns the bar instance for the given hwnd. Used by the wndProc
// to look up per-instance state.
var bars = map[uintptr]*Window{}

func registerBar(w *Window) {
	bars[w.hwnd] = w
}

func onPaint(hwnd uintptr) {
	var ps paintstruct
	r, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	defer procEndPaint.Call(hwnd, r)

	// Solid dark background.
	bg := struct{ Color uint32 }{0x00231B36} // purple-ish dark
	_, _, _ = procFillRect.Call(r, uintptr(unsafe.Pointer(&ps.Rect)), uintptr(unsafe.Pointer(&bg)))

	// Foreground color (light text).
	_, _, _ = procSetTextColor.Call(r, 0x00F5E0DC) // light pinkish
	_, _, _ = procSetBkMode.Call(r, 1)              // TRANSPARENT

	// Title.
	bar := bars[hwnd]
	title := "✨ SparkEnhance — working..."
	body := "Calling M3 to rewrite your prompt."
	if bar != nil {
		switch bar.state {
		case StateError:
			title = "⚠ SparkEnhance — error"
			body = bar.errorText
		case StateResult:
			title = "✓ SparkEnhance — done"
			body = "Click to paste  ·  Esc to dismiss"
		}
	}

	titlePtr, _ := syscall.UTF16FromString(title)
	bodyPtr, _ := syscall.UTF16FromString(body)
	var rc rect
	_, _, _ = procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))

	// Title at top.
	rc.Top = 8
	rc.Left = 12
	rc.Right -= 12
	rc.Bottom = 32
	_, _, _ = procDrawTextW.Call(
		r, uintptr(unsafe.Pointer(&titlePtr[0])), uintptr(len(titlePtr)-1),
		uintptr(unsafe.Pointer(&rc)),
		0x0004|0x0020, // DT_SINGLELINE | DT_VCENTER
	)

	// Body below.
	rc.Top = 36
	rc.Bottom -= 8
	_, _, _ = procDrawTextW.Call(
		r, uintptr(unsafe.Pointer(&bodyPtr[0])), uintptr(len(bodyPtr)-1),
		uintptr(unsafe.Pointer(&rc)),
		0x0004|0x0010, // DT_SINGLELINE | DT_WORDBREAK
	)
}
