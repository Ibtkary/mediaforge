package s3

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Signer implements mediastore.URLSigner using S3 presigned URLs.
type Signer struct {
	presignClient *s3.PresignClient
	bucket        string
}

// NewSigner creates a new S3-backed URLSigner.
func NewSigner(presignClient *s3.PresignClient, bucket string) *Signer {
	return &Signer{presignClient: presignClient, bucket: bucket}
}

// SignURL generates a presigned GET URL for the given storage key.
func (s *Signer) SignURL(key string, expires time.Time) (string, error) {
	ttl := time.Until(expires)
	if ttl <= 0 {
		return "", fmt.Errorf("s3 signer: expiry time is in the past")
	}

	input := &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}

	result, err := s.presignClient.PresignGetObject(context.Background(), input,
		s3.WithPresignExpires(ttl),
	)
	if err != nil {
		return "", fmt.Errorf("s3 signer: presign failed for %q: %w", key, err)
	}

	return result.URL, nil
}
