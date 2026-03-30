package storage

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRetry_SucceedsFirstAttempt(t *testing.T) {
	var calls int32
	err := WithRetry(context.Background(), RetryConfig{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: time.Second}, func(error) bool { return true }, func() error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestWithRetry_SucceedsAfterRetries(t *testing.T) {
	var calls int32
	err := WithRetry(context.Background(), RetryConfig{MaxRetries: 3, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, func(error) bool { return true }, func() error {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return errors.New("transient")
		}
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestWithRetry_ExhaustsRetries(t *testing.T) {
	expected := errors.New("persistent")
	var calls int32
	err := WithRetry(context.Background(), RetryConfig{MaxRetries: 2, BaseDelay: time.Millisecond, MaxDelay: 10 * time.Millisecond}, func(error) bool { return true }, func() error {
		atomic.AddInt32(&calls, 1)
		return expected
	})

	require.Error(t, err)
	assert.Equal(t, expected, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls)) // initial + 2 retries
}

func TestWithRetry_NonRetryableStopsImmediately(t *testing.T) {
	nonRetryable := errors.New("fatal")
	var calls int32
	err := WithRetry(context.Background(), RetryConfig{MaxRetries: 5, BaseDelay: time.Millisecond, MaxDelay: time.Second}, func(err error) bool {
		return err.Error() != "fatal"
	}, func() error {
		atomic.AddInt32(&calls, 1)
		return nonRetryable
	})

	require.Error(t, err)
	assert.Equal(t, nonRetryable, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestWithRetry_MaxRetriesZero(t *testing.T) {
	var calls int32
	err := WithRetry(context.Background(), RetryConfig{MaxRetries: 0, BaseDelay: time.Millisecond, MaxDelay: time.Second}, func(error) bool { return true }, func() error {
		atomic.AddInt32(&calls, 1)
		return errors.New("fail")
	})

	require.Error(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestWithRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var calls int32

	err := WithRetry(ctx, RetryConfig{MaxRetries: 10, BaseDelay: 100 * time.Millisecond, MaxDelay: time.Second}, func(error) bool { return true }, func() error {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			cancel()
		}
		return errors.New("fail")
	})

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestWithRetry_BackoffIncreases(t *testing.T) {
	cfg := RetryConfig{MaxRetries: 3, BaseDelay: 10 * time.Millisecond, MaxDelay: time.Second}
	var timestamps []time.Time

	WithRetry(context.Background(), cfg, func(error) bool { return true }, func() error {
		timestamps = append(timestamps, time.Now())
		return errors.New("fail")
	})

	require.Len(t, timestamps, 4)
	// Second gap should be roughly 2x the first gap (with jitter tolerance)
	gap1 := timestamps[2].Sub(timestamps[1])
	gap0 := timestamps[1].Sub(timestamps[0])
	if gap0 > 0 {
		ratio := float64(gap1) / float64(gap0)
		assert.Greater(t, ratio, 1.0, "backoff should increase")
	}
}

func TestWithRetry_MaxDelayCap(t *testing.T) {
	maxDelay := 20 * time.Millisecond
	cfg := RetryConfig{MaxRetries: 5, BaseDelay: 50 * time.Millisecond, MaxDelay: maxDelay}
	var timestamps []time.Time

	WithRetry(context.Background(), cfg, func(error) bool { return true }, func() error {
		timestamps = append(timestamps, time.Now())
		return errors.New("fail")
	})

	require.GreaterOrEqual(t, len(timestamps), 3)
	for i := 1; i < len(timestamps); i++ {
		gap := timestamps[i].Sub(timestamps[i-1])
		// Allow 50% tolerance above maxDelay for jitter + scheduling
		assert.Less(t, gap, maxDelay*2, "gap %d should be capped near maxDelay", i)
	}
}
