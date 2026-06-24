package camera

import (
	"fmt"
	"image"
	"image/color"

	"devicecapture/internal/domain/detection"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var colorMap = map[string]color.RGBA{
	"person": {R: 255, A: 255},
	"car":    {G: 255, A: 255},
	"truck":  {G: 255, A: 255},
}

var defaultColor = color.RGBA{B: 255, A: 255}

func getLabelColor(label string) color.RGBA {
	value := colorMap[label]
	if value.A == 0 {
		return defaultColor
	}
	return value
}

func DrawDetections(img image.Image, detections []detection.Detection) *image.RGBA {
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
		c := getLabelColor(d.Label)
		DrawBoundingBox(rgba, int(bbox.X1), int(bbox.Y1), int(bbox.X2), int(bbox.Y2), c)
		// Put the label at the x-min (left) & "below" the Y max (ymax + 8)
		txt := fmt.Sprintf("%s %d%%", d.Label, int(d.Confidence*100))
		DrawLabel(rgba, int(bbox.X1), int(bbox.Y1-8), txt, c)
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

func DrawLabel(img *image.RGBA, x int, y int, label string, c color.Color) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(c),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(label)
}
