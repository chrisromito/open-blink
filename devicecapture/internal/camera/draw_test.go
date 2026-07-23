package camera

import (
	"image"
	"image/color"
	"testing"

	"devicecapture/internal/domain/detection"
	"github.com/stretchr/testify/assert"
)

// createTestImage creates a simple test image
func createTestImage(width, height int) *image.RGBA {
	return image.NewRGBA(image.Rect(0, 0, width, height))
}

// createTestDetection creates a test detection for testing purposes
func createTestDetection(label string, x1, y1, x2, y2 float64) detection.Detection {
	return detection.Detection{
		Label: label,
		Bbox: detection.BBox{
			X1: x1,
			Y1: y1,
			X2: x2,
			Y2: y2,
		},
		Confidence: 0.9,
	}
}

func Test_Draw_Bounding_Box(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		minX   int
		minY   int
		maxX   int
		maxY   int
		color  color.Color
	}{
		{
			name:   "simple rectangle",
			width:  100,
			height: 100,
			minX:   10,
			minY:   10,
			maxX:   50,
			maxY:   50,
			color:  color.RGBA{R: 255, A: 255}, // Red
		},
		{
			name:   "edge rectangle",
			width:  50,
			height: 50,
			minX:   0,
			minY:   0,
			maxX:   49,
			maxY:   49,
			color:  color.RGBA{G: 255, A: 255}, // Green
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			img := createTestImage(test.width, test.height)

			DrawBoundingBox(img, test.minX, test.minY, test.maxX, test.maxY, test.color)

			// Check corners are drawn
			a.Equal(test.color, img.At(test.minX, test.minY), "top-left corner should be colored")
			a.Equal(test.color, img.At(test.maxX, test.minY), "top-right corner should be colored")
			a.Equal(
				test.color,
				img.At(test.minX, test.maxY),
				"bottom-left corner should be colored",
			)
			a.Equal(
				test.color,
				img.At(test.maxX, test.maxY),
				"bottom-right corner should be colored",
			)

			// Check that interior is not modified (assuming it was transparent)
			if test.maxX > test.minX+1 && test.maxY > test.minY+1 {
				interiorColor := img.At(test.minX+1, test.minY+1)
				a.NotEqual(test.color, interiorColor, "interior should not be colored")
			}
		})
	}
}

func Test_Draw_BBox(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		bbox   detection.BBox
		color  color.Color
		msg    string
	}{
		{
			name:   "valid bbox",
			width:  100,
			height: 100,
			bbox:   detection.BBox{X1: 10, Y1: 10, X2: 50, Y2: 50},
			color:  color.RGBA{R: 255, A: 255},
			msg:    "should draw bbox correctly",
		},
		{
			name:   "small bbox",
			width:  50,
			height: 50,
			bbox:   detection.BBox{X1: 5, Y1: 5, X2: 15, Y2: 15},
			color:  color.RGBA{B: 255, A: 255},
			msg:    "should handle small bboxes",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			originalImg := createTestImage(test.width, test.height)

			result := DrawBBox(originalImg, test.bbox, test.color)

			a.NotNil(result, test.msg)
			a.Equal(test.width, result.Bounds().Dx(), "width should be preserved")
			a.Equal(test.height, result.Bounds().Dy(), "height should be preserved")

			// Check that bbox corners are colored
			a.Equal(
				test.color,
				result.At(int(test.bbox.X1), int(test.bbox.Y1)),
				"bbox should be drawn",
			)
		})
	}
}

func Test_Draw_Detections(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		detections []detection.Detection
		color      color.Color
		wantEmpty  bool
		msg        string
	}{
		{
			name:       "single detection",
			width:      100,
			height:     100,
			detections: []detection.Detection{createTestDetection("dog", 10, 10, 50, 50)},
			wantEmpty:  false,
			msg:        "should draw single detection",
		},
		{
			name:   "multiple detections",
			width:  200,
			height: 200,
			detections: []detection.Detection{
				createTestDetection("cat", 10, 10, 50, 50),
				createTestDetection("person", 60, 60, 100, 100),
			},
			wantEmpty: false,
			msg:       "should draw multiple detections",
		},
		{
			name:       "no detections",
			width:      100,
			height:     100,
			detections: []detection.Detection{},
			wantEmpty:  true,
			msg:        "should handle empty detection slice",
		},
		{
			name:       "nil detections",
			width:      100,
			height:     100,
			detections: nil,
			wantEmpty:  true,
			msg:        "should handle nil detections",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			originalImg := createTestImage(test.width, test.height)

			result := DrawDetections(originalImg, test.detections)

			a.NotNil(result, test.msg)
			a.Equal(test.width, result.Bounds().Dx(), "width should be preserved")
			a.Equal(test.height, result.Bounds().Dy(), "height should be preserved")
		})
	}
}

func Test_Draw_Label(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
		x      int
		y      int
		label  string
		msg    string
	}{
		{
			name:   "simple label",
			width:  100,
			height: 100,
			x:      10,
			y:      20,
			label:  "dog",
			msg:    "should draw label without error",
		},
		{
			name:   "empty label",
			width:  50,
			height: 50,
			x:      5,
			y:      10,
			label:  "",
			msg:    "should handle empty label",
		},
		{
			name:   "long label",
			width:  200,
			height: 100,
			x:      10,
			y:      30,
			label:  "very_long_detection_label",
			msg:    "should handle long labels",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := assert.New(t)
			img := createTestImage(test.width, test.height)

			// DrawLabel should not panic
			a.NotPanics(func() {
				DrawLabel(img, test.x, test.y, test.label, color.RGBA{R: 255, A: 255})
			}, test.msg)

			// The function modifies the image, so it should still be valid
			a.NotNil(img, "image should remain valid after drawing label")
		})
	}
}

func Test_Draw_Functions_With_InvalidInputs(t *testing.T) {
	t.Run("out_of_bounds_bounding_box", func(t *testing.T) {
		a := assert.New(t)
		img := createTestImage(50, 50)

		// This should not panic even with out-of-bounds coordinates
		a.NotPanics(func() {
			DrawBoundingBox(img, 100, 100, 200, 200, color.RGBA{R: 255, A: 255})
		}, "should handle out-of-bounds coordinates gracefully")
	})

	t.Run("negative_coordinates", func(t *testing.T) {
		a := assert.New(t)
		img := createTestImage(50, 50)

		a.NotPanics(func() {
			DrawBoundingBox(img, -10, -10, 10, 10, color.RGBA{G: 255, A: 255})
		}, "should handle negative coordinates")
	})
}
