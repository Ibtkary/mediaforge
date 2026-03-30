package processing

import "image"

// Encoder converts an image to an output format.
// Implementations: CWebPEncoder (external binary), pure Go WebP, etc.
type Encoder interface {
	// Encode converts img to the target format at the given quality (1-100).
	Encode(img image.Image, quality int) ([]byte, error)

	// Extension returns the file extension including the dot (e.g., ".webp").
	Extension() string

	// ContentType returns the MIME type (e.g., "image/webp").
	ContentType() string
}
