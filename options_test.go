package mediaforge

import (
	"context"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mock implementations for testing ---

type mockStorage struct {
	uploadFn func(ctx context.Context, key string, r io.Reader, size int64) error
	deleteFn func(ctx context.Context, key string) error
	existsFn func(ctx context.Context, key string) (bool, error)
}

func (m *mockStorage) Upload(ctx context.Context, key string, r io.Reader, size int64) error {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, key, r, size)
	}
	return nil
}
func (m *mockStorage) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}
func (m *mockStorage) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, key)
	}
	return false, nil
}

type mockSigner struct {
	signFn func(key string, expires time.Time) (string, error)
}

func (m *mockSigner) SignURL(key string, expires time.Time) (string, error) {
	if m.signFn != nil {
		return m.signFn(key, expires)
	}
	return fmt.Sprintf("https://cdn.test.com/%s?signed=true", key), nil
}

type mockEncoder struct{}

func (m *mockEncoder) Encode(img image.Image, quality int) ([]byte, error) {
	return []byte("mock-webp-data"), nil
}
func (m *mockEncoder) Extension() string   { return ".webp" }
func (m *mockEncoder) ContentType() string { return "image/webp" }

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// --- NewClient tests ---

func TestNewClient_Success(t *testing.T) {
	client, err := NewClient(&mockStorage{}, &mockSigner{})
	require.NoError(t, err)
	assert.NotNil(t, client)
}

func TestNewClient_NilStorage(t *testing.T) {
	_, err := NewClient(nil, &mockSigner{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage must not be nil")
}

func TestNewClient_NilSigner(t *testing.T) {
	_, err := NewClient(&mockStorage{}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signer must not be nil")
}

func TestNewClient_WithOptions(t *testing.T) {
	fixed := fixedClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	client, err := NewClient(&mockStorage{}, &mockSigner{},
		WithClock(fixed),
		WithMaxFileSize(5*1024*1024),
		WithMinDimensions(50, 50),
		WithMaxDimensions(4000, 4000),
		WithVariants([]VariantConfig{
			{Name: "small", MaxWidth: 100, MaxHeight: 100, Quality: 80},
		}),
		WithRetries(5, 2*time.Second),
		WithTokenTTL(30*time.Minute),
		WithPendingTTL(1*time.Hour),
		WithUploadTimeout(30*time.Second),
		WithDeleteTimeout(5*time.Second),
	)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, int64(5*1024*1024), client.config.MaxFileSize)
	assert.Equal(t, 50, client.config.MinWidth)
	assert.Equal(t, 4000, client.config.MaxWidth)
	assert.Len(t, client.config.Variants, 1)
	assert.Equal(t, "small", client.config.Variants[0].Name)
	assert.Equal(t, 5, *client.config.MaxRetries)
	assert.Equal(t, 30*time.Minute, client.config.DefaultTokenTTL)
	assert.Equal(t, 1*time.Hour, client.config.PendingTTL)
}

func TestNewClient_DefaultsApplied(t *testing.T) {
	client, err := NewClient(&mockStorage{}, &mockSigner{})
	require.NoError(t, err)

	// Check defaults were applied
	assert.Equal(t, int64(10_485_760), client.config.MaxFileSize) // 10MB
	assert.Equal(t, 100, client.config.MinWidth)
	assert.Equal(t, 8000, client.config.MaxWidth)
	assert.Len(t, client.config.Variants, 3) // thumb, medium, large
}

func TestNewClient_InvalidConfig(t *testing.T) {
	_, err := NewClient(&mockStorage{}, &mockSigner{},
		WithMinDimensions(100, 100),
		WithMaxDimensions(50, 50), // min >= max
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MinWidth must be less than MaxWidth")
}

func TestNewClient_WithEncoder(t *testing.T) {
	client, err := NewClient(&mockStorage{}, &mockSigner{},
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)
	assert.NotNil(t, client.encoder)
}

func TestNewClient_WithClock(t *testing.T) {
	fixed := fixedClock{t: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)}
	client, err := NewClient(&mockStorage{}, &mockSigner{},
		WithClock(fixed),
	)
	require.NoError(t, err)
	assert.Equal(t, fixed.t, client.clock.Now())
}

// --- NewBunnyClient tests ---

func TestNewBunnyClient_Success(t *testing.T) {
	skipIfNoCWebP(t)

	client, err := NewBunnyClient("zone", "api-key", "https://cdn.example.com", "cdn-secret")
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.legacy) // Should have legacy for setStorageBaseURL
}

func TestNewBunnyClient_MissingZone(t *testing.T) {
	_, err := NewBunnyClient("", "api-key", "https://cdn.example.com", "cdn-secret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "StorageZone is required")
}

func TestNewBunnyClient_WithOptions(t *testing.T) {
	skipIfNoCWebP(t)

	client, err := NewBunnyClient("zone", "api-key", "https://cdn.example.com", "cdn-secret",
		WithMaxFileSize(5*1024*1024),
		WithRetries(1, 500*time.Millisecond),
	)
	require.NoError(t, err)
	assert.Equal(t, int64(5*1024*1024), client.config.MaxFileSize)
	assert.Equal(t, 1, *client.config.MaxRetries)
}

func TestNewBunnyClient_CustomEncoderSkipsCWebPCheck(t *testing.T) {
	// Should succeed even without cwebp binary because custom encoder is provided
	client, err := NewBunnyClient("zone", "api-key", "https://cdn.example.com", "cdn-secret",
		WithEncoder(&mockEncoder{}),
		WithCWebPPath("/nonexistent/cwebp"), // should be ignored
	)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// --- Integration: NewClient with signing ---

func TestNewClient_GetSignedURL(t *testing.T) {
	fixed := fixedClock{t: time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)}
	signer := &mockSigner{
		signFn: func(key string, expires time.Time) (string, error) {
			return fmt.Sprintf("https://cdn.test.com/%s?expires=%d", key, expires.Unix()), nil
		},
	}

	client, err := NewClient(&mockStorage{}, signer,
		WithClock(fixed),
		WithTokenTTL(1*time.Hour),
	)
	require.NoError(t, err)

	urlSet, err := client.GetSignedURL("stores/t1/images/abc.webp", []string{"thumb"}, 0)
	require.NoError(t, err)
	assert.Contains(t, urlSet.Original, "stores/t1/images/abc.webp")
	assert.Contains(t, urlSet.Variants["thumb"], "abc_thumb.webp")
}

// --- NewClient validation edge cases ---

func TestNewClient_InvalidVariant(t *testing.T) {
	_, err := NewClient(&mockStorage{}, &mockSigner{},
		WithVariants([]VariantConfig{
			{Name: "", MaxWidth: 100, MaxHeight: 100, Quality: 80},
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "variant name must not be empty")
}

func TestNewClient_DuplicateVariants(t *testing.T) {
	_, err := NewClient(&mockStorage{}, &mockSigner{},
		WithVariants([]VariantConfig{
			{Name: "thumb", MaxWidth: 100, MaxHeight: 100, Quality: 80},
			{Name: "thumb", MaxWidth: 200, MaxHeight: 200, Quality: 80},
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate variant name")
}

func TestNewClient_InvalidMIMEType(t *testing.T) {
	_, err := NewClient(&mockStorage{}, &mockSigner{},
		WithAllowedMIMETypes([]string{"text/plain"}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "allowed MIME type must start with image/")
}

func TestNewClient_NegativeRetries(t *testing.T) {
	_, err := NewClient(&mockStorage{}, &mockSigner{},
		WithRetries(-1, time.Second),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MaxRetries must be non-negative")
}

func TestNewClient_InvalidVariantQuality(t *testing.T) {
	_, err := NewClient(&mockStorage{}, &mockSigner{},
		WithVariants([]VariantConfig{
			{Name: "bad", MaxWidth: 100, MaxHeight: 100, Quality: 101},
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "variant quality must be 1-100")
}

func TestNewClient_InvalidVariantDimensions(t *testing.T) {
	_, err := NewClient(&mockStorage{}, &mockSigner{},
		WithVariants([]VariantConfig{
			{Name: "bad", MaxWidth: 0, MaxHeight: 100, Quality: 80},
		}),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "variant dimensions must be positive")
}

func TestNewClient_GetSignedURL_SignerError(t *testing.T) {
	signer := &mockSigner{
		signFn: func(key string, expires time.Time) (string, error) {
			return "", fmt.Errorf("signer failure")
		},
	}
	client, err := NewClient(&mockStorage{}, signer)
	require.NoError(t, err)

	_, err = client.GetSignedURL("stores/t1/images/abc.webp", nil, time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sign original URL")
}

func TestNewClient_GetSignedURL_VariantSignerError(t *testing.T) {
	callCount := 0
	signer := &mockSigner{
		signFn: func(key string, expires time.Time) (string, error) {
			callCount++
			if callCount > 1 {
				return "", fmt.Errorf("variant signer failure")
			}
			return "https://cdn.test.com/original?signed=true", nil
		},
	}
	client, err := NewClient(&mockStorage{}, signer)
	require.NoError(t, err)

	_, err = client.GetSignedURL("stores/t1/images/abc.webp", []string{"thumb"}, time.Hour)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sign variant URL")
}

func TestNewClient_InvalidPendingTTL(t *testing.T) {
	_, err := NewClient(&mockStorage{}, &mockSigner{},
		WithPendingTTL(-1*time.Hour),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PendingTTL must be positive")
}

func TestNewClient_InvalidTokenTTL(t *testing.T) {
	_, err := NewClient(&mockStorage{}, &mockSigner{},
		WithTokenTTL(-1*time.Hour),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DefaultTokenTTL must be positive")
}

// --- Backward compatibility: New(Config) still works ---

func TestNew_BackwardCompatibility(t *testing.T) {
	skipIfNoCWebP(t)

	// Start a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	cfg := testConfig()
	client, err := New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.obj)
	assert.NotNil(t, client.signer)
	assert.NotNil(t, client.clock)
	assert.NotNil(t, client.legacy)
}
