package lights

import "math"

func rgbToHsb(r, g, b uint8) (uint16, uint16, uint16) {
	red := float64(r) / 255.0
	green := float64(g) / 255.0
	blue := float64(b) / 255.0

	max := math.Max(red, math.Max(green, blue))
	min := math.Min(red, math.Min(green, blue))
	delta := max - min

	var h, s, v float64
	v = max // Brightness is the max of RGB

	if delta == 0 {
		h = 0
		s = 0
	} else { // Chromatic data...
		s = delta / max // Saturation is degree of variation from grey.

		deltaR := (((max - red) / 6) + (delta / 2)) / delta
		deltaG := (((max - green) / 6) + (delta / 2)) / delta
		deltaB := (((max - blue) / 6) + (delta / 2)) / delta

		if red == max {
			h = deltaB - deltaG // Color is closer to red
		} else if green == max {
			h = (1.0 / 3.0) + deltaR - deltaB // Color is closer to green
		} else if blue == max {
			h = (2.0 / 3.0) + deltaG - deltaR // Color is closer to blue
		}

		if h < 0 {
			h += 1
		}
		if h > 1 {
			h -= 1
		}
	}

	// Convert hue to degrees and then to uint16
	hue := uint16(math.Round(h * 0xFFFF))
	saturation := uint16(math.Round(s * 0xFFFF))
	brightness := uint16(math.Round(v * 0xFFFF))

	return hue, saturation, brightness
}
