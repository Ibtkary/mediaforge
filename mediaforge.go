package mediaforge

import (
	"image"
	"log/slog"
	"os/exec"
	"time"

	"github.com/Ibtkary/mediaforge/internal/processing"
	"github.com/Ibtkary/mediaforge/internal/signing"
	"github.com/Ibtkary/mediaforge/internal/storage"
)

// Client is the main entry point for the mediastore library.
type Client struct {
	config  Config
	obj     ObjectStorage
	signer  URLSigner
	encoder ImageEncoder
	clock   Clock
	logger  *slog.Logger
	hook    Hook

	// legacy holds the concrete StorageClient for backward-compatible New(Config).
	// This allows setStorageBaseURL to work for existing tests.
	legacy *storage.StorageClient
}

// New creates a new mediastore Client configured for Bunny.net.
// It applies defaults, validates config, and verifies that the cwebp binary exists.
//
// This is the backward-compatible constructor. For provider-agnostic usage, see NewClient.
func New(cfg Config) (*Client, error) {
	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if _, err := exec.LookPath(cfg.CWebPPath); err != nil {
		return nil, ErrConfig("cwebp binary not found at: " + cfg.CWebPPath).
			WithOp("New").
			WithCause(err)
	}

	sc := storage.NewStorageClient(cfg.StorageZone, cfg.StorageAPIKey, cfg.UploadTimeout, cfg.DeleteTimeout)

	return &Client{
		config:  cfg,
		obj:     storage.NewBunnyAdapter(sc),
		signer:  signing.NewBunnySigner(cfg.CDNBaseURL, cfg.CDNSecurityKey),
		encoder: nil, // nil = use default cwebp via pipeline fallback
		clock:   realClock{},
		logger:  discardLogger(),
		hook:    NoopHook{},
		legacy:  sc,
	}, nil
}

// NewClient creates a provider-agnostic mediastore Client.
// The storage and signer are required; use options for everything else.
//
// Example:
//
//	client, err := mediastore.NewClient(myStorage, mySigner,
//	    mediastore.WithVariants(mediastore.DefaultVariants()),
//	    mediastore.WithMaxFileSize(5 * 1024 * 1024),
//	)
func NewClient(storage ObjectStorage, signer URLSigner, opts ...Option) (*Client, error) {
	if storage == nil {
		return nil, ErrConfig("storage must not be nil").WithOp("NewClient")
	}
	if signer == nil {
		return nil, ErrConfig("signer must not be nil").WithOp("NewClient")
	}

	// Collect options
	o := &clientOptions{}
	for _, opt := range opts {
		opt(o)
	}

	// Apply options first, then fill any remaining zero-value fields with defaults.
	// This ensures that an explicit option always wins over the default value.
	cfg := Config{}
	o.applyToConfig(&cfg)
	cfg.applyDefaults()

	// Validate the non-provider fields
	if err := cfg.validateNonProvider(); err != nil {
		return nil, err
	}

	clk := o.clock
	if clk == nil {
		clk = realClock{}
	}
	lgr := o.logger
	if lgr == nil {
		lgr = discardLogger()
	}
	hk := o.hook
	if hk == nil {
		hk = NoopHook{}
	}

	return &Client{
		config:  cfg,
		obj:     storage,
		signer:  signer,
		encoder: o.encoder,
		clock:   clk,
		logger:  lgr,
		hook:    hk,
		legacy:  nil,
	}, nil
}

// NewBunnyClient creates a Client configured for Bunny.net with functional options.
//
// Example:
//
//	client, err := mediastore.NewBunnyClient("zone", "apiKey", "https://cdn.example.com", "cdnKey",
//	    mediastore.WithVariants(mediastore.DefaultVariants()),
//	)
func NewBunnyClient(zone, apiKey, cdnBase, cdnSecurityKey string, opts ...Option) (*Client, error) {
	// Collect options for timeouts
	o := &clientOptions{}
	for _, opt := range opts {
		opt(o)
	}

	// Build config to get timeout defaults
	cfg := Config{
		StorageZone:    zone,
		StorageAPIKey:  apiKey,
		CDNBaseURL:     cdnBase,
		CDNSecurityKey: cdnSecurityKey,
	}
	o.applyToConfig(&cfg)
	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Check cwebp only if no custom encoder
	if o.encoder == nil {
		cwebpPath := cfg.CWebPPath
		if _, err := exec.LookPath(cwebpPath); err != nil {
			return nil, ErrConfig("cwebp binary not found at: " + cwebpPath).
				WithOp("NewBunnyClient").
				WithCause(err)
		}
	}

	sc := storage.NewStorageClient(zone, apiKey, cfg.UploadTimeout, cfg.DeleteTimeout)

	clk := o.clock
	if clk == nil {
		clk = realClock{}
	}
	lgr := o.logger
	if lgr == nil {
		lgr = discardLogger()
	}
	hk := o.hook
	if hk == nil {
		hk = NoopHook{}
	}

	return &Client{
		config:  cfg,
		obj:     storage.NewBunnyAdapter(sc),
		signer:  signing.NewBunnySigner(cdnBase, cdnSecurityKey),
		encoder: o.encoder,
		clock:   clk,
		logger:  lgr,
		hook:    hk,
		legacy:  sc,
	}, nil
}

// setStorageBaseURL overrides the storage base URL (used for testing with httptest).
// Only works for Bunny-backed clients created via New() or NewBunnyClient().
func (c *Client) setStorageBaseURL(url string) {
	if c.legacy != nil {
		c.legacy.SetBaseURL(url)
	}
}

// buildValidateConfig bridges mediastore.Config → processing.ValidateConfig.
func (c *Client) buildValidateConfig() processing.ValidateConfig {
	return processing.ValidateConfig{
		MaxFileSize:      c.config.MaxFileSize,
		AllowedMIMETypes: c.config.AllowedMIMETypes,
		MinWidth:         c.config.MinWidth,
		MinHeight:        c.config.MinHeight,
		MaxWidth:         c.config.MaxWidth,
		MaxHeight:        c.config.MaxHeight,
	}
}

// buildProcessConfig bridges mediastore.Config → processing.ProcessConfig.
func (c *Client) buildProcessConfig() processing.ProcessConfig {
	variants := make([]processing.VariantConfig, len(c.config.Variants))
	for i, v := range c.config.Variants {
		variants[i] = processing.VariantConfig{
			Name:      v.Name,
			MaxWidth:  v.MaxWidth,
			MaxHeight: v.MaxHeight,
			Quality:   v.Quality,
		}
	}
	var enc processing.Encoder
	if c.encoder != nil {
		enc = &publicEncoderAdapter{c.encoder}
	}

	return processing.ProcessConfig{
		Variants:          variants,
		OriginalMaxWidth:  OriginalMaxWidth,
		OriginalMaxHeight: OriginalMaxHeight,
		OriginalQuality:   OriginalQuality,
		CWebPPath:         c.config.CWebPPath,
		Encoder:           enc, // nil = fallback to cwebp in pipeline
	}
}

// publicEncoderAdapter wraps the public ImageEncoder to satisfy internal processing.Encoder.
type publicEncoderAdapter struct {
	enc ImageEncoder
}

func (a *publicEncoderAdapter) Encode(img image.Image, quality int) ([]byte, error) {
	return a.enc.Encode(img, quality)
}
func (a *publicEncoderAdapter) Extension() string   { return a.enc.Extension() }
func (a *publicEncoderAdapter) ContentType() string { return a.enc.ContentType() }

// buildRetryConfig bridges mediastore.Config → storage.RetryConfig.
func (c *Client) buildRetryConfig() storage.RetryConfig {
	maxRetries := 0
	if c.config.MaxRetries != nil {
		maxRetries = *c.config.MaxRetries
	}
	return storage.RetryConfig{
		MaxRetries: maxRetries,
		BaseDelay:  c.config.RetryBaseDelay,
		MaxDelay:   30 * time.Second,
	}
}
