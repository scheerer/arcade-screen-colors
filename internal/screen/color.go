package screen

import (
	"image"
	"image/color"
	"math"
	"sort"
)

func AverageColor(img *image.RGBA, pixelGridSize int) color.RGBA {
	var sumR, sumG, sumB, sumA uint64
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y
	totalPixels := uint64(width * height)

	for y := 0; y < height; y += pixelGridSize {
		for x := 0; x < width; x += pixelGridSize {
			r, g, b, a := img.At(x, y).RGBA()
			sumR += uint64(r >> 8)
			sumG += uint64(g >> 8)
			sumB += uint64(b >> 8)
			sumA += uint64(a >> 8)
		}
	}

	return color.RGBA{
		R: uint8(sumR / totalPixels),
		G: uint8(sumG / totalPixels),
		B: uint8(sumB / totalPixels),
		A: uint8(sumA / totalPixels),
	}
}

// SquaredAverageColor calculates the squared average color of the image
func SquaredAverageColor(img *image.RGBA, pixelGridSize int) color.RGBA {
	var sumR, sumG, sumB, sumA uint64
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y
	totalPixels := uint64(width * height)

	for y := 0; y < height; y += pixelGridSize {
		for x := 0; x < width; x += pixelGridSize {
			r, g, b, a := img.At(x, y).RGBA()
			sumR += uint64((r >> 8) * (r >> 8))
			sumG += uint64((g >> 8) * (g >> 8))
			sumB += uint64((b >> 8) * (b >> 8))
			sumA += uint64((a >> 8) * (a >> 8))
		}
	}

	return color.RGBA{
		R: uint8(math.Sqrt(float64(sumR / totalPixels))),
		G: uint8(math.Sqrt(float64(sumG / totalPixels))),
		B: uint8(math.Sqrt(float64(sumB / totalPixels))),
		A: uint8(math.Sqrt(float64(sumA / totalPixels))),
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
			reds = append(reds, uint8(r>>8))
			greens = append(greens, uint8(g>>8))
			blues = append(blues, uint8(b>>8))
			alphas = append(alphas, uint8(a>>8))
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
