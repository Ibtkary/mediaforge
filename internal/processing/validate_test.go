package processing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultValidateCfg() ValidateConfig {
	return ValidateConfig{
		MaxFileSize:      10_485_760,
		AllowedMIMETypes: []string{"image/jpeg", "image/png", "image/webp"},
		MinWidth:         10,
		MinHeight:        10,
		MaxWidth:         8000,
		MaxHeight:        8000,
	}
}

func TestValidateImage_ValidJPEG(t *testing.T) {
	data := createTestJPEG(200, 150)
	info, err := ValidateImage(data, defaultValidateCfg())

	require.NoError(t, err)
	assert.Equal(t, 200, info.Width)
	assert.Equal(t, 150, info.Height)
	assert.Equal(t, "image/jpeg", info.MIME)
	assert.Equal(t, "jpeg", info.Format)
}

func TestValidateImage_ValidPNG(t *testing.T) {
	data := createTestPNG(300, 200)
	info, err := ValidateImage(data, defaultValidateCfg())

	require.NoError(t, err)
	assert.Equal(t, 300, info.Width)
	assert.Equal(t, 200, info.Height)
	assert.Equal(t, "image/png", info.MIME)
	assert.Equal(t, "png", info.Format)
}

func TestValidateImage_FileTooLarge(t *testing.T) {
	data := createTestJPEG(200, 200)
	cfg := defaultValidateCfg()
	cfg.MaxFileSize = 100 // tiny limit

	_, err := ValidateImage(data, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

func TestValidateImage_DisallowedMIME(t *testing.T) {
	data := createTestPNG(200, 200)
	cfg := defaultValidateCfg()
	cfg.AllowedMIMETypes = []string{"image/jpeg"} // PNG not allowed

	_, err := ValidateImage(data, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestValidateImage_TooSmall(t *testing.T) {
	data := createTestJPEG(5, 5) // below min 10x10
	cfg := defaultValidateCfg()

	_, err := ValidateImage(data, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "below minimum")
}

func TestValidateImage_TooLarge(t *testing.T) {
	data := createTestJPEG(200, 200)
	cfg := defaultValidateCfg()
	cfg.MaxWidth = 100
	cfg.MaxHeight = 100

	_, err := ValidateImage(data, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceed maximum")
}

func TestValidateImage_CorruptData(t *testing.T) {
	data := []byte("not an image at all, just random text data here")
	cfg := defaultValidateCfg()
	cfg.AllowedMIMETypes = []string{"text/plain; charset=utf-8"} // allow the detected MIME

	_, err := ValidateImage(data, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode")
}

func TestValidateImage_NonImageData(t *testing.T) {
	data := []byte(`{"key": "value"}`)
	cfg := defaultValidateCfg()

	_, err := ValidateImage(data, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestValidateImage_ExactMinBoundary(t *testing.T) {
	data := createTestJPEG(10, 10)
	cfg := defaultValidateCfg()

	info, err := ValidateImage(data, cfg)
	require.NoError(t, err)
	assert.Equal(t, 10, info.Width)
	assert.Equal(t, 10, info.Height)
}

func TestValidateImage_ExactMaxBoundary(t *testing.T) {
	data := createTestJPEG(100, 100)
	cfg := defaultValidateCfg()
	cfg.MaxWidth = 100
	cfg.MaxHeight = 100

	info, err := ValidateImage(data, cfg)
	require.NoError(t, err)
	assert.Equal(t, 100, info.Width)
	assert.Equal(t, 100, info.Height)
}
