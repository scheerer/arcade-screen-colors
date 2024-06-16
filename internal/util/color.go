package util

import (
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/scheerer/arcade-screen-colors/internal/logging"
)

var logger = logging.New("util")

func RgbToHsb(r, g, b uint8) (uint16, uint16, uint16) {
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

func IsColorGreyish(saturation uint16) bool {
	satThreshold := float64(0xFFFF) * 0.2
	return float64(saturation) <= satThreshold
}

func AverageColor(img *image.RGBA, pixelGridSize int) color.RGBA {
	var sumR, sumG, sumB, sumA, totalPixels uint64
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	for y := 0; y < height; y += pixelGridSize {
		for x := 0; x < width; x += pixelGridSize {
			totalPixels++
			r, g, b, a := img.At(x, y).RGBA()
			sumR += uint64(r)
			sumG += uint64(g)
			sumB += uint64(b)
			sumA += uint64(a)
		}
	}

	return color.RGBA{
		R: uint8(float64(sumR/totalPixels) / 0xFFFF * 0xFF),
		G: uint8(float64(sumG/totalPixels) / 0xFFFF * 0xFF),
		B: uint8(float64(sumB/totalPixels) / 0xFFFF * 0xFF),
		A: uint8(float64(sumA/totalPixels) / 0xFFFF * 0xFF),
	}
}

// SquaredAverageColor calculates the squared average color of the image
func SquaredAverageColor(img *image.RGBA, pixelGridSize int) color.RGBA {
	var sumR, sumG, sumB, sumA, totalPixels uint64
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	for y := 0; y < height; y += pixelGridSize {
		for x := 0; x < width; x += pixelGridSize {
			totalPixels++

			r, g, b, a := img.At(x, y).RGBA()
			sumR += uint64(r) * uint64(r)
			sumG += uint64(g) * uint64(g)
			sumB += uint64(b) * uint64(b)
			sumA += uint64(a) * uint64(a)
		}
	}

	return color.RGBA{
		R: uint8(math.Sqrt(float64(sumR/totalPixels)) / 0xFFFF * 0xFF),
		G: uint8(math.Sqrt(float64(sumG/totalPixels)) / 0xFFFF * 0xFF),
		B: uint8(math.Sqrt(float64(sumB/totalPixels)) / 0xFFFF * 0xFF),
		A: uint8(math.Sqrt(float64(sumA/totalPixels)) / 0xFFFF * 0xFF),
	}
}

// MedianColor calculates the median color of the image
func MedianColor(img *image.RGBA, pixelGridSize int) color.RGBA {
	var reds, greens, blues, alphas []uint8
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	for y := 0; y < height; y += pixelGridSize {
		for x := 0; x < width; x += pixelGridSize {
			r, g, b, a := img.At(x, y).RGBA()
			reds = append(reds, uint8(float64(r)/0xFFFF*0xFF))
			greens = append(greens, uint8(float64(g)/0xFFFF*0xFF))
			blues = append(blues, uint8(float64(b)/0xFFFF*0xFF))
			alphas = append(alphas, uint8(float64(a)/0xFFFF*0xFF))
		}
	}

	sort.Slice(reds, func(i, j int) bool { return reds[i] < reds[j] })
	sort.Slice(greens, func(i, j int) bool { return greens[i] < greens[j] })
	sort.Slice(blues, func(i, j int) bool { return blues[i] < blues[j] })
	sort.Slice(alphas, func(i, j int) bool { return alphas[i] < alphas[j] })

	median := func(values []uint8) uint8 {
		n := len(values)
		if n%2 == 0 {
			return uint8((int(values[n/2-1]) + int(values[n/2])) / 2)
		}
		return values[n/2]
	}

	return color.RGBA{
		R: median(reds),
		G: median(greens),
		B: median(blues),
		A: median(alphas),
	}
}

// ModeColor calculates the mode color of the image
func ModeColor(img *image.RGBA, pixelGridSize int) color.RGBA {
	colorCount := make(map[color.RGBA]int)
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	for y := 0; y < height; y += pixelGridSize {
		for x := 0; x < width; x += pixelGridSize {
			c := img.At(x, y).(color.RGBA)
			colorCount[c]++
		}
	}

	var modeColor color.RGBA
	maxCount := 0
	for c, count := range colorCount {
		if count > maxCount {
			maxCount = count
			modeColor = c
		}
	}

	return modeColor
}
