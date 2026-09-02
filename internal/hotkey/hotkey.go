package hotkey

import (
	"fmt"
	"sync"
)

// Hotkey fires a callback when the registered key chord is pressed.
// All methods are safe for concurrent use.
type Hotkey struct {
	id      int
	hmod    uint32
	keys    []uint32
	callback func()
	hwnd    uintptr

	registerOnce sync.Once
	ready       chan struct{} // closed when RegisterHotKey succeeds
}

const (
	modCtrl  = 0x0002
	modShift = 0x0004
	modAlt   = 0x0001
	modWin   = 0x0008
)

// Parse parses a chord string like "ctrl+shift+e" into modifiers + vk.
func Parse(chord string) (hmod uint32, vk uint32, err error) {
	hmod = 0
	vk = 0
	for _, part := range parseChord(chord) {
		switch part {
		case "ctrl", "control":
			hmod |= modCtrl
		case "shift":
			hmod |= modShift
		case "alt":
			hmod |= modAlt
		case "win":
			hmod |= modWin
		default:
			// single char: 'e' → VK_E
			if len(part) == 1 {
				vk = uint32(part[0])
				if part[0] >= 'a' && part[0] <= 'z' {
					vk = uint32(part[0] - 'a' + 'A')
				}
			} else {
				vk, err = nameToVK(part)
				if err != nil {
					return 0, 0, err
				}
			}
		}
	}
	return hmod, vk, nil
}

func parseChord(chord string) []string {
	const sep = "+"
	parts := make([]string, 0, 4)
	start := 0
	for i := 0; i <= len(chord); i++ {
		if i == len(chord) || chord[i] == '+' {
			p := chord[start:i]
			for ; len(p) > 0 && (p[0] == ' ' || p[0] == '\t'); p = p[1:] {}
			for ; len(p) > 0 && (p[len(p)-1] == ' ' || p[len(p)-1] == '\t'); p = p[:len(p)-1] {}
			if p != "" {
				parts = append(parts, p)
			}
			start = i + 1
		}
	}
	return parts
}

// nameToVK maps named keys to their virtual-key codes.
func nameToVK(name string) (uint32, error) {
	m := map[string]uint32{
		"a": 0x41, "b": 0x42, "c": 0x43, "d": 0x44, "e": 0x45,
		"f": 0x46, "g": 0x47, "h": 0x48, "i": 0x49, "j": 0x4a,
		"k": 0x4b, "l": 0x4c, "m": 0x4d, "n": 0x4e, "o": 0x4f,
		"p": 0x50, "q": 0x51, "r": 0x52, "s": 0x53, "t": 0x54,
		"u": 0x55, "v": 0x56, "w": 0x57, "x": 0x58, "y": 0x59, "z": 0x5a,
		"0": 0x30, "1": 0x31, "2": 0x32, "3": 0x33, "4": 0x34,
		"5": 0x35, "6": 0x36, "7": 0x37, "8": 0x38, "9": 0x39,
		"f1": 0x70, "f2": 0x71, "f3": 0x72, "f4": 0x73,
		"f5": 0x74, "f6": 0x75, "f7": 0x76, "f8": 0x77,
		"f9": 0x78, "f10": 0x79, "f11": 0x7a, "f12": 0x7b,
		"space":     0x20,
		"enter":     0x0d,
		"return":    0x0d,
		"escape":    0x1b,
		"esc":       0x1b,
		"tab":       0x09,
		"backspace": 0x08,
		"delete":    0x2e,
		"del":       0x2e,
		"home":      0x24,
		"end":       0x23,
		"pageup":    0x21,
		"pagedown":  0x22,
		"left":      0x25,
		"right":     0x27,
		"up":        0x26,
		"down":      0x28,
		"insert":    0x2d,
		"ins":       0x2d,
	}
	if vk, ok := m[name]; ok {
		return vk, nil
	}
	return 0, fmt.Errorf("unknown key: %s", name)
}
