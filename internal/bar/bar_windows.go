package bar

import (
	"syscall"
	"unsafe"
)

func barWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	b, ok := barInstances[hwnd]
	if !ok {
		return 0
	}

	switch msg {
	case 0x0200: // WM_MOUSEMOVE
		x := int32(wParam & 0xFFFF)
		y := int32((wParam >> 16) & 0xFFFF)
		if b.hitTest(x, y) > 0 && hCursorHand != 0 {
			_, _, _ = procSetCursor.Call(hCursorHand)
		}

	case 0x0201: // WM_LBUTTONDOWN
		x := int32(lParam & 0xFFFF)
		y := int32((lParam >> 16) & 0xFFFF)
		switch b.hitTest(x, y) {
		case 1: // dismiss
			b.Hide()
			return 0
		case 2: // copy
			b.mu.Lock()
			text := b.lastEnhanced
			b.mu.Unlock()
			if text != "" {
				_ = writeClipboardDirect(text)
			}
			return 0
		}

	case 0x0100: // WM_KEYDOWN
		if wParam == 0x1B { // ESC
			b.Hide()
			return 0
		}

	case 0x000F: // WM_PAINT
		drawBar(b)
		return 0

	case 0x0010: // WM_CLOSE
		b.Hide()
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func drawBar(b *Window) {
	var ps paintstruct
	r, _, _ := procBeginPaint.Call(b.hwnd, uintptr(unsafe.Pointer(&ps)))
	defer procEndPaint.Call(b.hwnd, r)

	// Background.
	brush, _, _ := procGetSysColorBrush.Call(15) // COLOR_3DFACE
	_, _, _ = procFillRect.Call(r, uintptr(unsafe.Pointer(&ps.Rect)), brush)

	// Text.
	_, _, _ = procSetTextColor.Call(r, 0x00F5E0DC)
	_, _, _ = procSetBkMode.Call(r, 1) // TRANSPARENT

	// Gather state.
	b.mu.Lock()
	state := b.state
	title, body := b.stateTitleBody()
	score := b.score
	b.mu.Unlock()

	// Title.
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	var rc rect
	_, _, _ = procGetClientRect.Call(b.hwnd, uintptr(unsafe.Pointer(&rc)))
	rc.Right -= 80
	_, _, _ = procDrawTextW.Call(
		r, uintptr(unsafe.Pointer(titlePtr)), uintptr(uint32(len(title)-1)),
		uintptr(unsafe.Pointer(&rc)),
		0x0024, // DT_SINGLELINE|DT_VCENTER
	)

	// Score badge.
	if state == StateResult && score > 0 {
		badge := syscall.StringToUTF16(" " + itoa(score) + " ")
		_, _, _ = procSetTextColor.Call(r, uintptr(scoreColor(score)))
		badgeRc := rect{Right: barWidth - 10, Top: 8, Bottom: 30}
		_, _, _ = procDrawTextW.Call(
			r, uintptr(unsafe.Pointer(&badge[0])), uintptr(uint32(len(badge)-1)),
			uintptr(unsafe.Pointer(&badgeRc)),
			0x0004|0x0002|0x0020, // SINGLELINE|RIGHT|VCENTER
		)
		_, _, _ = procSetTextColor.Call(r, 0x00F5E0DC)
	}

	// Body text.
	if body != "" {
		bodyPtr, _ := syscall.UTF16PtrFromString(body)
		bodyRc := rect{Left: 12, Top: 34, Right: barWidth - 12, Bottom: btnY - 6}
		_, _, _ = procDrawTextW.Call(
			r, uintptr(unsafe.Pointer(bodyPtr)), uintptr(uint32(len(body)-1)),
			uintptr(unsafe.Pointer(&bodyRc)),
			0x0024|0x0010, // SINGLELINE|WORDBREAK
		)
	}

	// Buttons.
	drawButton(r, closeX, btnY, closeW, btnH, "Dismiss")
	if state == StateResult {
		drawButton(r, copyX, btnY, copyW, btnH, "Copy")
	}
}

func drawButton(r uintptr, x, y, w, h int32, label string) {
	brush, _, _ := procGetSysColorBrush.Call(16) // COLOR_BTNFACE
	_, _, _ = procFillRect.Call(r, uintptr(unsafe.Pointer(&rect{x, y, x + w, y + h})), brush)

	labelPtr, _ := syscall.UTF16PtrFromString(label)
	_, _, _ = procSetTextColor.Call(r, 0x00808080)
	_, _, _ = procSetBkMode.Call(r, 1)
	_, _, _ = procDrawTextW.Call(
		r, uintptr(unsafe.Pointer(labelPtr)), uintptr(uint32(len(label)-1)),
		uintptr(unsafe.Pointer(&rect{x, y, x + w, y + h})),
		0x0004|0x0008|0x0020, // SINGLELINE|CENTER|VCENTER
	)
	_, _, _ = procSetTextColor.Call(r, 0x00F5E0DC)
}

func (b *Window) stateTitleBody() (title, body string) {
	switch b.state {
	case StateEnhancing:
		return "SparkEnhance", "Calling M3 to rewrite your prompt..."
	case StateError:
		return "Error", b.errorText
	case StateResult:
		// Show first 300 chars of enhanced text as preview.
		text := b.lastEnhanced
		if len(text) > 300 {
			text = text[:300] + "…"
		}
		return "Enhanced", text
	default:
		return "SparkEnhance", "Press Ctrl+Shift+E to enhance any text"
	}
}

func scoreColor(score int) uint32 {
	switch {
	case score >= 70:
		return 0x0090EE90 // green
	case score >= 50:
		return 0x00FFA500 // orange
	default:
		return 0x00FF6B6B // red
	}
}

// itoa converts a small non-negative int to decimal string.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	const digits = "0123456789"
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = digits[n%10]
		n /= 10
	}
	return string(buf[i:])
}
