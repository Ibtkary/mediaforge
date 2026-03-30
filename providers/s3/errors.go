package s3

import (
	"errors"
	"net"

	smithy "github.com/aws/smithy-go"
)

// S3Error represents an error from S3 operations.
// It implements the mediastore.RetryableError interface.
type S3Error struct {
	Message   string
	Operation string
	Retryable bool
	err       error
}

func (e *S3Error) Error() string {
	return "s3 " + e.Operation + ": " + e.Message
}

func (e *S3Error) Unwrap() error {
	return e.err
}

// IsRetryable returns whether this error should be retried.
func (e *S3Error) IsRetryable() bool {
	return e.Retryable
}

// retryableErrorCodes are AWS API error codes that indicate a transient failure.
var retryableErrorCodes = map[string]bool{
	"RequestTimeout":           true,
	"RequestTimeoutException":  true,
	"ThrottlingException":      true,
	"Throttling":               true,
	"TooManyRequestsException": true,
	"ServiceUnavailable":       true,
	"InternalError":            true,
	"InternalServiceError":     true,
	"SlowDown":                 true,
	"EC2ThrottledException":    true,
}

// isRetryable determines if an AWS error should be retried.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are retryable
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	// Check smithy APIError code (structured, SDK-version-safe)
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return retryableErrorCodes[apiErr.ErrorCode()]
	}

	return false
}
