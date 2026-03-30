package mediaforge

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ValidConfig(t *testing.T) {
	cfg := testConfig()
	client, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, client)
}

func TestNew_AppliesDefaults(t *testing.T) {
	cfg := testConfig()
	client, err := New(cfg)

	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, int64(defaultMaxFileSize), client.config.MaxFileSize)
	assert.Equal(t, defaultMinWidth, client.config.MinWidth)
	assert.Equal(t, defaultMaxRetries, *client.config.MaxRetries)
	assert.Equal(t, DefaultVariants(), client.config.Variants)
	assert.Equal(t, defaultTokenTTL, client.config.DefaultTokenTTL)
}

func TestNew_MissingRequiredField(t *testing.T) {
	cfg := testConfig()
	cfg.StorageZone = ""
	client, err := New(cfg)

	require.Error(t, err)
	assert.Nil(t, client)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryConfig, mErr.Category)
}

func TestNew_CWebPNotFound(t *testing.T) {
	cfg := testConfig()
	cfg.CWebPPath = "/nonexistent/path/to/cwebp-binary-that-does-not-exist"
	client, err := New(cfg)

	require.Error(t, err)
	assert.Nil(t, client)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryConfig, mErr.Category)
	assert.Contains(t, mErr.Msg, "cwebp binary not found")
	assert.Equal(t, "New", mErr.Op)
}

func TestNew_PreservesCustomConfig(t *testing.T) {
	cfg := testConfig()
	cfg.MaxFileSize = 5_000_000
	cfg.MaxRetries = intPtr(5)
	client, err := New(cfg)

	require.NoError(t, err)
	assert.Equal(t, int64(5_000_000), client.config.MaxFileSize)
	assert.Equal(t, 5, *client.config.MaxRetries)
}

func TestNew_CDNBaseURLSlashStripped(t *testing.T) {
	cfg := testConfig()
	cfg.CDNBaseURL = "https://cdn.example.com/"
	client, err := New(cfg)

	require.NoError(t, err)
	assert.Equal(t, "https://cdn.example.com", client.config.CDNBaseURL)
}
