package signing

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// GenerateToken creates a Bunny CDN URL authentication token.
//
// Algorithm (per Bunny CDN Token Authentication spec):
//
//	base64url(SHA-256(securityKey + urlPath + expiresUnix))
//
// Note: This is plain SHA-256 of concatenated values, NOT HMAC-SHA256.
// The urlPath must include a leading slash: "/stores/t1/images/abc.webp"
func GenerateToken(securityKey, urlPath string, expiresUnix int64) string {
	raw := fmt.Sprintf("%s%s%d", securityKey, urlPath, expiresUnix)
	hash := sha256.Sum256([]byte(raw))
	token := base64.StdEncoding.EncodeToString(hash[:])

	// Make URL-safe: replace +/ with -_, remove padding
	token = strings.ReplaceAll(token, "+", "-")
	token = strings.ReplaceAll(token, "/", "_")
	token = strings.TrimRight(token, "=")

	return token
}

// BuildSignedURL constructs a full signed CDN URL.
// Output: "{cdnBase}{urlPath}?token={token}&expires={expiresUnix}"
func BuildSignedURL(cdnBase, urlPath, token string, expiresUnix int64) string {
	return fmt.Sprintf("%s%s?token=%s&expires=%d", cdnBase, urlPath, token, expiresUnix)
}
