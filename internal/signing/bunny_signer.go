package signing

import (
	"fmt"
	"strings"
	"time"
)

// BunnySigner implements the URLSigner interface for Bunny CDN.
// It wraps the existing GenerateToken and BuildSignedURL functions.
type BunnySigner struct {
	cdnBaseURL     string
	cdnSecurityKey string
}

// NewBunnySigner creates a BunnySigner with the given CDN configuration.
func NewBunnySigner(cdnBaseURL, cdnSecurityKey string) *BunnySigner {
	return &BunnySigner{
		cdnBaseURL:     strings.TrimRight(cdnBaseURL, "/"),
		cdnSecurityKey: cdnSecurityKey,
	}
}

// SignURL generates a signed CDN URL for the given storage key with an expiration time.
// The key should be the storage path (e.g., "stores/t1/images/abc.webp").
func (s *BunnySigner) SignURL(key string, expires time.Time) (string, error) {
	if key == "" {
		return "", fmt.Errorf("signing: key must not be empty")
	}
	if s.cdnSecurityKey == "" {
		return "", fmt.Errorf("signing: CDN security key is not configured")
	}

	urlPath := "/" + key
	expiresUnix := expires.Unix()
	token := GenerateToken(s.cdnSecurityKey, urlPath, expiresUnix)
	return BuildSignedURL(s.cdnBaseURL, urlPath, token, expiresUnix), nil
}
