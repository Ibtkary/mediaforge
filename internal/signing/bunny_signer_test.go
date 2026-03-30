package signing

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBunnySigner_SignURL_Success(t *testing.T) {
	signer := NewBunnySigner("https://cdn.example.com", "my-secret-key")
	expires := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	url, err := signer.SignURL("stores/t1/images/abc.webp", expires)

	require.NoError(t, err)
	assert.Contains(t, url, "https://cdn.example.com/stores/t1/images/abc.webp")
	assert.Contains(t, url, "token=")
	assert.Contains(t, url, "expires=")
}

func TestBunnySigner_SignURL_EmptyKey(t *testing.T) {
	signer := NewBunnySigner("https://cdn.example.com", "my-secret-key")
	expires := time.Now().Add(1 * time.Hour)

	_, err := signer.SignURL("", expires)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "key must not be empty")
}

func TestBunnySigner_SignURL_EmptySecurityKey(t *testing.T) {
	signer := NewBunnySigner("https://cdn.example.com", "")
	expires := time.Now().Add(1 * time.Hour)

	_, err := signer.SignURL("stores/t1/images/abc.webp", expires)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "security key is not configured")
}

func TestBunnySigner_SignURL_TrailingSlashTrimmed(t *testing.T) {
	signer := NewBunnySigner("https://cdn.example.com///", "my-secret-key")
	expires := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	url, err := signer.SignURL("stores/t1/images/abc.webp", expires)

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(url, "https://cdn.example.com/stores/"),
		"URL should not have double slashes: %s", url)
}

func TestBunnySigner_SignURL_Deterministic(t *testing.T) {
	signer := NewBunnySigner("https://cdn.example.com", "my-secret-key")
	expires := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	url1, err1 := signer.SignURL("stores/t1/images/abc.webp", expires)
	url2, err2 := signer.SignURL("stores/t1/images/abc.webp", expires)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, url1, url2, "same input should produce same URL")
}

func TestBunnySigner_SignURL_DifferentKeys(t *testing.T) {
	signer := NewBunnySigner("https://cdn.example.com", "my-secret-key")
	expires := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	url1, _ := signer.SignURL("stores/t1/images/abc.webp", expires)
	url2, _ := signer.SignURL("stores/t1/images/def.webp", expires)

	assert.NotEqual(t, url1, url2, "different keys should produce different URLs")
}

func TestBunnySigner_ConsistentWithDirectFunctions(t *testing.T) {
	cdnBase := "https://cdn.example.com"
	securityKey := "my-secret-key"
	key := "stores/t1/images/abc.webp"
	expires := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)

	// Generate via signer
	signer := NewBunnySigner(cdnBase, securityKey)
	signerURL, err := signer.SignURL(key, expires)
	require.NoError(t, err)

	// Generate via direct functions
	urlPath := "/" + key
	expiresUnix := expires.Unix()
	token := GenerateToken(securityKey, urlPath, expiresUnix)
	directURL := BuildSignedURL(cdnBase, urlPath, token, expiresUnix)

	assert.Equal(t, directURL, signerURL, "signer should produce same URL as direct functions")
}
