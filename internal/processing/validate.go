package processing

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	_ "golang.org/x/image/webp"
)

// ValidateConfig holds parameters for image validation.
type ValidateConfig struct {
	MaxFileSize                        int64
	AllowedMIMETypes                   []string
	MinWidth, MinHeight                int
	MaxWidth, MaxHeight                int
}

// ImageInfo contains metadata about a validated image.
type ImageInfo struct {
	Width, Height int
	MIME          string // "image/jpeg", etc.
	Format        string // "jpeg", "png", "webp"
}

// ValidateImage validates image data against the given config.
// It checks file size, MIME type, decodability, and dimensions.
func ValidateImage(data []byte, cfg ValidateConfig) (*ImageInfo, error) {
	// 1. File size check
	if int64(len(data)) > cfg.MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d", len(data), cfg.MaxFileSize)
	}

	// 2. MIME type detection
	mime := http.DetectContentType(data)
	if !isAllowedMIME(mime, cfg.AllowedMIMETypes) {
		return nil, fmt.Errorf("MIME type %q is not allowed", mime)
	}

	// 3. Decode image config to get dimensions
	imgCfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// 4. Dimension checks
	if imgCfg.Width < cfg.MinWidth || imgCfg.Height < cfg.MinHeight {
		return nil, fmt.Errorf("image dimensions %dx%d below minimum %dx%d", imgCfg.Width, imgCfg.Height, cfg.MinWidth, cfg.MinHeight)
	}
	if imgCfg.Width > cfg.MaxWidth || imgCfg.Height > cfg.MaxHeight {
		return nil, fmt.Errorf("image dimensions %dx%d exceed maximum %dx%d", imgCfg.Width, imgCfg.Height, cfg.MaxWidth, cfg.MaxHeight)
	}

	return &ImageInfo{
		Width:  imgCfg.Width,
		Height: imgCfg.Height,
		MIME:   mime,
		Format: format,
	}, nil
}

func isAllowedMIME(mime string, allowed []string) bool {
	for _, a := range allowed {
		if a == mime {
			return true
		}
	}
	return false
}
