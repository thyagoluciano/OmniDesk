package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// GenerateIconBytes creates a clean 64x64 PNG icon representing Crossover.
func GenerateIconBytes() []byte {
	const size = 64
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Indigo/Blue background circle
	bgColor := color.RGBA{R: 59, G: 130, B: 246, A: 255}
	accentColor := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	radiusSq := float64((size/2 - 3) * (size/2 - 3))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x - size/2)
			dy := float64(y - size/2)
			if dx*dx+dy*dy <= radiusSq {
				img.Set(x, y, bgColor)
			}
		}
	}

	// Draw white intersecting arrows / cross symbol
	for i := 18; i <= 46; i++ {
		for w := -2; w <= 2; w++ {
			// Main diagonal
			img.Set(i, i+w, accentColor)
			// Anti diagonal
			img.Set(i, 64-i+w, accentColor)
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
