package processing

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResizeImage_DownscaleWide(t *testing.T) {
	img := createTestImage(1000, 500)
	result := ResizeImage(img, 400, 400)

	assert.Equal(t, 400, result.Bounds().Dx())
	assert.Equal(t, 200, result.Bounds().Dy())
}

func TestResizeImage_DownscaleTall(t *testing.T) {
	img := createTestImage(500, 1000)
	result := ResizeImage(img, 400, 400)

	assert.Equal(t, 200, result.Bounds().Dx())
	assert.Equal(t, 400, result.Bounds().Dy())
}

func TestResizeImage_DownscaleSquare(t *testing.T) {
	img := createTestImage(800, 800)
	result := ResizeImage(img, 400, 400)

	assert.Equal(t, 400, result.Bounds().Dx())
	assert.Equal(t, 400, result.Bounds().Dy())
}

func TestResizeImage_NoUpscale(t *testing.T) {
	img := createTestImage(200, 150)
	result := ResizeImage(img, 400, 400)

	assert.Equal(t, 200, result.Bounds().Dx())
	assert.Equal(t, 150, result.Bounds().Dy())
}

func TestResizeImage_ExactMatch(t *testing.T) {
	img := createTestImage(400, 400)
	result := ResizeImage(img, 400, 400)

	assert.Equal(t, 400, result.Bounds().Dx())
	assert.Equal(t, 400, result.Bounds().Dy())
}

func TestResizeImage_ReturnsNRGBA(t *testing.T) {
	img := createTestImage(200, 200)
	result := ResizeImage(img, 100, 100)

	require.NotNil(t, result)
	_, ok := interface{}(result).(*image.NRGBA)
	assert.True(t, ok, "result should be *image.NRGBA")
}

func TestResizeImage_AspectRatioPreserved(t *testing.T) {
	img := createTestImage(1600, 900) // 16:9
	result := ResizeImage(img, 800, 800)

	w := result.Bounds().Dx()
	h := result.Bounds().Dy()
	assert.Equal(t, 800, w)
	assert.Equal(t, 450, h)

	// Verify aspect ratio is maintained
	origRatio := 1600.0 / 900.0
	newRatio := float64(w) / float64(h)
	assert.InDelta(t, origRatio, newRatio, 0.02)
}

func TestResizeImage_VerySmallResult(t *testing.T) {
	img := createTestImage(10000, 1)
	result := ResizeImage(img, 10, 10)

	assert.GreaterOrEqual(t, result.Bounds().Dx(), 1)
	assert.GreaterOrEqual(t, result.Bounds().Dy(), 1)
}
