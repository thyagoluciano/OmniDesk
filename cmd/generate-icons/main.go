package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
)

func main() {
	const size = 256
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Clear with transparent
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Transparent}, image.Point{}, draw.Src)

	// Draw rounded square background
	cornerRad := 54.0
	rectMin := 16.0
	rectMax := float64(size - 16)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx := float64(x)
			fy := float64(y)

			if fx < rectMin || fx > rectMax || fy < rectMin || fy > rectMax {
				continue
			}

			// Check rounded corners
			inside := true
			// Top-Left
			if fx < rectMin+cornerRad && fy < rectMin+cornerRad {
				dx := fx - (rectMin + cornerRad)
				dy := fy - (rectMin + cornerRad)
				if dx*dx+dy*dy > cornerRad*cornerRad {
					inside = false
				}
			}
			// Top-Right
			if fx > rectMax-cornerRad && fy < rectMin+cornerRad {
				dx := fx - (rectMax - cornerRad)
				dy := fy - (rectMin + cornerRad)
				if dx*dx+dy*dy > cornerRad*cornerRad {
					inside = false
				}
			}
			// Bottom-Left
			if fx < rectMin+cornerRad && fy > rectMax-cornerRad {
				dx := fx - (rectMin + cornerRad)
				dy := fy - (rectMax - cornerRad)
				if dx*dx+dy*dy > cornerRad*cornerRad {
					inside = false
				}
			}
			// Bottom-Right
			if fx > rectMax-cornerRad && fy > rectMax-cornerRad {
				dx := fx - (rectMax - cornerRad)
				dy := fy - (rectMax - cornerRad)
				if dx*dx+dy*dy > cornerRad*cornerRad {
					inside = false
				}
			}

			if inside {
				// Gradient from #0284c7 (2, 132, 199) to #06b6d4 (6, 182, 212)
				t := (fx + fy) / (2 * size)
				r := uint8(float64(2)*(1-t) + float64(6)*t)
				g := uint8(float64(132)*(1-t) + float64(182)*t)
				b := uint8(float64(199)*(1-t) + float64(212)*t)
				img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}

	// Draw crisp intersecting arrows / omnidesk cross
	white := color.RGBA{255, 255, 255, 255}
	strokeWidth := 10.0

	// Line 1: (72, 184) to (184, 72)
	drawLine(img, 72, 184, 184, 72, strokeWidth, white)
	// Arrow head 1 at (184, 72): from (140, 72) to (184, 72) to (184, 116)
	drawLine(img, 140, 72, 184, 72, strokeWidth, white)
	drawLine(img, 184, 72, 184, 116, strokeWidth, white)

	// Line 2: (72, 72) to (184, 184)
	drawLine(img, 72, 72, 184, 184, strokeWidth, white)
	// Arrow head 2 at (184, 184): from (140, 184) to (184, 184) to (184, 140)
	drawLine(img, 140, 184, 184, 184, strokeWidth, white)
	drawLine(img, 184, 184, 184, 140, strokeWidth, white)

	f, err := os.Create("assets/omnidesk.png")
	if err != nil {
		log.Fatalf("Cannot create file: %v", err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		log.Fatalf("Cannot encode PNG: %v", err)
	}

	log.Println("assets/omnidesk.png generated successfully.")
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 float64, width float64, c color.Color) {
	dx := x1 - x0
	dy := y1 - y0
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	ux := dx / length
	uy := dy / length

	halfW := width / 2.0
	minX := int(math.Max(0, math.Min(x0, x1)-width))
	maxX := int(math.Min(float64(img.Bounds().Dx()-1), math.Max(x0, x1)+width))
	minY := int(math.Max(0, math.Min(y0, y1)-width))
	maxY := int(math.Min(float64(img.Bounds().Dy()-1), math.Max(y0, y1)+width))

	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := float64(x) - x0
			py := float64(y) - y0
			proj := px*ux + py*uy
			if proj >= 0 && proj <= length {
				perp := math.Abs(px*(-uy) + py*ux)
				if perp <= halfW {
					img.Set(x, y, c)
				}
			}
		}
	}
}
