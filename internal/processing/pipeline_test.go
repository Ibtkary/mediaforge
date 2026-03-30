package processing

import (
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultProcessConfig() ProcessConfig {
	return ProcessConfig{
		Variants: []VariantConfig{
			{Name: "thumb", MaxWidth: 150, MaxHeight: 150, Quality: 80},
			{Name: "medium", MaxWidth: 400, MaxHeight: 400, Quality: 85},
			{Name: "large", MaxWidth: 800, MaxHeight: 800, Quality: 90},
		},
		OriginalMaxWidth:  2000,
		OriginalMaxHeight: 2000,
		OriginalQuality:   92,
		CWebPPath:         "cwebp",
	}
}

func TestProcess_FullPipeline(t *testing.T) {
	skipIfNoCWebP(t)

	data := createTestJPEG(1000, 800)
	valCfg := defaultValidateCfg()
	procCfg := defaultProcessConfig()

	result, err := Process(data, valCfg, procCfg)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Hash
	assert.Len(t, result.ContentHash, 64)
	assert.Equal(t, ContentHash(data), result.ContentHash)

	// Original dimensions
	assert.Equal(t, 1000, result.Width)
	assert.Equal(t, 800, result.Height)
	assert.Equal(t, "image/jpeg", result.OriginalMIME)

	// Original WebP
	assert.NotEmpty(t, result.OriginalWebP)
	assert.Equal(t, "RIFF", string(result.OriginalWebP[:4]))

	// All variants produced
	require.Len(t, result.Variants, 3)
	assert.Contains(t, result.Variants, "thumb")
	assert.Contains(t, result.Variants, "medium")
	assert.Contains(t, result.Variants, "large")
}

func TestProcess_ValidationFailure(t *testing.T) {
	skipIfNoCWebP(t)

	data := createTestJPEG(200, 200)
	valCfg := defaultValidateCfg()
	valCfg.MaxFileSize = 10 // too small — will fail

	_, err := Process(data, valCfg, defaultProcessConfig())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestProcess_ContentHashMatches(t *testing.T) {
	skipIfNoCWebP(t)

	data := createTestPNG(300, 200)
	directHash := ContentHash(data)

	result, err := Process(data, defaultValidateCfg(), defaultProcessConfig())

	require.NoError(t, err)
	assert.Equal(t, directHash, result.ContentHash)
}

func TestProcess_VariantDimensions(t *testing.T) {
	skipIfNoCWebP(t)

	data := createTestJPEG(1000, 800)
	result, err := Process(data, defaultValidateCfg(), defaultProcessConfig())

	require.NoError(t, err)

	// Thumb: 1000×800 → max 150×150 → 150×120 (aspect ratio preserved)
	thumb := result.Variants["thumb"]
	require.NotNil(t, thumb)
	assert.Equal(t, 150, thumb.Width)
	assert.Equal(t, 120, thumb.Height)

	// Medium: 1000×800 → max 400×400 → 400×320
	medium := result.Variants["medium"]
	require.NotNil(t, medium)
	assert.Equal(t, 400, medium.Width)
	assert.Equal(t, 320, medium.Height)

	// Large: 1000×800 → max 800×800 → 800×640
	large := result.Variants["large"]
	require.NotNil(t, large)
	assert.Equal(t, 800, large.Width)
	assert.Equal(t, 640, large.Height)
}

func TestProcess_AllVariantsProduced(t *testing.T) {
	skipIfNoCWebP(t)

	data := createTestJPEG(500, 500)
	procCfg := ProcessConfig{
		Variants: []VariantConfig{
			{Name: "a", MaxWidth: 100, MaxHeight: 100, Quality: 80},
			{Name: "b", MaxWidth: 200, MaxHeight: 200, Quality: 85},
		},
		OriginalMaxWidth:  2000,
		OriginalMaxHeight: 2000,
		OriginalQuality:   92,
		CWebPPath:         "cwebp",
	}

	result, err := Process(data, defaultValidateCfg(), procCfg)

	require.NoError(t, err)
	require.Len(t, result.Variants, 2)
	assert.NotEmpty(t, result.Variants["a"].Data)
	assert.NotEmpty(t, result.Variants["b"].Data)
}

func TestProcess_CWebPFailure(t *testing.T) {
	data := createTestJPEG(200, 200)
	procCfg := defaultProcessConfig()
	procCfg.CWebPPath = "/nonexistent/cwebp"

	_, err := Process(data, defaultValidateCfg(), procCfg)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "encoding original")
}

func TestProcess_DecodeFailure(t *testing.T) {
	data := createTruncatedJPEG(200, 200)

	_, err := Process(data, defaultValidateCfg(), defaultProcessConfig())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "image decode failed")
}

func TestProcess_PNGInput(t *testing.T) {
	skipIfNoCWebP(t)

	data := createTestPNG(400, 300)
	result, err := Process(data, defaultValidateCfg(), defaultProcessConfig())

	require.NoError(t, err)
	assert.Equal(t, "image/png", result.OriginalMIME)
	assert.Equal(t, 400, result.Width)
	assert.Equal(t, 300, result.Height)
	assert.NotEmpty(t, result.OriginalWebP)
}

// --- Mock Encoder tests (no cwebp required) ---

type mockTestEncoder struct {
	callCount int
}

func (m *mockTestEncoder) Encode(_ image.Image, quality int) ([]byte, error) {
	m.callCount++
	return []byte("mock-encoded-data"), nil
}
func (m *mockTestEncoder) Extension() string   { return ".webp" }
func (m *mockTestEncoder) ContentType() string { return "image/webp" }

func TestProcess_WithMockEncoder(t *testing.T) {
	enc := &mockTestEncoder{}
	data := createTestJPEG(400, 300)
	cfg := ProcessConfig{
		Variants: []VariantConfig{
			{Name: "thumb", MaxWidth: 150, MaxHeight: 150, Quality: 80},
			{Name: "medium", MaxWidth: 400, MaxHeight: 400, Quality: 85},
		},
		OriginalMaxWidth:  2000,
		OriginalMaxHeight: 2000,
		OriginalQuality:   92,
		Encoder:           enc,
	}

	result, err := Process(data, defaultValidateCfg(), cfg)

	require.NoError(t, err)
	assert.Equal(t, 3, enc.callCount, "should encode original + 2 variants")
	assert.Equal(t, []byte("mock-encoded-data"), result.OriginalWebP)
	assert.Len(t, result.Variants, 2)
	assert.Equal(t, 400, result.Width)
	assert.Equal(t, 300, result.Height)
}

func TestProcess_MockEncoder_NoSkipIfNoCWebP(t *testing.T) {
	// This test runs without cwebp — proving the encoder interface works
	enc := &mockTestEncoder{}
	data := createTestPNG(200, 200)
	cfg := ProcessConfig{
		Variants: []VariantConfig{
			{Name: "small", MaxWidth: 100, MaxHeight: 100, Quality: 80},
		},
		OriginalMaxWidth:  2000,
		OriginalMaxHeight: 2000,
		OriginalQuality:   92,
		Encoder:           enc,
	}

	result, err := Process(data, defaultValidateCfg(), cfg)

	require.NoError(t, err)
	assert.Equal(t, 2, enc.callCount, "should encode original + 1 variant")
	assert.Contains(t, result.Variants, "small")
}

func TestProcess_NilEncoderFallsToCWebP(t *testing.T) {
	skipIfNoCWebP(t)

	data := createTestJPEG(200, 200)
	cfg := defaultProcessConfig()
	cfg.Encoder = nil // explicitly nil

	result, err := Process(data, defaultValidateCfg(), cfg)

	require.NoError(t, err)
	assert.NotEmpty(t, result.OriginalWebP)
}

func TestCWebPEncoder_ImplementsEncoder(t *testing.T) {
	var _ Encoder = &CWebPEncoder{}
}

// --- ProcessOriginal and ProcessVariant tests ---

func TestProcessOriginal_Success(t *testing.T) {
	enc := &mockTestEncoder{}
	data := createTestJPEG(400, 300)
	cfg := ProcessConfig{
		OriginalMaxWidth:  2000,
		OriginalMaxHeight: 2000,
		OriginalQuality:   92,
		Encoder:           enc,
	}

	result, img, err := ProcessOriginal(data, defaultValidateCfg(), cfg)

	require.NoError(t, err)
	assert.NotNil(t, img)
	assert.Equal(t, 400, result.Width)
	assert.Equal(t, 300, result.Height)
	assert.NotEmpty(t, result.EncodedData)
	assert.NotEmpty(t, result.ContentHash)
	assert.Equal(t, "image/jpeg", result.OriginalMIME)
}

func TestProcessOriginal_ValidationFailure(t *testing.T) {
	enc := &mockTestEncoder{}
	data := createTestJPEG(200, 200)
	valCfg := defaultValidateCfg()
	valCfg.MaxFileSize = 1 // too small

	_, _, err := ProcessOriginal(data, valCfg, ProcessConfig{Encoder: enc})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed")
}

func TestProcessVariant_Success(t *testing.T) {
	enc := &mockTestEncoder{}
	img := createTestImage(400, 300)
	vc := VariantConfig{Name: "thumb", MaxWidth: 150, MaxHeight: 150, Quality: 80}

	result, err := ProcessVariant(img, vc, enc)

	require.NoError(t, err)
	assert.NotEmpty(t, result.Data)
	assert.True(t, result.Width <= 150)
	assert.True(t, result.Height <= 150)
}

func TestProcessVariant_NilEncoder(t *testing.T) {
	img := createTestImage(400, 300)
	vc := VariantConfig{Name: "thumb", MaxWidth: 150, MaxHeight: 150, Quality: 80}

	_, err := ProcessVariant(img, vc, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encoder must not be nil")
}
