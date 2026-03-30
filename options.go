package mediaforge

import (
	"log/slog"
	"time"
)

// Option configures the Client via functional options.
type Option func(*clientOptions)

// clientOptions holds all optional configuration for the Client.
type clientOptions struct {
	encoder        ImageEncoder
	clock          Clock
	logger         *slog.Logger
	hook           Hook
	variants       []VariantConfig
	maxFileSize    int64
	allowedMIMEs   []string
	minWidth       int
	minHeight      int
	maxWidth       int
	maxHeight      int
	uploadTimeout  time.Duration
	deleteTimeout  time.Duration
	tokenTTL       time.Duration
	pendingTTL     time.Duration
	maxRetries     *int
	retryBaseDelay time.Duration
	cwebpPath        string
	dedup            *bool
	lazyVariants     *bool
	smartVariantSkip *bool
}

// WithEncoder sets a custom image encoder (e.g., pure Go WebP, AVIF).
// If not set, the default CWebP binary encoder is used.
func WithEncoder(e ImageEncoder) Option {
	return func(o *clientOptions) { o.encoder = e }
}

// WithClock sets a custom clock for time operations.
// Useful for deterministic testing. If not set, real system time is used.
func WithClock(c Clock) Option {
	return func(o *clientOptions) { o.clock = c }
}

// WithVariants sets the image variant configurations.
// If not set, DefaultVariants() is used (thumb, medium, large).
func WithVariants(v []VariantConfig) Option {
	return func(o *clientOptions) { o.variants = v }
}

// WithMaxFileSize sets the maximum allowed file size in bytes.
// Default: 10MB.
func WithMaxFileSize(n int64) Option {
	return func(o *clientOptions) { o.maxFileSize = n }
}

// WithAllowedMIMETypes sets the allowed image MIME types.
// Default: image/jpeg, image/png, image/webp.
func WithAllowedMIMETypes(types []string) Option {
	return func(o *clientOptions) { o.allowedMIMEs = types }
}

// WithMinDimensions sets the minimum allowed image dimensions.
// Default: 100x100.
func WithMinDimensions(w, h int) Option {
	return func(o *clientOptions) { o.minWidth = w; o.minHeight = h }
}

// WithMaxDimensions sets the maximum allowed image dimensions.
// Default: 8000x8000.
func WithMaxDimensions(w, h int) Option {
	return func(o *clientOptions) { o.maxWidth = w; o.maxHeight = h }
}

// WithRetries sets the maximum number of retries and the base delay for exponential backoff.
// Default: 3 retries, 1s base delay.
func WithRetries(n int, baseDelay time.Duration) Option {
	return func(o *clientOptions) { o.maxRetries = &n; o.retryBaseDelay = baseDelay }
}

// WithUploadTimeout sets the per-file upload timeout.
// Default: 60s.
func WithUploadTimeout(d time.Duration) Option {
	return func(o *clientOptions) { o.uploadTimeout = d }
}

// WithDeleteTimeout sets the per-file delete timeout.
// Default: 10s.
func WithDeleteTimeout(d time.Duration) Option {
	return func(o *clientOptions) { o.deleteTimeout = d }
}

// WithTokenTTL sets the default TTL for signed URLs.
// Default: 1 hour.
func WithTokenTTL(d time.Duration) Option {
	return func(o *clientOptions) { o.tokenTTL = d }
}

// WithPendingTTL sets the TTL for pending (uncommitted) assets.
// Default: 2 hours.
func WithPendingTTL(d time.Duration) Option {
	return func(o *clientOptions) { o.pendingTTL = d }
}

// WithDedup enables content-hash deduplication.
// When enabled, before uploading, the client checks if a file with the same
// content hash already exists. If so, it skips the upload and returns existing metadata.
// Default: false.
func WithDedup(enabled bool) Option {
	return func(o *clientOptions) { o.dedup = &enabled }
}

// WithLazyVariants enables lazy variant generation.
// When enabled, only the original image is uploaded. Variants can be generated
// later via GenerateVariants().
// Default: false.
func WithLazyVariants(enabled bool) Option {
	return func(o *clientOptions) { o.lazyVariants = &enabled }
}

// WithSmartVariantSkip enables smart variant skipping.
// When enabled, variants are skipped if the source image is already smaller than
// the variant's max dimensions (since the result would be identical to the original).
// Default: false.
func WithSmartVariantSkip(enabled bool) Option {
	return func(o *clientOptions) { o.smartVariantSkip = &enabled }
}

// WithCWebPPath sets the path to the cwebp binary.
// Only relevant when using the default CWebP encoder.
// Default: "cwebp" (found via PATH).
func WithCWebPPath(path string) Option {
	return func(o *clientOptions) { o.cwebpPath = path }
}

// applyToConfig merges non-zero option values into a Config.
func (o *clientOptions) applyToConfig(cfg *Config) {
	if len(o.variants) > 0 {
		cfg.Variants = o.variants
	}
	if o.maxFileSize > 0 {
		cfg.MaxFileSize = o.maxFileSize
	}
	if len(o.allowedMIMEs) > 0 {
		cfg.AllowedMIMETypes = o.allowedMIMEs
	}
	if o.minWidth > 0 {
		cfg.MinWidth = o.minWidth
	}
	if o.minHeight > 0 {
		cfg.MinHeight = o.minHeight
	}
	if o.maxWidth > 0 {
		cfg.MaxWidth = o.maxWidth
	}
	if o.maxHeight > 0 {
		cfg.MaxHeight = o.maxHeight
	}
	if o.uploadTimeout > 0 {
		cfg.UploadTimeout = o.uploadTimeout
	}
	if o.deleteTimeout > 0 {
		cfg.DeleteTimeout = o.deleteTimeout
	}
	if o.tokenTTL != 0 {
		cfg.DefaultTokenTTL = o.tokenTTL
	}
	if o.pendingTTL != 0 {
		cfg.PendingTTL = o.pendingTTL
	}
	if o.maxRetries != nil {
		cfg.MaxRetries = o.maxRetries
	}
	if o.retryBaseDelay > 0 {
		cfg.RetryBaseDelay = o.retryBaseDelay
	}
	if o.cwebpPath != "" {
		cfg.CWebPPath = o.cwebpPath
	}
	if o.dedup != nil {
		cfg.Dedup = *o.dedup
	}
	if o.lazyVariants != nil {
		cfg.LazyVariants = *o.lazyVariants
	}
	if o.smartVariantSkip != nil {
		cfg.SmartVariantSkip = *o.smartVariantSkip
	}
}
