package processing

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os/exec"
	"testing"
)

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, red)
		}
	}
	return img
}

func createTestJPEG(width, height int) []byte {
	img := createTestImage(width, height)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		panic("failed to create test JPEG: " + err.Error())
	}
	return buf.Bytes()
}

func createTestPNG(width, height int) []byte {
	img := createTestImage(width, height)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic("failed to create test PNG: " + err.Error())
	}
	return buf.Bytes()
}

// createTruncatedJPEG creates a JPEG with valid headers but truncated scan data.
// image.DecodeConfig succeeds (reads SOF header for dimensions).
// image.Decode fails (scan data is incomplete).
func createTruncatedJPEG(width, height int) []byte {
	full := createTestJPEG(width, height)
	return full[:len(full)*6/10] // keep ~60%: all headers, truncate scan data
}

func skipIfNoCWebP(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cwebp"); err != nil {
		t.Skip("cwebp not found, skipping test")
	}
}
