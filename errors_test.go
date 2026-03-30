package mediaforge

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name:     "basic error",
			err:      ErrConfig("missing API key"),
			expected: "mediastore [config]: missing API key",
		},
		{
			name:     "error with op",
			err:      ErrValidation("file too large").WithOp("Upload"),
			expected: "mediastore [validation] Upload: file too large",
		},
		{
			name:     "error with detail",
			err:      ErrProcessing("resize failed").WithDetail("width=0"),
			expected: "mediastore [processing]: resize failed (width=0)",
		},
		{
			name:     "error with op and detail",
			err:      ErrStorage("timeout", true).WithOp("PutObject").WithDetail("elapsed=30s"),
			expected: "mediastore [storage] PutObject: timeout (elapsed=30s)",
		},
		{
			name: "error with cause",
			err:  ErrStorage("connection failed", true).WithCause(fmt.Errorf("dial tcp: timeout")),
			expected: "mediastore [storage]: connection failed: dial tcp: timeout",
		},
		{
			name: "full error",
			err: ErrSigning("token generation failed").
				WithOp("GenerateToken").
				WithDetail("keyLen=0").
				WithCause(fmt.Errorf("empty key")),
			expected: "mediastore [signing] GenerateToken: token generation failed (keyLen=0): empty key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := ErrStorage("failed", true).WithCause(cause)

	unwrapped := errors.Unwrap(err)
	assert.Equal(t, cause, unwrapped)
}

func TestErrorUnwrapNil(t *testing.T) {
	err := ErrConfig("no cause")
	assert.Nil(t, errors.Unwrap(err))
}

func TestErrorIs(t *testing.T) {
	tests := []struct {
		name   string
		err    *Error
		target *Error
		match  bool
	}{
		{
			name:   "same category matches",
			err:    ErrConfig("a"),
			target: &Error{Category: CategoryConfig},
			match:  true,
		},
		{
			name:   "different category no match",
			err:    ErrConfig("a"),
			target: &Error{Category: CategoryStorage},
			match:  false,
		},
		{
			name:   "storage matches storage",
			err:    ErrStorage("a", true),
			target: &Error{Category: CategoryStorage},
			match:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.match, errors.Is(tt.err, tt.target))
		})
	}
}

func TestErrorIsNotErrorType(t *testing.T) {
	err := ErrConfig("test")
	assert.False(t, errors.Is(err, fmt.Errorf("not a mediastore error")))
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "retryable storage error",
			err:       ErrStorage("timeout", true),
			retryable: true,
		},
		{
			name:      "non-retryable storage error",
			err:       ErrStorage("not found", false),
			retryable: false,
		},
		{
			name:      "config error never retryable",
			err:       ErrConfig("missing key"),
			retryable: false,
		},
		{
			name:      "validation error never retryable",
			err:       ErrValidation("bad input"),
			retryable: false,
		},
		{
			name:      "processing error never retryable",
			err:       ErrProcessing("corrupt"),
			retryable: false,
		},
		{
			name:      "signing error never retryable",
			err:       ErrSigning("expired"),
			retryable: false,
		},
		{
			name:      "wrapped retryable error",
			err:       fmt.Errorf("outer: %w", ErrStorage("timeout", true)),
			retryable: true,
		},
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "non-mediastore error",
			err:       fmt.Errorf("plain error"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.retryable, IsRetryable(tt.err))
		})
	}
}

func TestIsRetryable_ChainWithNonRetryableFirst(t *testing.T) {
	// Chain: non-retryable *Error wrapping retryable *Error.
	// Previous implementation had an infinite loop here because errors.As kept
	// finding the outer *Error on every iteration without advancing past it.
	inner := ErrStorage("timeout", true)
	outer := ErrStorage("outer failure", false).WithCause(inner)

	assert.True(t, IsRetryable(outer), "should find retryable error deeper in the chain")
}

func TestIsRetryable_AllNonRetryable(t *testing.T) {
	inner := ErrStorage("not found", false)
	outer := ErrStorage("outer", false).WithCause(inner)
	assert.False(t, IsRetryable(outer))
}

func TestConstructorCategories(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		category ErrorCategory
	}{
		{"config", ErrConfig("test"), CategoryConfig},
		{"validation", ErrValidation("test"), CategoryValidation},
		{"processing", ErrProcessing("test"), CategoryProcessing},
		{"storage", ErrStorage("test", false), CategoryStorage},
		{"signing", ErrSigning("test"), CategorySigning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.category, tt.err.Category)
		})
	}
}

func TestBuilderMethodsChain(t *testing.T) {
	err := ErrStorage("failed", true).
		WithOp("Upload").
		WithDetail("size=15MB").
		WithCause(fmt.Errorf("too big"))

	require.NotNil(t, err)
	assert.Equal(t, CategoryStorage, err.Category)
	assert.Equal(t, "failed", err.Msg)
	assert.True(t, err.Retryable)
	assert.Equal(t, "Upload", err.Op)
	assert.Equal(t, "size=15MB", err.Detail)
	assert.EqualError(t, err.err, "too big")
}
