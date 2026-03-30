package mediaforge

import (
	"strings"
	"time"
)

const (
	defaultMaxFileSize    = 10_485_760 // 10MB
	defaultMinWidth       = 100
	defaultMinHeight      = 100
	defaultMaxWidth       = 8000
	defaultMaxHeight      = 8000
	defaultMaxRetries     = 3
	defaultUploadTimeout  = 60 * time.Second
	defaultDeleteTimeout  = 10 * time.Second
	defaultTokenTTL       = 1 * time.Hour
	defaultPendingTTL     = 2 * time.Hour
	defaultRetryBaseDelay = 1 * time.Second
	defaultCWebPPath      = "cwebp"

	// Original image constants (hardcoded, not a variant).
	OriginalMaxWidth  = 2000
	OriginalMaxHeight = 2000
	OriginalQuality   = 92
)

// VariantConfig defines dimensions and quality for a named image variant.
type VariantConfig struct {
	Name      string `json:"name"`
	MaxWidth  int    `json:"max_width"`
	MaxHeight int    `json:"max_height"`
	Quality   int    `json:"quality"`
}

// Config holds all configuration for the mediastore client.
type Config struct {
	// Required
	StorageZone    string
	StorageAPIKey  string
	CDNBaseURL     string
	CDNSecurityKey string

	// Optional with defaults
	CWebPPath        string
	Variants         []VariantConfig
	DefaultTokenTTL  time.Duration
	PendingTTL       time.Duration
	MaxFileSize      int64
	AllowedMIMETypes []string
	MinWidth         int
	MinHeight        int
	MaxWidth         int
	MaxHeight        int
	UploadTimeout    time.Duration
	DeleteTimeout    time.Duration
	MaxRetries       *int // nil = use default (3), pointer to allow explicit 0
	RetryBaseDelay   time.Duration
	Dedup            bool // Check content hash before upload to avoid duplicates
	LazyVariants     bool // Upload original only; generate variants on demand
	SmartVariantSkip bool // Skip variants when source image fits within variant bounds
}

// DefaultVariants returns the standard thumb/medium/large variant configs.
func DefaultVariants() []VariantConfig {
	return []VariantConfig{
		{Name: "thumb", MaxWidth: 150, MaxHeight: 150, Quality: 80},
		{Name: "medium", MaxWidth: 400, MaxHeight: 400, Quality: 85},
		{Name: "large", MaxWidth: 800, MaxHeight: 800, Quality: 90},
	}
}

func (c *Config) applyDefaults() {
	if c.CWebPPath == "" {
		c.CWebPPath = defaultCWebPPath
	}
	if len(c.Variants) == 0 {
		c.Variants = DefaultVariants()
	}
	if c.DefaultTokenTTL == 0 {
		c.DefaultTokenTTL = defaultTokenTTL
	}
	if c.PendingTTL == 0 {
		c.PendingTTL = defaultPendingTTL
	}
	if c.MaxFileSize == 0 {
		c.MaxFileSize = defaultMaxFileSize
	}
	if len(c.AllowedMIMETypes) == 0 {
		c.AllowedMIMETypes = []string{"image/jpeg", "image/png", "image/webp"}
	}
	if c.MinWidth == 0 {
		c.MinWidth = defaultMinWidth
	}
	if c.MinHeight == 0 {
		c.MinHeight = defaultMinHeight
	}
	if c.MaxWidth == 0 {
		c.MaxWidth = defaultMaxWidth
	}
	if c.MaxHeight == 0 {
		c.MaxHeight = defaultMaxHeight
	}
	if c.UploadTimeout == 0 {
		c.UploadTimeout = defaultUploadTimeout
	}
	if c.DeleteTimeout == 0 {
		c.DeleteTimeout = defaultDeleteTimeout
	}
	if c.MaxRetries == nil {
		c.MaxRetries = intPtr(defaultMaxRetries)
	}
	if c.RetryBaseDelay == 0 {
		c.RetryBaseDelay = defaultRetryBaseDelay
	}

	c.CDNBaseURL = strings.TrimRight(c.CDNBaseURL, "/")
}

func (c *Config) validate() error {
	if c.StorageZone == "" {
		return ErrConfig("StorageZone is required")
	}
	if c.StorageAPIKey == "" {
		return ErrConfig("StorageAPIKey is required")
	}
	if c.CDNBaseURL == "" {
		return ErrConfig("CDNBaseURL is required")
	}
	if c.CDNSecurityKey == "" {
		return ErrConfig("CDNSecurityKey is required")
	}
	return c.validateNonProvider()
}

// validateNonProvider validates config fields that are not provider-specific.
// Used by NewClient() where StorageZone/APIKey/CDN fields are not relevant.
func (c *Config) validateNonProvider() error {
	if c.MaxFileSize <= 0 {
		return ErrConfig("MaxFileSize must be positive")
	}
	if c.MinWidth <= 0 || c.MinHeight <= 0 {
		return ErrConfig("MinWidth and MinHeight must be positive")
	}
	if c.MaxWidth <= 0 || c.MaxHeight <= 0 {
		return ErrConfig("MaxWidth and MaxHeight must be positive")
	}
	if c.MinWidth >= c.MaxWidth {
		return ErrConfig("MinWidth must be less than MaxWidth")
	}
	if c.MinHeight >= c.MaxHeight {
		return ErrConfig("MinHeight must be less than MaxHeight")
	}
	if c.DefaultTokenTTL <= 0 {
		return ErrConfig("DefaultTokenTTL must be positive")
	}
	if c.PendingTTL <= 0 {
		return ErrConfig("PendingTTL must be positive")
	}
	if c.MaxRetries != nil && *c.MaxRetries < 0 {
		return ErrConfig("MaxRetries must be non-negative")
	}

	// Validate variants
	seen := make(map[string]bool, len(c.Variants))
	for _, v := range c.Variants {
		if v.Name == "" {
			return ErrConfig("variant name must not be empty")
		}
		if seen[v.Name] {
			return ErrConfig("duplicate variant name: " + v.Name)
		}
		seen[v.Name] = true
		if v.Quality < 1 || v.Quality > 100 {
			return ErrConfig("variant quality must be 1-100: " + v.Name)
		}
		if v.MaxWidth <= 0 || v.MaxHeight <= 0 {
			return ErrConfig("variant dimensions must be positive: " + v.Name)
		}
	}

	// Validate MIME types
	for _, mime := range c.AllowedMIMETypes {
		if !strings.HasPrefix(mime, "image/") {
			return ErrConfig("allowed MIME type must start with image/: " + mime)
		}
	}

	return nil
}

func intPtr(v int) *int { return &v }
