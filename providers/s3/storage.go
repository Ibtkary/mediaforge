package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	smithy "github.com/aws/smithy-go"
)

// Storage implements mediastore.ObjectStorage for AWS S3.
type Storage struct {
	client *s3.Client
	bucket string
}

// NewStorage creates a new S3-backed ObjectStorage.
func NewStorage(client *s3.Client, bucket string) *Storage {
	return &Storage{client: client, bucket: bucket}
}

// Upload stores data at the given key in S3.
// Content type is inferred from the key extension (.webp → image/webp).
func (s *Storage) Upload(ctx context.Context, key string, r io.Reader, size int64) error {
	ct := contentTypeFromKey(key)
	input := &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          r,
		ContentLength: &size,
		ContentType:   &ct,
	}

	_, err := s.client.PutObject(ctx, input)
	if err != nil {
		return &S3Error{
			Message:   fmt.Sprintf("upload %q: %s", key, err.Error()),
			Operation: "PutObject",
			Retryable: isRetryable(err),
			err:       err,
		}
	}
	return nil
}

// Delete removes the object at key from S3.
// Returns nil if the object does not exist.
func (s *Storage) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return &S3Error{
			Message:   fmt.Sprintf("delete %q: %s", key, err.Error()),
			Operation: "DeleteObject",
			Retryable: isRetryable(err),
			err:       err,
		}
	}
	return nil
}

// Exists checks whether an object exists at key in S3.
func (s *Storage) Exists(ctx context.Context, key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}

	_, err := s.client.HeadObject(ctx, input)
	if err != nil {
		// AWS SDK v2: types.NoSuchKey for some configurations
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return false, nil
		}
		// HeadObject returns smithy.APIError with code "NotFound" in many setups
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "NotFound" {
			return false, nil
		}
		// Final guard: HTTP 404 via transport layer
		type httpResponseErr interface {
			HTTPStatusCode() int
		}
		var httpErr httpResponseErr
		if errors.As(err, &httpErr) && httpErr.HTTPStatusCode() == http.StatusNotFound {
			return false, nil
		}
		return false, &S3Error{
			Message:   fmt.Sprintf("exists %q: %s", key, err.Error()),
			Operation: "HeadObject",
			Retryable: isRetryable(err),
			err:       err,
		}
	}
	return true, nil
}

// contentTypeFromKey returns a MIME type based on the key's file extension.
func contentTypeFromKey(key string) string {
	if strings.HasSuffix(key, ".webp") {
		return "image/webp"
	}
	if strings.HasSuffix(key, ".jpg") || strings.HasSuffix(key, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(key, ".png") {
		return "image/png"
	}
	return "application/octet-stream"
}
