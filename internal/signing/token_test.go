package signing

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateToken_Deterministic(t *testing.T) {
	token1 := GenerateToken("secret-key", "/stores/t1/images/abc.webp", 1700000000)
	token2 := GenerateToken("secret-key", "/stores/t1/images/abc.webp", 1700000000)
	assert.Equal(t, token1, token2)
}

func TestGenerateToken_ChangesWithKey(t *testing.T) {
	token1 := GenerateToken("key-a", "/path", 1700000000)
	token2 := GenerateToken("key-b", "/path", 1700000000)
	assert.NotEqual(t, token1, token2)
}

func TestGenerateToken_ChangesWithPath(t *testing.T) {
	token1 := GenerateToken("secret", "/path-a", 1700000000)
	token2 := GenerateToken("secret", "/path-b", 1700000000)
	assert.NotEqual(t, token1, token2)
}

func TestGenerateToken_ChangesWithExpiry(t *testing.T) {
	token1 := GenerateToken("secret", "/path", 1700000000)
	token2 := GenerateToken("secret", "/path", 1700000001)
	assert.NotEqual(t, token1, token2)
}

func TestGenerateToken_URLSafeBase64(t *testing.T) {
	// Generate several tokens and check none contain +, /, or =
	keys := []string{"key1", "key2", "test-key-long-enough-to-produce-varied-output"}
	paths := []string{"/a", "/b/c", "/stores/tenant/images/file.webp"}
	for _, key := range keys {
		for _, path := range paths {
			token := GenerateToken(key, path, 1700000000)
			assert.False(t, strings.Contains(token, "+"), "token should not contain +: %s", token)
			assert.False(t, strings.Contains(token, "/"), "token should not contain /: %s", token)
			assert.False(t, strings.HasSuffix(token, "="), "token should not have = padding: %s", token)
		}
	}
}

func TestGenerateToken_NonEmpty(t *testing.T) {
	token := GenerateToken("key", "/path", 1700000000)
	assert.NotEmpty(t, token)
}

func TestBuildSignedURL_Format(t *testing.T) {
	tests := []struct {
		name    string
		cdnBase string
		urlPath string
		token   string
		expires int64
		want    string
	}{
		{
			name:    "standard URL",
			cdnBase: "https://cdn.nafees.com",
			urlPath: "/stores/t1/images/abc.webp",
			token:   "abc123token",
			expires: 1700000000,
			want:    "https://cdn.nafees.com/stores/t1/images/abc.webp?token=abc123token&expires=1700000000",
		},
		{
			name:    "different base",
			cdnBase: "https://other-cdn.example.com",
			urlPath: "/files/image.webp",
			token:   "xyz",
			expires: 9999999999,
			want:    "https://other-cdn.example.com/files/image.webp?token=xyz&expires=9999999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSignedURL(tt.cdnBase, tt.urlPath, tt.token, tt.expires)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildSignedURL_Integration(t *testing.T) {
	key := "my-security-key"
	path := "/stores/tenant1/images/abc_def.webp"
	expires := int64(1700000000)

	token := GenerateToken(key, path, expires)
	url := BuildSignedURL("https://cdn.nafees.com", path, token, expires)

	assert.Contains(t, url, "https://cdn.nafees.com/stores/tenant1/images/abc_def.webp")
	assert.Contains(t, url, "token="+token)
	assert.Contains(t, url, "expires=1700000000")
}
