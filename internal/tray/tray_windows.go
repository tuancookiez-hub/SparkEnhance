package tray

import (
	"syscall"
	"unsafe"
)

// notifyIconDataW mirrors NOTIFYICONDATAW (minimum version).
type notifyIconDataW struct {
	Size            uint32
	Wnd             uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
}

// iconToHICON loads an HICON from raw .ico bytes.
func iconToHICON(data []byte) (uintptr, error) {
	if len(data) < 22 {
		return 0, syscall.EINVAL
	}
	// Read the offset and size from the first ICONDIRENTRY (bytes 6-21).
	offset := int(uint32(data[10]) | uint32(data[11])<<8 | uint32(data[12])<<16 | uint32(data[13])<<24)
	size := int(uint32(data[14]) | uint32(data[15])<<8 | uint32(data[16])<<16 | uint32(data[17])<<24)
	if offset+size > len(data) {
		return 0, syscall.EINVAL
	}
	imgData := data[offset : offset+size]
	const LR_DEFAULTCOLOR = 0x00000002
	h, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&imgData[0])),
		uintptr(len(imgData)),
		1, // fIcon
		uintptr(0x00030000),
		0, 0, // auto-size from resource
		LR_DEFAULTCOLOR,
	)
	if h != 0 {
		return h, nil
	}
	return defaultIcon(), nil
}
