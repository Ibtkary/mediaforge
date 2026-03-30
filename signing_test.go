package mediaforge

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSignedURL_Success(t *testing.T) {
	client, err := New(testConfig())
	require.NoError(t, err)

	set, err := client.GetSignedURL("stores/t1/images/abc.webp", []string{"thumb", "medium"}, 30*time.Minute)

	require.NoError(t, err)
	require.NotNil(t, set)
	assert.NotEmpty(t, set.Original)
	assert.Contains(t, set.Original, "token=")
	assert.Contains(t, set.Original, "expires=")
	require.Len(t, set.Variants, 2)
	assert.NotEmpty(t, set.Variants["thumb"])
	assert.NotEmpty(t, set.Variants["medium"])
}

func TestGetSignedURL_EmptyPath(t *testing.T) {
	client, err := New(testConfig())
	require.NoError(t, err)

	_, err = client.GetSignedURL("", nil, 0)

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryValidation, mErr.Category)
	assert.Equal(t, "GetSignedURL", mErr.Op)
}

func TestGetSignedURL_DefaultTTL(t *testing.T) {
	client, err := New(testConfig())
	require.NoError(t, err)

	before := time.Now()
	set, err := client.GetSignedURL("stores/t1/images/abc.webp", nil, 0)
	require.NoError(t, err)

	// Extract expires from URL
	parts := strings.Split(set.Original, "expires=")
	require.Len(t, parts, 2)

	// The expires should be approximately now + defaultTokenTTL (1 hour)
	expectedMin := before.Add(defaultTokenTTL - time.Second).Unix()
	expectedMax := before.Add(defaultTokenTTL + time.Second).Unix()

	var expiresUnix int64
	_, parseErr := parseExpiresFromURL(set.Original, &expiresUnix)
	require.True(t, parseErr)
	assert.True(t, expiresUnix >= expectedMin && expiresUnix <= expectedMax,
		"expires %d not in range [%d, %d]", expiresUnix, expectedMin, expectedMax)
}

func TestGetSignedURL_CustomTTL(t *testing.T) {
	client, err := New(testConfig())
	require.NoError(t, err)

	before := time.Now()
	set, err := client.GetSignedURL("stores/t1/images/abc.webp", nil, 30*time.Minute)
	require.NoError(t, err)

	expectedMin := before.Add(30*time.Minute - time.Second).Unix()
	expectedMax := before.Add(30*time.Minute + time.Second).Unix()

	var expiresUnix int64
	_, parseErr := parseExpiresFromURL(set.Original, &expiresUnix)
	require.True(t, parseErr)
	assert.True(t, expiresUnix >= expectedMin && expiresUnix <= expectedMax)
}

func TestGetSignedURL_NegativeTTL(t *testing.T) {
	client, err := New(testConfig())
	require.NoError(t, err)

	before := time.Now()
	set, err := client.GetSignedURL("stores/t1/images/abc.webp", nil, -5*time.Minute)
	require.NoError(t, err)

	// Negative TTL should fall back to default
	expectedMin := before.Add(defaultTokenTTL - time.Second).Unix()
	expectedMax := before.Add(defaultTokenTTL + time.Second).Unix()

	var expiresUnix int64
	_, parseErr := parseExpiresFromURL(set.Original, &expiresUnix)
	require.True(t, parseErr)
	assert.True(t, expiresUnix >= expectedMin && expiresUnix <= expectedMax)
}

func TestGetSignedURL_NoVariants(t *testing.T) {
	client, err := New(testConfig())
	require.NoError(t, err)

	set, err := client.GetSignedURL("stores/t1/images/abc.webp", nil, time.Hour)

	require.NoError(t, err)
	assert.NotEmpty(t, set.Original)
	assert.Empty(t, set.Variants)
}

func TestGetSignedURL_URLFormat(t *testing.T) {
	client, err := New(testConfig())
	require.NoError(t, err)

	set, err := client.GetSignedURL("stores/t1/images/abc.webp", []string{"thumb"}, time.Hour)

	require.NoError(t, err)
	// Original URL format: {cdnBase}/{storagePath}?token=...&expires=...
	assert.True(t, strings.HasPrefix(set.Original, "https://cdn.example.com/stores/t1/images/abc.webp?token="))
	assert.Contains(t, set.Original, "&expires=")

	// Variant URL
	thumbURL := set.Variants["thumb"]
	assert.True(t, strings.HasPrefix(thumbURL, "https://cdn.example.com/stores/t1/images/abc_thumb.webp?token="))
}

func TestGetSignedURL_ConsistentExpiry(t *testing.T) {
	client, err := New(testConfig())
	require.NoError(t, err)

	set, err := client.GetSignedURL("stores/t1/images/abc.webp", []string{"thumb", "medium"}, time.Hour)

	require.NoError(t, err)

	// All URLs should have the same expires value
	var origExpires int64
	parseExpiresFromURL(set.Original, &origExpires)

	for _, vURL := range set.Variants {
		var vExpires int64
		parseExpiresFromURL(vURL, &vExpires)
		assert.Equal(t, origExpires, vExpires)
	}
}

func TestGetSignedURL_DeterministicTokens(t *testing.T) {
	client, err := New(testConfig())
	require.NoError(t, err)

	// Two calls at effectively the same second produce the same token
	set1, err1 := client.GetSignedURL("stores/t1/images/abc.webp", nil, time.Hour)
	set2, err2 := client.GetSignedURL("stores/t1/images/abc.webp", nil, time.Hour)

	require.NoError(t, err1)
	require.NoError(t, err2)

	// Extract tokens — they should match if called within the same second
	token1 := extractToken(set1.Original)
	token2 := extractToken(set2.Original)
	assert.Equal(t, token1, token2)
}

// --- helpers ---

func parseExpiresFromURL(url string, out *int64) (string, bool) {
	idx := strings.Index(url, "expires=")
	if idx < 0 {
		return "", false
	}
	expStr := url[idx+len("expires="):]
	// Trim anything after the number
	if ampIdx := strings.Index(expStr, "&"); ampIdx >= 0 {
		expStr = expStr[:ampIdx]
	}
	var n int64
	for _, c := range expStr {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		} else {
			break
		}
	}
	*out = n
	return expStr, n > 0
}

func extractToken(url string) string {
	idx := strings.Index(url, "token=")
	if idx < 0 {
		return ""
	}
	rest := url[idx+len("token="):]
	if ampIdx := strings.Index(rest, "&"); ampIdx >= 0 {
		return rest[:ampIdx]
	}
	return rest
}
