package processing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCWebPEncoder_Extension(t *testing.T) {
	enc := &CWebPEncoder{BinaryPath: "cwebp"}
	assert.Equal(t, ".webp", enc.Extension())
}

func TestCWebPEncoder_ContentType(t *testing.T) {
	enc := &CWebPEncoder{BinaryPath: "cwebp"}
	assert.Equal(t, "image/webp", enc.ContentType())
}

func TestCWebPEncoder_Encode(t *testing.T) {
	skipIfNoCWebP(t)

	enc := &CWebPEncoder{BinaryPath: "cwebp"}
	img := createTestImage(100, 100)

	data, err := enc.Encode(img, 80)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestCWebPEncoder_Encode_InvalidPath(t *testing.T) {
	enc := &CWebPEncoder{BinaryPath: "/nonexistent/cwebp"}
	img := createTestImage(100, 100)

	_, err := enc.Encode(img, 80)
	assert.Error(t, err)
}
