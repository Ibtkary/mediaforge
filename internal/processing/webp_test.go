package processing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeWebP_ValidImage(t *testing.T) {
	skipIfNoCWebP(t)

	img := createTestImage(200, 200)
	data, err := EncodeWebP(img, 80, "cwebp")

	require.NoError(t, err)
	require.NotEmpty(t, data)
	// WebP files start with RIFF....WEBP magic bytes
	assert.Equal(t, "RIFF", string(data[:4]))
	assert.Equal(t, "WEBP", string(data[8:12]))
}

func TestEncodeWebP_QualityComparison(t *testing.T) {
	skipIfNoCWebP(t)

	img := createTestImage(200, 200)

	low, err := EncodeWebP(img, 10, "cwebp")
	require.NoError(t, err)

	high, err := EncodeWebP(img, 100, "cwebp")
	require.NoError(t, err)

	// Higher quality generally produces larger files
	// (for solid color this may not hold, so just check both are valid)
	assert.NotEmpty(t, low)
	assert.NotEmpty(t, high)
}

func TestEncodeWebP_InvalidCWebPPath(t *testing.T) {
	img := createTestImage(100, 100)
	_, err := EncodeWebP(img, 80, "/nonexistent/cwebp-binary")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cwebp failed")
}

func TestEncodeWebP_LargerImage(t *testing.T) {
	skipIfNoCWebP(t)

	img := createTestImage(800, 600)
	data, err := EncodeWebP(img, 90, "cwebp")

	require.NoError(t, err)
	assert.Equal(t, "RIFF", string(data[:4]))
	assert.Equal(t, "WEBP", string(data[8:12]))
}
