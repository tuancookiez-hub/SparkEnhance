// Package placement computes the floating bar's screen position.
package placement

import "github.com/tuancookiez-hub/SparkEnhance/internal/platform"

// Bar returns the (x, y) top-left corner to centre the bar
// at the bottom of the primary monitor.
func Bar(mon platform.Monitor, w, h int) (int, int) {
	x := mon.X + (mon.Width-w)/2
	y := mon.Y + mon.Height - h - 24
	return x, y
}

// LimitSize ensures the bar doesn't exceed monitor bounds or a max height ratio.
func LimitSize(mon platform.Monitor, w, h, maxW, maxH int, maxHRatio float64) (int, int) {
	if w > mon.Width {
		w = mon.Width
	}
	maxHActual := int(float64(mon.Height) * maxHRatio)
	if h > maxHActual {
		h = maxHActual
	}
	return w, h
}
