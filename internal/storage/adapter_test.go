package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAdapter(handler http.Handler) (*BunnyAdapter, *httptest.Server) {
	server := httptest.NewServer(handler)
	sc := NewStorageClient("test-zone", "test-api-key", 10*time.Second, 10*time.Second)
	sc.baseURL = server.URL
	return NewBunnyAdapter(sc), server
}

func TestBunnyAdapter_Upload_Success(t *testing.T) {
	var capturedBody []byte
	var capturedPath string

	adapter, server := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	data := []byte("test image data")
	err := adapter.Upload(context.Background(), "stores/t1/images/test.webp", bytes.NewReader(data), int64(len(data)))

	require.NoError(t, err)
	assert.Equal(t, "/test-zone/stores/t1/images/test.webp", capturedPath)
	assert.Equal(t, data, capturedBody)
}

func TestBunnyAdapter_Upload_ServerError(t *testing.T) {
	adapter, server := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	data := []byte("test")
	err := adapter.Upload(context.Background(), "stores/t1/images/test.webp", bytes.NewReader(data), int64(len(data)))

	require.Error(t, err)
	var se *StorageError
	assert.ErrorAs(t, err, &se)
	assert.True(t, se.Retryable)
}

func TestBunnyAdapter_Upload_ReadError(t *testing.T) {
	adapter, server := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	err := adapter.Upload(context.Background(), "key", &failingReader{}, 0)

	require.Error(t, err)
	var se *StorageError
	assert.ErrorAs(t, err, &se)
	assert.False(t, se.Retryable)
}

func TestBunnyAdapter_Delete_Success(t *testing.T) {
	var capturedPath string
	adapter, server := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := adapter.Delete(context.Background(), "stores/t1/images/test.webp")

	require.NoError(t, err)
	assert.Equal(t, "/test-zone/stores/t1/images/test.webp", capturedPath)
}

func TestBunnyAdapter_Delete_NotFound(t *testing.T) {
	adapter, server := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := adapter.Delete(context.Background(), "stores/t1/images/gone.webp")
	require.Error(t, err)
}

func TestBunnyAdapter_Exists_Found(t *testing.T) {
	adapter, server := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exists, err := adapter.Exists(context.Background(), "stores/t1/images/test.webp")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestBunnyAdapter_Exists_NotFound(t *testing.T) {
	adapter, server := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	exists, err := adapter.Exists(context.Background(), "stores/t1/images/test.webp")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestBunnyAdapter_Upload_ContextCancelled(t *testing.T) {
	var calls atomic.Int32
	adapter, server := newTestAdapter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	data := []byte("test")
	err := adapter.Upload(ctx, "key", bytes.NewReader(data), int64(len(data)))
	require.Error(t, err)
}

// failingReader always returns an error on Read.
type failingReader struct{}

func (f *failingReader) Read(p []byte) (int, error) {
	return 0, io.ErrUnexpectedEOF
}
