package processing

import "image"

// CWebPEncoder encodes images to WebP using the cwebp binary.
type CWebPEncoder struct {
	BinaryPath string
}

// Encode converts img to WebP format using the cwebp binary.
func (e *CWebPEncoder) Encode(img image.Image, quality int) ([]byte, error) {
	return EncodeWebP(img, quality, e.BinaryPath)
}

// Extension returns ".webp".
func (e *CWebPEncoder) Extension() string { return ".webp" }

// ContentType returns "image/webp".
func (e *CWebPEncoder) ContentType() string { return "image/webp" }
