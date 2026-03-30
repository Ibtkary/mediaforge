package mediaforge

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// retryableErr implements RetryableError.
type retryableErr struct {
	retryable bool
	msg       string
}

func (e *retryableErr) Error() string      { return e.msg }
func (e *retryableErr) IsRetryable() bool { return e.retryable }

// plainErr is a plain error without RetryableError interface.
type plainErr struct{ msg string }

func (e *plainErr) Error() string { return e.msg }

func TestIsRetryableError_Nil(t *testing.T) {
	assert.False(t, isRetryableError(nil))
}

func TestIsRetryableError_RetryableTrue(t *testing.T) {
	err := &retryableErr{retryable: true, msg: "server error"}
	assert.True(t, isRetryableError(err))
}

func TestIsRetryableError_RetryableFalse(t *testing.T) {
	err := &retryableErr{retryable: false, msg: "bad request"}
	assert.False(t, isRetryableError(err))
}

func TestIsRetryableError_PlainError(t *testing.T) {
	err := &plainErr{msg: "some error"}
	assert.False(t, isRetryableError(err))
}

func TestIsRetryableError_WrappedRetryable(t *testing.T) {
	inner := &retryableErr{retryable: true, msg: "server error"}
	wrapped := fmt.Errorf("upload failed: %w", inner)
	assert.True(t, isRetryableError(wrapped))
}

func TestIsRetryableError_WrappedNonRetryable(t *testing.T) {
	inner := &retryableErr{retryable: false, msg: "bad request"}
	wrapped := fmt.Errorf("upload failed: %w", inner)
	assert.False(t, isRetryableError(wrapped))
}
