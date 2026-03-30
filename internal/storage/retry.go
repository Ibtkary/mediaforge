package storage

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// RetryConfig configures the retry behavior.
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration // cap at 30s
}

// WithRetry executes op with exponential backoff and jitter.
// Total calls = MaxRetries + 1. If shouldRetry returns false, the error is returned immediately.
func WithRetry(ctx context.Context, cfg RetryConfig, shouldRetry func(error) bool, op func() error) error {
	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := backoffDelay(cfg.BaseDelay, cfg.MaxDelay, attempt-1)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		lastErr = op()
		if lastErr == nil {
			return nil
		}

		if !shouldRetry(lastErr) {
			return lastErr
		}
	}

	return lastErr
}

// backoffDelay calculates delay with exponential backoff and ±25% jitter, capped at maxDelay.
func backoffDelay(base, maxDelay time.Duration, attempt int) time.Duration {
	delay := float64(base) * math.Pow(2, float64(attempt))
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	// ±25% jitter
	jitter := delay * 0.25
	delay = delay - jitter + (rand.Float64() * 2 * jitter)

	return time.Duration(delay)
}
