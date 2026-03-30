package mediaforge

import (
	"context"
	"errors"
	"image"
	"io"
	"time"
)

// ObjectStorage abstracts upload/delete/exists operations for any storage provider.
// Implementations: Bunny.net, AWS S3, Cloudflare R2, MinIO, etc.
type ObjectStorage interface {
	// Upload stores data at the given key.
	// The caller owns the reader; implementations must not close it.
	Upload(ctx context.Context, key string, r io.Reader, size int64) error

	// Delete removes the object at key.
	// Implementations should return nil if the object does not exist.
	Delete(ctx context.Context, key string) error

	// Exists checks whether an object exists at key.
	Exists(ctx context.Context, key string) (bool, error)
}

// URLSigner generates authenticated/signed URLs for stored objects.
// Implementations: Bunny CDN token, S3 presigned URLs, Cloudflare signed URLs.
type URLSigner interface {
	// SignURL returns a time-limited URL for the given storage key.
	SignURL(key string, expires time.Time) (string, error)
}

// ImageEncoder converts an image to the target output format.
// Implementations: CWebPEncoder (external binary), pure Go encoder, passthrough, etc.
type ImageEncoder interface {
	// Encode converts img to the target format at the given quality (1-100).
	Encode(img image.Image, quality int) ([]byte, error)

	// Extension returns the file extension including the dot (e.g., ".webp").
	Extension() string

	// ContentType returns the MIME type (e.g., "image/webp").
	ContentType() string
}

// RetryableError is an optional interface that storage errors can implement
// to indicate whether the operation should be retried.
type RetryableError interface {
	IsRetryable() bool
}

// isRetryableError checks if err implements RetryableError and returns true.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	var re RetryableError
	if errors.As(err, &re) {
		return re.IsRetryable()
	}
	return false
}
