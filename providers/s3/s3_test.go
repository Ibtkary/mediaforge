package s3

import (
	"fmt"
	"testing"

	smithy "github.com/aws/smithy-go"
	"github.com/stretchr/testify/assert"
)

func TestS3Error_Error(t *testing.T) {
	err := &S3Error{
		Message:   "upload failed",
		Operation: "PutObject",
		Retryable: true,
	}
	assert.Equal(t, "s3 PutObject: upload failed", err.Error())
}

func TestS3Error_IsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		retryable bool
	}{
		{"retryable", true},
		{"non-retryable", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &S3Error{Message: "test", Operation: "test", Retryable: tt.retryable}
			assert.Equal(t, tt.retryable, err.IsRetryable())
		})
	}
}

func TestS3Error_Unwrap(t *testing.T) {
	inner := assert.AnError
	err := &S3Error{Message: "test", Operation: "test", err: inner}
	assert.ErrorIs(t, err, inner)
}

func TestIsRetryable_NilError(t *testing.T) {
	assert.False(t, isRetryable(nil))
}

// mockAPIError implements smithy.APIError for testing.
type mockAPIError struct {
	code    string
	message string
}

var _ smithy.APIError = (*mockAPIError)(nil) // compile-time check

func (e *mockAPIError) Error() string                 { return e.message }
func (e *mockAPIError) ErrorCode() string              { return e.code }
func (e *mockAPIError) ErrorMessage() string           { return e.message }
func (e *mockAPIError) ErrorFault() smithy.ErrorFault { return smithy.FaultUnknown }

func TestIsRetryable_SmithyAPICodes(t *testing.T) {
	tests := []struct {
		code      string
		retryable bool
	}{
		{"RequestTimeout", true},
		{"ThrottlingException", true},
		{"Throttling", true},
		{"TooManyRequestsException", true},
		{"ServiceUnavailable", true},
		{"InternalError", true},
		{"InternalServiceError", true},
		{"SlowDown", true},
		{"AccessDenied", false},
		{"NoSuchBucket", false},
		{"NoSuchKey", false},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			err := &mockAPIError{code: tt.code, message: "test"}
			assert.Equal(t, tt.retryable, isRetryable(err))
		})
	}
}

func TestIsRetryable_WrappedSmithyError(t *testing.T) {
	inner := &mockAPIError{code: "ThrottlingException", message: "rate limited"}
	wrapped := fmt.Errorf("s3 operation: %w", inner)
	assert.True(t, isRetryable(wrapped))
}

func TestIsRetryable_PlainError(t *testing.T) {
	assert.False(t, isRetryable(fmt.Errorf("some random error")))
}

func TestNewStorage(t *testing.T) {
	s := NewStorage(nil, "my-bucket")
	assert.NotNil(t, s)
	assert.Equal(t, "my-bucket", s.bucket)
}

func TestNewSigner(t *testing.T) {
	s := NewSigner(nil, "my-bucket")
	assert.NotNil(t, s)
	assert.Equal(t, "my-bucket", s.bucket)
}

func TestContentTypeFromKey(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"stores/t1/images/abc.webp", "image/webp"},
		{"stores/t1/images/abc_thumb.webp", "image/webp"},
		{"images/photo.jpg", "image/jpeg"},
		{"images/photo.jpeg", "image/jpeg"},
		{"images/photo.png", "image/png"},
		{"data/file.bin", "application/octet-stream"},
		{"no-extension", "application/octet-stream"},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.expected, contentTypeFromKey(tt.key))
		})
	}
}
