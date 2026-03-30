package mediaforge

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadReader_Success(t *testing.T) {
	var uploadCount atomic.Int32

	ms := &mockStorage{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ int64) error {
			uploadCount.Add(1)
			return nil
		},
	}

	client, err := NewClient(ms, &mockSigner{},
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	data := createTestJPEG(200, 200)
	asset, err := client.UploadReader(context.Background(), "tenant1", bytes.NewReader(data))
	require.NoError(t, err)
	assert.NotNil(t, asset)
	assert.Equal(t, int32(4), uploadCount.Load(), "should upload original + 3 variants")
}

func TestUploadReader_ReadError(t *testing.T) {
	ms := &mockStorage{}

	client, err := NewClient(ms, &mockSigner{},
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	_, err = client.UploadReader(context.Background(), "tenant1", &failReader{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read image data")
}

func TestUploadReader_ProducesValidAsset(t *testing.T) {
	ms := &mockStorage{}

	client, err := NewClient(ms, &mockSigner{},
		WithEncoder(&mockEncoder{}),
	)
	require.NoError(t, err)

	data := createTestJPEG(300, 200)
	asset, err := client.UploadReader(context.Background(), "tenant1", bytes.NewReader(data))
	require.NoError(t, err)

	assert.Equal(t, "tenant1", asset.TenantID)
	assert.Equal(t, 300, asset.Width)
	assert.Equal(t, 200, asset.Height)
	assert.NotEmpty(t, asset.ContentHash)
	assert.NotEmpty(t, asset.SessionID)
	assert.True(t, strings.HasSuffix(asset.StoragePath, ".webp"))
}

// failReader always returns an error on Read.
type failReader struct{}

func (f *failReader) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
