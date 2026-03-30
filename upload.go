package mediaforge

import (
	"bytes"
	"context"
	"io"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/Ibtkary/mediaforge/internal/pathutil"
	"github.com/Ibtkary/mediaforge/internal/processing"
	"github.com/Ibtkary/mediaforge/internal/storage"
)

// cleanupContext returns a fresh background context for best-effort cleanup operations.
// The upload context must not be used for cleanup because it may already be cancelled
// (e.g. when the upload itself fails due to context cancellation).
func cleanupContext() context.Context {
	return context.Background()
}

// UploadReader processes and uploads an image from an io.Reader for a tenant.
// It reads the entire image into memory (required for decoding), then processes
// and uploads. For the []byte variant, see Upload.
//
// The read is capped at MaxFileSize+1 bytes; if the stream exceeds the limit,
// a validation error is returned without reading the entire body.
func (c *Client) UploadReader(ctx context.Context, tenantID string, r io.Reader) (*PendingAsset, error) {
	const op = "UploadReader"
	// Read one byte beyond the limit so we can detect oversized streams early,
	// before the full validation pass inside Upload.
	limited := io.LimitReader(r, c.config.MaxFileSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, ErrValidation("failed to read image data").WithOp(op).WithCause(err)
	}
	if int64(len(data)) > c.config.MaxFileSize {
		return nil, ErrValidation("image exceeds maximum allowed file size").WithOp(op)
	}
	return c.Upload(ctx, tenantID, data)
}

// Upload processes and uploads an image for a tenant.
// It validates, hashes, decodes, resizes, encodes, and uploads
// the original plus all configured variants to storage.
//
// Behavior modifiers:
//   - WithDedup(true): checks if content already exists before uploading
//   - WithLazyVariants(true): uploads original only, skips variants
//   - WithSmartVariantSkip(true): skips variants where source is already smaller
func (c *Client) Upload(ctx context.Context, tenantID string, data []byte) (*PendingAsset, error) {
	const op = "Upload"

	// 1. Sanitize tenant ID
	tenantID, err := pathutil.SanitizeTenantID(tenantID)
	if err != nil {
		return nil, ErrValidation("invalid tenant ID").WithOp(op).WithCause(err)
	}

	// 2. Process image (validate → hash → decode → resize → encode)
	result, err := processing.Process(data, c.buildValidateConfig(), c.buildProcessConfig())
	if err != nil {
		return nil, classifyProcessingError(err, op)
	}

	// 3. Generate filename
	hashPrefix := processing.HashPrefix(result.ContentHash, 16)
	uuidStr := strings.ReplaceAll(uuid.New().String(), "-", "")
	uuidPrefix := uuidStr[:8]
	baseFilename := pathutil.BuildOriginalFilename(hashPrefix, uuidPrefix)

	// 4. Build original storage path
	originalPath := pathutil.BuildStoragePath(tenantID, baseFilename)

	// 5. Dedup check: if enabled, check if content already exists
	retryCfg := c.buildRetryConfig()
	if c.config.Dedup {
		exists, existsErr := c.obj.Exists(ctx, originalPath)
		if existsErr == nil && exists {
			c.logger.Info("dedup: content already exists, skipping upload", "path", originalPath, "hash", result.ContentHash)
			now := c.clock.Now()
			return &PendingAsset{
				StoragePath:   originalPath,
				ContentHash:   result.ContentHash,
				FileSizeBytes: int64(len(result.OriginalWebP)),
				OriginalMIME:  result.OriginalMIME,
				Width:         result.Width,
				Height:        result.Height,
				Variants:      make(map[string]VariantInfo),
				SessionID:     uuid.New().String(),
				TenantID:      tenantID,
				UploadedAt:    now,
				ExpiresAt:     now.Add(c.config.PendingTTL),
			}, nil
		}
		// If Exists failed, log and continue with normal upload
		if existsErr != nil {
			c.logger.Warn("dedup: exists check failed, proceeding with upload", "error", existsErr)
		}
	}

	// 6. Upload original with retry
	origSize := int64(len(result.OriginalWebP))
	c.logger.Info("uploading original", "path", originalPath, "size", origSize, "tenant", tenantID)
	c.hook.BeforeUpload(ctx, originalPath, origSize)
	uploadStart := c.clock.Now()
	err = storage.WithRetry(ctx, retryCfg, c.shouldRetry, func() error {
		return c.obj.Upload(ctx, originalPath, bytes.NewReader(result.OriginalWebP), origSize)
	})
	c.hook.AfterUpload(ctx, originalPath, origSize, c.clock.Now().Sub(uploadStart), err)
	if err != nil {
		c.logger.Error("upload original failed", "path", originalPath, "error", err)
		return nil, wrapStorageError(err, op, "uploading original")
	}

	// 7. Upload variants (unless lazy mode is enabled)
	variants := make(map[string]VariantInfo, len(result.Variants))
	var uploadedVariantPaths []string

	if !c.config.LazyVariants {
		// Sort variant names for deterministic upload order and log output.
		variantNames := make([]string, 0, len(result.Variants))
		for name := range result.Variants {
			variantNames = append(variantNames, name)
		}
		sort.Strings(variantNames)

		for _, name := range variantNames {
			vr := result.Variants[name]
			// Smart variant skip: if enabled and the source image fits within the variant's max
			// dimensions, the variant is identical to the original — skip it.
			if c.config.SmartVariantSkip {
				variantCfg := c.findVariantConfig(name)
				if variantCfg != nil && result.Width <= variantCfg.MaxWidth && result.Height <= variantCfg.MaxHeight {
					c.logger.Info("skipping variant (source fits within variant bounds)", "variant", name,
						"srcWidth", result.Width, "srcHeight", result.Height,
						"maxWidth", variantCfg.MaxWidth, "maxHeight", variantCfg.MaxHeight)
					continue
				}
			}

			variantFilename := pathutil.BuildVariantFilename(baseFilename, name)
			variantPath := pathutil.BuildStoragePath(tenantID, variantFilename)
			variantSize := int64(len(vr.Data))

			c.logger.Info("uploading variant", "path", variantPath, "variant", name, "size", variantSize)
			c.hook.BeforeUpload(ctx, variantPath, variantSize)
			vStart := c.clock.Now()
			err = storage.WithRetry(ctx, retryCfg, c.shouldRetry, func() error {
				return c.obj.Upload(ctx, variantPath, bytes.NewReader(vr.Data), variantSize)
			})
			c.hook.AfterUpload(ctx, variantPath, variantSize, c.clock.Now().Sub(vStart), err)
			if err != nil {
				c.logger.Error("upload variant failed", "path", variantPath, "variant", name, "error", err)
				c.cleanupPartialUpload(cleanupContext(), originalPath, uploadedVariantPaths, retryCfg)
				return nil, wrapStorageError(err, op, "uploading variant "+name)
			}

			uploadedVariantPaths = append(uploadedVariantPaths, variantPath)
			variants[name] = VariantInfo{
				Path:      variantPath,
				Width:     vr.Width,
				Height:    vr.Height,
				SizeBytes: variantSize,
			}
		}
	} else {
		c.logger.Info("lazy variants enabled, skipping variant uploads", "path", originalPath)
	}

	// 8. Build PendingAsset
	now := c.clock.Now()
	return &PendingAsset{
		StoragePath:   originalPath,
		ContentHash:   result.ContentHash,
		FileSizeBytes: int64(len(result.OriginalWebP)),
		OriginalMIME:  result.OriginalMIME,
		Width:         result.Width,
		Height:        result.Height,
		Variants:      variants,
		SessionID:     uuid.New().String(),
		TenantID:      tenantID,
		UploadedAt:    now,
		ExpiresAt:     now.Add(c.config.PendingTTL),
	}, nil
}

// findVariantConfig looks up a variant config by name.
func (c *Client) findVariantConfig(name string) *VariantConfig {
	for i := range c.config.Variants {
		if c.config.Variants[i].Name == name {
			return &c.config.Variants[i]
		}
	}
	return nil
}

// shouldRetry checks if an error should be retried.
func (c *Client) shouldRetry(err error) bool {
	if isRetryableError(err) {
		return true
	}
	return storage.IsRetryableStorageError(err)
}

// classifyProcessingError maps internal processing errors to public error categories.
func classifyProcessingError(err error, op string) *Error {
	msg := err.Error()
	if strings.HasPrefix(msg, "validation failed:") {
		return ErrValidation("image validation failed").WithOp(op).WithCause(err)
	}
	return ErrProcessing("image processing failed").WithOp(op).WithCause(err)
}

// wrapStorageError wraps storage-layer errors into public *Error with retryable flag.
func wrapStorageError(err error, op, detail string) *Error {
	retryable := isRetryableError(err) || storage.IsRetryableStorageError(err)
	return ErrStorage("storage operation failed", retryable).
		WithOp(op).WithDetail(detail).WithCause(err)
}

// cleanupPartialUpload attempts best-effort deletion of already-uploaded files.
func (c *Client) cleanupPartialUpload(ctx context.Context, originalPath string, variantPaths []string, retryCfg storage.RetryConfig) {
	_ = storage.WithRetry(ctx, retryCfg, c.shouldRetry, func() error {
		return c.obj.Delete(ctx, originalPath)
	})
	for _, vp := range variantPaths {
		_ = storage.WithRetry(ctx, retryCfg, c.shouldRetry, func() error {
			return c.obj.Delete(ctx, vp)
		})
	}
}
