//go:build ignore

// Generate assets/icon.ico — run with: go run internal/icon/main.go
// Produces a multi-resolution .ico (16, 32, 48) with a sparkle gradient icon.
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	sizes := []int{16, 32, 48}
	imgs := make([]image.Image, len(sizes))
	for i, s := range sizes {
		imgs[i] = drawIcon(s)
	}
	ico := buildICO(imgs)
	if err := os.WriteFile("assets/icon.ico", ico, 0644); err != nil {
		panic(err)
	}
	println("written assets/icon.ico")
}

func drawIcon(size int) image.Image {
	m := image.NewRGBA(image.Rect(0, 0, size, size))
	// Purple background
	bg := color.RGBA{R: 124, G: 58, B: 237, A: 255} // #7C3AED
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			m.Set(x, y, bg)
		}
	}
	// White sparkle (star burst)
	cx, cy := size/2, size/2
	rOuter := size / 3
	rInner := size / 8
	gold := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	// 4-point star
	for y := -rOuter; y <= rOuter; y++ {
		for x := -rOuter; x <= rOuter; x++ {
			// Manhattan distance gives a 4-point star
			if abs(x)+abs(y) <= rOuter && (x != 0 || y != 0) {
				// Make the cross thicker near the center
				if abs(x) <= rInner || abs(y) <= rInner {
					px, py := cx+x, cy+y
					if px >= 0 && px < size && py >= 0 && py < size {
						m.Set(px, py, gold)
					}
				} else if (abs(x)+abs(y)) <= rOuter-(rOuter-rInner) {
					// thin cross arms
					px, py := cx+x, cy+y
					if px >= 0 && px < size && py >= 0 && py < size {
						m.Set(px, py, gold)
					}
				}
			}
		}
	}
	return m
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func buildICO(imgs []image.Image) []byte {
	n := len(imgs)
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint16(0))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(n))

	pngs := make([][]byte, n)
	offset := 6 + n*16

	for i, img := range imgs {
		b := new(bytes.Buffer)
		png.Encode(b, img)
		pngs[i] = b.Bytes()
		w := img.Bounds().Dx()
		h := img.Bounds().Dy()
		if w >= 256 {
			w = 0
		}
		if h >= 256 {
			h = 0
		}
		binary.Write(buf, binary.LittleEndian, uint8(w))
		binary.Write(buf, binary.LittleEndian, uint8(h))
		binary.Write(buf, binary.LittleEndian, uint8(0))
		binary.Write(buf, binary.LittleEndian, uint8(0))
		binary.Write(buf, binary.LittleEndian, uint16(1))
		binary.Write(buf, binary.LittleEndian, uint16(32))
		binary.Write(buf, binary.LittleEndian, uint32(len(pngs[i])))
		binary.Write(buf, binary.LittleEndian, uint32(offset))
		offset += len(pngs[i])
	}
	for _, p := range pngs {
		buf.Write(p)
	}
	return buf.Bytes()
}
