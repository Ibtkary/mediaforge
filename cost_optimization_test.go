package mediaforge

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDedup_SkipsUploadWhenExists(t *testing.T) {
	var uploadCount atomic.Int32

	ms := &mockStorage{
		existsFn: func(_ context.Context, key string) (bool, error) {
			return true, nil // file already exists
		},
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ int64) error {
			uploadCount.Add(1)
			return nil
		},
	}

	client, err := NewClient(ms, &mockSigner{},
		WithDedup(true),
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	data := createTestJPEG(200, 200)
	asset, err := client.Upload(context.Background(), "tenant1", data)
	require.NoError(t, err)
	assert.NotNil(t, asset)

	// No upload calls should have been made
	assert.Equal(t, int32(0), uploadCount.Load(), "dedup should skip all uploads")
	assert.Empty(t, asset.Variants, "dedup asset should have no variants")
}

func TestDedup_UploadsWhenNotExists(t *testing.T) {
	var uploadCount atomic.Int32

	ms := &mockStorage{
		existsFn: func(_ context.Context, key string) (bool, error) {
			return false, nil // file does not exist
		},
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ int64) error {
			uploadCount.Add(1)
			return nil
		},
	}

	client, err := NewClient(ms, &mockSigner{},
		WithDedup(true),
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	data := createTestJPEG(200, 200)
	_, err = client.Upload(context.Background(), "tenant1", data)
	require.NoError(t, err)

	// Should upload original + 3 variants = 4
	assert.Equal(t, int32(4), uploadCount.Load(), "should upload all files when not exists")
}

func TestDedup_DisabledByDefault(t *testing.T) {
	var existsCount atomic.Int32

	ms := &mockStorage{
		existsFn: func(_ context.Context, _ string) (bool, error) {
			existsCount.Add(1)
			return true, nil
		},
	}

	client, err := NewClient(ms, &mockSigner{},
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	data := createTestJPEG(200, 200)
	_, err = client.Upload(context.Background(), "tenant1", data)
	require.NoError(t, err)

	// Exists should never be called when dedup is disabled
	assert.Equal(t, int32(0), existsCount.Load(), "exists should not be called when dedup is off")
}

func TestLazyVariants_UploadsOnlyOriginal(t *testing.T) {
	var uploadCount atomic.Int32
	var uploadedKeys []string

	ms := &mockStorage{
		uploadFn: func(_ context.Context, key string, _ io.Reader, _ int64) error {
			uploadCount.Add(1)
			uploadedKeys = append(uploadedKeys, key)
			return nil
		},
	}

	client, err := NewClient(ms, &mockSigner{},
		WithLazyVariants(true),
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	data := createTestJPEG(200, 200)
	asset, err := client.Upload(context.Background(), "tenant1", data)
	require.NoError(t, err)

	// Only 1 upload (original)
	assert.Equal(t, int32(1), uploadCount.Load(), "lazy variants should upload only original")
	assert.Empty(t, asset.Variants, "lazy variants should have no variants")
}

func TestSmartVariantSkip_SkipsWhenSmaller(t *testing.T) {
	var uploadedKeys []string

	ms := &mockStorage{
		uploadFn: func(_ context.Context, key string, _ io.Reader, _ int64) error {
			uploadedKeys = append(uploadedKeys, key)
			return nil
		},
	}

	// 100x100 image: fits in thumb (150), medium (400), large (800) — all should be skipped
	client, err := NewClient(ms, &mockSigner{},
		WithSmartVariantSkip(true),
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	data := createTestJPEG(100, 100)
	asset, err := client.Upload(context.Background(), "tenant1", data)
	require.NoError(t, err)

	// Only original uploaded — all variants skipped
	assert.Len(t, uploadedKeys, 1, "only original should be uploaded")
	assert.Empty(t, asset.Variants, "all variants should be skipped for small image")
}

func TestSmartVariantSkip_UploadsLargerVariants(t *testing.T) {
	var uploadedKeys []string

	ms := &mockStorage{
		uploadFn: func(_ context.Context, key string, _ io.Reader, _ int64) error {
			uploadedKeys = append(uploadedKeys, key)
			return nil
		},
	}

	// 1000x800 image: larger than thumb (150) and medium (400), fits in large (800)
	client, err := NewClient(ms, &mockSigner{},
		WithSmartVariantSkip(true),
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	data := createTestJPEG(1000, 800)
	asset, err := client.Upload(context.Background(), "tenant1", data)
	require.NoError(t, err)

	// original + thumb + medium = 3 (large skipped because 1000>800 but 800 <= 800 is false for width)
	// Actually 1000 > 800, so large is NOT skipped. thumb: 1000>150, medium: 1000>400, large: 1000>800
	// All variants should be uploaded
	assert.Len(t, uploadedKeys, 4, "all variants should be uploaded for large image")
	assert.Len(t, asset.Variants, 3)
}

func TestSmartVariantSkip_DisabledByDefault(t *testing.T) {
	var uploadCount atomic.Int32

	ms := &mockStorage{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ int64) error {
			uploadCount.Add(1)
			return nil
		},
	}

	// 100x100 image with default settings (no smart skip)
	client, err := NewClient(ms, &mockSigner{},
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	data := createTestJPEG(100, 100)
	_, err = client.Upload(context.Background(), "tenant1", data)
	require.NoError(t, err)

	// All 4 uploads (original + 3 variants) even though image is small
	assert.Equal(t, int32(4), uploadCount.Load(), "without smart skip, all variants should be uploaded")
}

func TestDedup_ExistsError_ContinuesUpload(t *testing.T) {
	var uploadCount atomic.Int32

	ms := &mockStorage{
		existsFn: func(_ context.Context, _ string) (bool, error) {
			return false, assert.AnError // exists check fails
		},
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ int64) error {
			uploadCount.Add(1)
			return nil
		},
	}

	client, err := NewClient(ms, &mockSigner{},
		WithDedup(true),
		WithEncoder(&mockEncoder{}),
		WithClock(fixedClock{t: time.Now()}),
	)
	require.NoError(t, err)

	data := createTestJPEG(200, 200)
	_, err = client.Upload(context.Background(), "tenant1", data)
	require.NoError(t, err)

	// Should fall through to normal upload
	assert.Equal(t, int32(4), uploadCount.Load(), "should proceed with upload when exists check fails")
}
