package mediaforge

import (
	"context"
	"time"
)

// Hook is called around storage operations for metrics, tracing, or monitoring.
// Implementors must provide all four methods. To skip individual callbacks,
// embed NoopHook and override only the methods you care about.
type Hook interface {
	// BeforeUpload is called before each file upload (original + variants).
	BeforeUpload(ctx context.Context, key string, size int64)

	// AfterUpload is called after each file upload completes (success or failure).
	AfterUpload(ctx context.Context, key string, size int64, dur time.Duration, err error)

	// BeforeDelete is called before each file deletion.
	BeforeDelete(ctx context.Context, key string)

	// AfterDelete is called after each file deletion completes.
	AfterDelete(ctx context.Context, key string, dur time.Duration, err error)
}

// WithHook sets an observability hook for storage operations.
// Useful for OpenTelemetry spans, Prometheus metrics, etc.
func WithHook(h Hook) Option {
	return func(o *clientOptions) { o.hook = h }
}

// NoopHook is a default no-op implementation.
type NoopHook struct{}

func (NoopHook) BeforeUpload(context.Context, string, int64)                      {}
func (NoopHook) AfterUpload(context.Context, string, int64, time.Duration, error) {}
func (NoopHook) BeforeDelete(context.Context, string)                             {}
func (NoopHook) AfterDelete(context.Context, string, time.Duration, error)        {}
