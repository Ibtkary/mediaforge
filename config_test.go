package mediaforge

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConfig returns a valid Config for tests.
func testConfig() Config {
	return Config{
		StorageZone:    "test-zone",
		StorageAPIKey:  "test-api-key",
		CDNBaseURL:     "https://cdn.example.com",
		CDNSecurityKey: "test-security-key",
	}
}

func TestConfigValidation_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantMsg string
	}{
		{
			name:    "missing StorageZone",
			modify:  func(c *Config) { c.StorageZone = "" },
			wantMsg: "StorageZone is required",
		},
		{
			name:    "missing StorageAPIKey",
			modify:  func(c *Config) { c.StorageAPIKey = "" },
			wantMsg: "StorageAPIKey is required",
		},
		{
			name:    "missing CDNBaseURL",
			modify:  func(c *Config) { c.CDNBaseURL = "" },
			wantMsg: "CDNBaseURL is required",
		},
		{
			name:    "missing CDNSecurityKey",
			modify:  func(c *Config) { c.CDNSecurityKey = "" },
			wantMsg: "CDNSecurityKey is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.applyDefaults()
			tt.modify(&cfg)
			err := cfg.validate()

			require.Error(t, err)
			var mErr *Error
			require.True(t, errors.As(err, &mErr))
			assert.Equal(t, CategoryConfig, mErr.Category)
			assert.Contains(t, mErr.Msg, tt.wantMsg)
		})
	}
}

func TestConfigValidation_InvalidRanges(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
	}{
		{
			name:   "negative MaxFileSize",
			modify: func(c *Config) { c.MaxFileSize = -1 },
		},
		{
			name:   "zero MaxFileSize",
			modify: func(c *Config) { c.MaxFileSize = 0 },
		},
		{
			name:   "zero MinWidth",
			modify: func(c *Config) { c.MinWidth = 0 },
		},
		{
			name:   "negative MinHeight",
			modify: func(c *Config) { c.MinHeight = -1 },
		},
		{
			name:   "zero MaxWidth",
			modify: func(c *Config) { c.MaxWidth = 0 },
		},
		{
			name:   "MinWidth >= MaxWidth",
			modify: func(c *Config) { c.MinWidth = 8000; c.MaxWidth = 8000 },
		},
		{
			name:   "MinHeight >= MaxHeight",
			modify: func(c *Config) { c.MinHeight = 9000; c.MaxHeight = 8000 },
		},
		{
			name:   "negative DefaultTokenTTL",
			modify: func(c *Config) { c.DefaultTokenTTL = -time.Second },
		},
		{
			name:   "negative PendingTTL",
			modify: func(c *Config) { c.PendingTTL = -time.Second },
		},
		{
			name:   "negative MaxRetries",
			modify: func(c *Config) { c.MaxRetries = intPtr(-1) },
		},
		{
			name: "variant quality > 100",
			modify: func(c *Config) {
				c.Variants = []VariantConfig{{Name: "bad", MaxWidth: 100, MaxHeight: 100, Quality: 101}}
			},
		},
		{
			name: "variant quality < 1",
			modify: func(c *Config) {
				c.Variants = []VariantConfig{{Name: "bad", MaxWidth: 100, MaxHeight: 100, Quality: 0}}
			},
		},
		{
			name: "variant zero dimensions",
			modify: func(c *Config) {
				c.Variants = []VariantConfig{{Name: "bad", MaxWidth: 0, MaxHeight: 100, Quality: 80}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.applyDefaults()
			tt.modify(&cfg)
			err := cfg.validate()

			require.Error(t, err)
			var mErr *Error
			require.True(t, errors.As(err, &mErr))
			assert.Equal(t, CategoryConfig, mErr.Category)
		})
	}
}

func TestConfigValidation_DuplicateVariantNames(t *testing.T) {
	cfg := testConfig()
	cfg.applyDefaults()
	cfg.Variants = []VariantConfig{
		{Name: "thumb", MaxWidth: 150, MaxHeight: 150, Quality: 80},
		{Name: "thumb", MaxWidth: 300, MaxHeight: 300, Quality: 85},
	}
	err := cfg.validate()

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Contains(t, mErr.Msg, "duplicate variant name")
}

func TestConfigValidation_EmptyVariantName(t *testing.T) {
	cfg := testConfig()
	cfg.applyDefaults()
	cfg.Variants = []VariantConfig{
		{Name: "", MaxWidth: 150, MaxHeight: 150, Quality: 80},
	}
	err := cfg.validate()

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Contains(t, mErr.Msg, "variant name must not be empty")
}

func TestConfigValidation_InvalidMIMEType(t *testing.T) {
	cfg := testConfig()
	cfg.applyDefaults()
	cfg.AllowedMIMETypes = []string{"image/jpeg", "application/pdf"}
	err := cfg.validate()

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Contains(t, mErr.Msg, "image/")
}

func TestConfigDefaults_Applied(t *testing.T) {
	cfg := testConfig()
	cfg.applyDefaults()

	assert.Equal(t, defaultCWebPPath, cfg.CWebPPath)
	assert.Equal(t, DefaultVariants(), cfg.Variants)
	assert.Equal(t, defaultTokenTTL, cfg.DefaultTokenTTL)
	assert.Equal(t, defaultPendingTTL, cfg.PendingTTL)
	assert.Equal(t, int64(defaultMaxFileSize), cfg.MaxFileSize)
	assert.Equal(t, []string{"image/jpeg", "image/png", "image/webp"}, cfg.AllowedMIMETypes)
	assert.Equal(t, defaultMinWidth, cfg.MinWidth)
	assert.Equal(t, defaultMinHeight, cfg.MinHeight)
	assert.Equal(t, defaultMaxWidth, cfg.MaxWidth)
	assert.Equal(t, defaultMaxHeight, cfg.MaxHeight)
	assert.Equal(t, defaultUploadTimeout, cfg.UploadTimeout)
	assert.Equal(t, defaultDeleteTimeout, cfg.DeleteTimeout)
	assert.Equal(t, defaultMaxRetries, *cfg.MaxRetries)
	assert.Equal(t, defaultRetryBaseDelay, cfg.RetryBaseDelay)
}

func TestConfigDefaults_CustomValuesPreserved(t *testing.T) {
	cfg := testConfig()
	cfg.CWebPPath = "/usr/local/bin/cwebp"
	cfg.MaxFileSize = 5_000_000
	cfg.MinWidth = 50
	cfg.MaxWidth = 4000
	cfg.MinHeight = 50
	cfg.MaxHeight = 4000
	cfg.MaxRetries = intPtr(5)
	cfg.DefaultTokenTTL = 30 * time.Minute
	cfg.PendingTTL = 4 * time.Hour
	cfg.UploadTimeout = 120 * time.Second
	cfg.DeleteTimeout = 30 * time.Second
	cfg.RetryBaseDelay = 2 * time.Second
	cfg.AllowedMIMETypes = []string{"image/png"}
	cfg.Variants = []VariantConfig{
		{Name: "small", MaxWidth: 200, MaxHeight: 200, Quality: 75},
	}

	cfg.applyDefaults()

	assert.Equal(t, "/usr/local/bin/cwebp", cfg.CWebPPath)
	assert.Equal(t, int64(5_000_000), cfg.MaxFileSize)
	assert.Equal(t, 50, cfg.MinWidth)
	assert.Equal(t, 4000, cfg.MaxWidth)
	assert.Equal(t, 50, cfg.MinHeight)
	assert.Equal(t, 4000, cfg.MaxHeight)
	assert.Equal(t, 5, *cfg.MaxRetries)
	assert.Equal(t, 30*time.Minute, cfg.DefaultTokenTTL)
	assert.Equal(t, 4*time.Hour, cfg.PendingTTL)
	assert.Equal(t, 120*time.Second, cfg.UploadTimeout)
	assert.Equal(t, 30*time.Second, cfg.DeleteTimeout)
	assert.Equal(t, 2*time.Second, cfg.RetryBaseDelay)
	assert.Equal(t, []string{"image/png"}, cfg.AllowedMIMETypes)
	assert.Len(t, cfg.Variants, 1)
	assert.Equal(t, "small", cfg.Variants[0].Name)
}

func TestConfigValidation_ValidComplete(t *testing.T) {
	cfg := testConfig()
	cfg.applyDefaults()
	err := cfg.validate()
	assert.NoError(t, err)
}

func TestConfigDefaults_CDNBaseURLTrailingSlash(t *testing.T) {
	cfg := testConfig()
	cfg.CDNBaseURL = "https://cdn.example.com/"
	cfg.applyDefaults()

	assert.Equal(t, "https://cdn.example.com", cfg.CDNBaseURL)
}

func TestConfigDefaults_CDNBaseURLMultipleTrailingSlashes(t *testing.T) {
	cfg := testConfig()
	cfg.CDNBaseURL = "https://cdn.example.com///"
	cfg.applyDefaults()

	assert.Equal(t, "https://cdn.example.com", cfg.CDNBaseURL)
}

func TestConfigDefaults_MaxRetriesZeroPreserved(t *testing.T) {
	cfg := testConfig()
	cfg.MaxRetries = intPtr(0)
	cfg.applyDefaults()
	require.NotNil(t, cfg.MaxRetries)
	assert.Equal(t, 0, *cfg.MaxRetries)
}

func TestDefaultVariants(t *testing.T) {
	variants := DefaultVariants()
	require.Len(t, variants, 3)

	assert.Equal(t, "thumb", variants[0].Name)
	assert.Equal(t, 150, variants[0].MaxWidth)
	assert.Equal(t, 150, variants[0].MaxHeight)
	assert.Equal(t, 80, variants[0].Quality)

	assert.Equal(t, "medium", variants[1].Name)
	assert.Equal(t, 400, variants[1].MaxWidth)
	assert.Equal(t, 400, variants[1].MaxHeight)
	assert.Equal(t, 85, variants[1].Quality)

	assert.Equal(t, "large", variants[2].Name)
	assert.Equal(t, 800, variants[2].MaxWidth)
	assert.Equal(t, 800, variants[2].MaxHeight)
	assert.Equal(t, 90, variants[2].Quality)
}
