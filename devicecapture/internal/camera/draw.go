package camera

import (
	"devicecapture/internal/domain/detection"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	"image"
	"image/color"
)

func DrawDetections(img image.Image, detections []detection.Detection, c color.Color) *image.RGBA {
	// Convert image to RGBA (so we can modify it)
	bounds := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	// Copy the original image onto the new RGBA surface
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	for _, d := range detections {
		bbox := d.Bbox
		DrawBoundingBox(rgba, int(bbox.X1), int(bbox.Y1), int(bbox.X2), int(bbox.Y2), c)
		// Put the label at the x-min (left) & "below" the Y max (ymax + 8)
		DrawLabel(rgba, int(bbox.X1), int(bbox.Y1-8), d.Label)
	}
	return rgba
}

// DrawBBox draws a rectangle [detection.BBox] on [image.Image] with [color.Color]
func DrawBBox(img image.Image, bbox detection.BBox, c color.Color) *image.RGBA {
	// Convert image to RGBA (so we can modify it)
	bounds := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	// Copy the original image onto the new RGBA surface
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	// Now we can draw the bbox
	DrawBoundingBox(rgba, int(bbox.X1), int(bbox.Y1), int(bbox.X2), int(bbox.Y2), c)
	return rgba
}

// DrawBoundingBox draws a hollow rectangle on a mutable image
func DrawBoundingBox(img *image.RGBA, minX, minY, maxX, maxY int, c color.Color) {
	// Draw horizontal lines
	for x := minX; x <= maxX; x++ {
		img.Set(x, minY, c)
		img.Set(x, maxY, c)
	}
	// Draw vertical lines
	for y := minY; y <= maxY; y++ {
		img.Set(minX, y, c)
		img.Set(maxX, y, c)
	}
}

func DrawLabel(img *image.RGBA, x int, y int, label string) {
	col := color.RGBA{R: 200, G: 100, A: 255}
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(label)
}
