package processing

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// VariantConfig defines parameters for a single variant.
type VariantConfig struct {
	Name      string
	MaxWidth  int
	MaxHeight int
	Quality   int
}

// ProcessConfig holds pipeline configuration.
type ProcessConfig struct {
	Variants                            []VariantConfig
	OriginalMaxWidth, OriginalMaxHeight int
	OriginalQuality                     int
	CWebPPath                           string
	Encoder                             Encoder // nil = fallback to CWebPEncoder
}

// VariantResult holds the output for a single processed variant.
type VariantResult struct {
	Data          []byte
	Width, Height int
}

// ProcessResult holds the complete output of the processing pipeline.
type ProcessResult struct {
	OriginalWebP  []byte
	Variants      map[string]*VariantResult
	ContentHash   string
	Width, Height int    // original uploaded image dimensions
	OriginalMIME  string
}

// Process runs the full image processing pipeline:
// validate → hash → decode → resize original → encode → resize+encode variants.
func Process(data []byte, valCfg ValidateConfig, procCfg ProcessConfig) (*ProcessResult, error) {
	enc := procCfg.resolveEncoder()

	// 1. Content hash (before any processing)
	contentHash := ContentHash(data)

	// 2. Validate
	imgInfo, err := ValidateImage(data, valCfg)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 3. Decode full image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image decode failed: %w", err)
	}

	// 4. Resize original and encode
	resizedOrig := ResizeImage(img, procCfg.OriginalMaxWidth, procCfg.OriginalMaxHeight)
	originalEncoded, err := enc.Encode(resizedOrig, procCfg.OriginalQuality)
	if err != nil {
		return nil, fmt.Errorf("encoding original: %w", err)
	}

	// 5. Process variants (resize from original img for best quality)
	variants := make(map[string]*VariantResult, len(procCfg.Variants))
	for _, vc := range procCfg.Variants {
		resized := ResizeImage(img, vc.MaxWidth, vc.MaxHeight)
		encoded, err := enc.Encode(resized, vc.Quality)
		if err != nil {
			return nil, fmt.Errorf("encoding variant %q: %w", vc.Name, err)
		}
		variants[vc.Name] = &VariantResult{
			Data:   encoded,
			Width:  resized.Bounds().Dx(),
			Height: resized.Bounds().Dy(),
		}
	}

	return &ProcessResult{
		OriginalWebP: originalEncoded,
		Variants:     variants,
		ContentHash:  contentHash,
		Width:        imgInfo.Width,
		Height:       imgInfo.Height,
		OriginalMIME: imgInfo.MIME,
	}, nil
}

// OriginalResult holds the processed original image plus the decoded image for variant processing.
type OriginalResult struct {
	EncodedData  []byte
	ContentHash  string
	Width        int
	Height       int
	OriginalMIME string
}

// ProcessOriginal runs the first part of the pipeline: validate → hash → decode → resize → encode original.
// It returns the decoded image for use in variant processing.
func ProcessOriginal(data []byte, valCfg ValidateConfig, procCfg ProcessConfig) (*OriginalResult, image.Image, error) {
	enc := procCfg.resolveEncoder()
	contentHash := ContentHash(data)

	imgInfo, err := ValidateImage(data, valCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("validation failed: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("image decode failed: %w", err)
	}

	resizedOrig := ResizeImage(img, procCfg.OriginalMaxWidth, procCfg.OriginalMaxHeight)
	originalEncoded, err := enc.Encode(resizedOrig, procCfg.OriginalQuality)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding original: %w", err)
	}

	return &OriginalResult{
		EncodedData:  originalEncoded,
		ContentHash:  contentHash,
		Width:        imgInfo.Width,
		Height:       imgInfo.Height,
		OriginalMIME: imgInfo.MIME,
	}, img, nil
}

// ProcessVariant processes a single variant from the decoded source image.
// This enables process-upload-release per variant to reduce memory usage.
func ProcessVariant(img image.Image, vc VariantConfig, enc Encoder) (*VariantResult, error) {
	if enc == nil {
		return nil, fmt.Errorf("encoder must not be nil")
	}

	resized := ResizeImage(img, vc.MaxWidth, vc.MaxHeight)
	encoded, err := enc.Encode(resized, vc.Quality)
	if err != nil {
		return nil, fmt.Errorf("encoding variant %q: %w", vc.Name, err)
	}

	return &VariantResult{
		Data:   encoded,
		Width:  resized.Bounds().Dx(),
		Height: resized.Bounds().Dy(),
	}, nil
}

// resolveEncoder returns the configured encoder, or falls back to CWebPEncoder.
func (cfg *ProcessConfig) resolveEncoder() Encoder {
	if cfg.Encoder != nil {
		return cfg.Encoder
	}
	return &CWebPEncoder{BinaryPath: cfg.CWebPPath}
}
