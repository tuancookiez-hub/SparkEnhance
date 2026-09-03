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
	"math"
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

	// Transparent background.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			m.Set(x, y, color.Transparent)
		}
	}

	// Vintage Americana: red 5-point star with white rays + striped C below.
	red := color.RGBA{R: 230, G: 48, B: 39, A: 255}    // #E63027
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255} // #FFFFFF
	black := color.RGBA{R: 0, G: 0, B: 0, A: 255}

	// Draw 5-point star at top.
	drawStar(m, size/2, size/2-size/8, size/3, red, black)

	// Draw stylized "C" below — candy-cane stripes.
	drawC(m, size/2, size/2+size/4, size/4, red, white, black)

	return m
}

// drawStar draws a 5-pointed star with a black outline.
func drawStar(m *image.RGBA, cx, cy, r int, fill, outline color.RGBA) {
	// Compute 10 points of a 5-point star.
	const pi = 3.14159265
	points := make([][2]int, 10)
	for i := 0; i < 10; i++ {
		angle := -pi/2 + float64(i)*pi/5
		rad := float64(r)
		if i%2 == 1 {
			rad = float64(r) * 0.4
		}
		points[i] = [2]int{
			cx + int(rad*cos(angle)),
			cy + int(rad*sin(angle)),
		}
	}
	// Fill with red.
	fillPolygon(m, points, fill)
	// Outline.
	drawPolygonOutline(m, points, outline, 1)
}

func cos(x float64) float64 {
	// Use math.Cos via a small lookup for hot loop; or import math.
	// For simplicity, use the math package below.
	return _cos(x)
}

func sin(x float64) float64 {
	return _sin(x)
}

var (
	_cos = func(x float64) float64 {
		// 1st-order Taylor for [-pi, pi] is not great; use builtin via init.
		return mathCos(x)
	}
	_sin = func(x float64) float64 {
		return mathSin(x)
	}
)

func mathCos(x float64) float64 {
	return math.Cos(x)
}
func mathSin(x float64) float64 {
	return math.Sin(x)
}

// drawC draws a stylized "C" shape made of red+white curved stripes.
func drawC(m *image.RGBA, cx, cy, r int, red, white, outline color.RGBA) {
	// C spans roughly 270° (from -135° to +135° in standard math coords).
	// We draw alternating stripes by sweeping arcs.
	const pi = 3.14159265
	startAngle := pi * 0.25  // 45° = lower right
	endAngle := pi * 1.75    // 315° = upper right (going counter-clockwise)
	_ = startAngle
	_ = endAngle

	// Simpler: just draw a thick C with 3 horizontal red stripes.
	// Inner: white ring; outer: red ring; with vertical black caps.
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			dist := x*x + y*y
			rr := r * r
			ir := (r - r/3) * (r - r/3)
			if dist > rr {
				continue
			}
			// Decide if this pixel is in the C-shape (not in the gap on the right).
			// The C opens to the right; gap spans roughly 60° around 0°.
			ang := atan2(float64(y), float64(x))
			// ang in [-pi, pi]; gap when -pi/6 < ang < pi/6
			if ang > -pi/6 && ang < pi/6 {
				// gap area — leave transparent
				continue
			}
			if dist > ir {
				// outer ring — red
				m.Set(cx+x, cy+y, red)
			} else {
				// inner — white (or red stripe)
				// Alternate stripes by angle: every 45° change color.
				stripe := int(ang/(pi/4)) % 2
				if stripe < 0 {
					stripe = -stripe
				}
				if stripe%2 == 0 {
					m.Set(cx+x, cy+y, red)
				} else {
					m.Set(cx+x, cy+y, white)
				}
			}
		}
	}
	// Black outline (thin ring at outer edge).
	for y := -r - 1; y <= r+1; y++ {
		for x := -r - 1; x <= r+1; x++ {
			dist := x*x + y*y
			rr := r * r
			ir := (r - r/3) * (r - r/3)
			if dist > rr && dist <= (r+1)*(r+1) {
				m.Set(cx+x, cy+y, outline)
			}
			if dist <= ir && dist > (r/3-1)*(r/3-1) {
				ang := atan2(float64(y), float64(x))
				if !(ang > -pi/6 && ang < pi/6) {
					m.Set(cx+x, cy+y, outline)
				}
			}
		}
	}
}

func atan2(y, x float64) float64 {
	return math.Atan2(y, x)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func fillPolygon(m *image.RGBA, points [][2]int, c color.RGBA) {
	bounds := m.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if pointInPolygon(x, y, points) {
				m.Set(x, y, c)
			}
		}
	}
}

func drawPolygonOutline(m *image.RGBA, points [][2]int, c color.RGBA, thickness int) {
	for i := 0; i < len(points); i++ {
		j := (i + 1) % len(points)
		drawLine(m, points[i][0], points[i][1], points[j][0], points[j][1], c, thickness)
	}
}

func drawLine(m *image.RGBA, x0, y0, x1, y1 int, c color.RGBA, thickness int) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 >= x1 {
		sx = -1
	}
	if y0 >= y1 {
		sy = -1
	}
	err := dx - dy
	for {
		for tx := -thickness / 2; tx <= thickness/2; tx++ {
			for ty := -thickness / 2; ty <= thickness/2; ty++ {
				px, py := x0+tx, y0+ty
				if px >= 0 && px < m.Bounds().Dx() && py >= 0 && py < m.Bounds().Dy() {
					m.Set(px, py, c)
				}
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func pointInPolygon(x, y int, poly [][2]int) bool {
	if len(poly) < 3 {
		return false
	}
	inside := false
	j := len(poly) - 1
	for i := 0; i < len(poly); i++ {
		xi, yi := poly[i][0], poly[i][1]
		xj, yj := poly[j][0], poly[j][1]
		if ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
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
