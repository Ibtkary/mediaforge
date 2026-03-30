package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(handler http.Handler) (*StorageClient, *httptest.Server) {
	server := httptest.NewServer(handler)
	sc := NewStorageClient("test-zone", "test-api-key", 10*time.Second, 10*time.Second)
	sc.baseURL = server.URL
	return sc, server
}

// --- Upload Tests ---

func TestUpload_Success(t *testing.T) {
	var capturedPath, capturedKey, capturedContentType string
	var capturedBody []byte

	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedKey = r.Header.Get("AccessKey")
		capturedContentType = r.Header.Get("Content-Type")
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	err := sc.Upload(context.Background(), "stores/t1/images/file.webp", []byte("image-data"))

	require.NoError(t, err)
	assert.Equal(t, "/test-zone/stores/t1/images/file.webp", capturedPath)
	assert.Equal(t, "test-api-key", capturedKey)
	assert.Equal(t, "application/octet-stream", capturedContentType)
	assert.Equal(t, []byte("image-data"), capturedBody)
}

func TestUpload_Unauthorized(t *testing.T) {
	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	err := sc.Upload(context.Background(), "path", []byte("data"))

	require.Error(t, err)
	se, ok := err.(*StorageError)
	require.True(t, ok)
	assert.Equal(t, 401, se.StatusCode)
	assert.False(t, se.Retryable)
}

func TestUpload_NotFound(t *testing.T) {
	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := sc.Upload(context.Background(), "path", []byte("data"))

	require.Error(t, err)
	se, ok := err.(*StorageError)
	require.True(t, ok)
	assert.Equal(t, 404, se.StatusCode)
	assert.False(t, se.Retryable)
}

func TestUpload_ServerError(t *testing.T) {
	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	err := sc.Upload(context.Background(), "path", []byte("data"))

	require.Error(t, err)
	se, ok := err.(*StorageError)
	require.True(t, ok)
	assert.Equal(t, 500, se.StatusCode)
	assert.True(t, se.Retryable)
}

func TestUpload_ContextTimeout(t *testing.T) {
	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()
	sc.uploadTimeout = 50 * time.Millisecond

	err := sc.Upload(context.Background(), "path", []byte("data"))
	require.Error(t, err)
}

// --- Delete Tests ---

func TestDelete_Success(t *testing.T) {
	var capturedPath, capturedMethod string

	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := sc.Delete(context.Background(), "stores/t1/images/file.webp")

	require.NoError(t, err)
	assert.Equal(t, "/test-zone/stores/t1/images/file.webp", capturedPath)
	assert.Equal(t, http.MethodDelete, capturedMethod)
}

func TestDelete_NotFound(t *testing.T) {
	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	err := sc.Delete(context.Background(), "path")

	require.Error(t, err)
	se, ok := err.(*StorageError)
	require.True(t, ok)
	assert.Equal(t, 404, se.StatusCode)
	assert.False(t, se.Retryable)
}

func TestDelete_ServiceUnavailable(t *testing.T) {
	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := sc.Delete(context.Background(), "path")

	require.Error(t, err)
	se, ok := err.(*StorageError)
	require.True(t, ok)
	assert.Equal(t, 503, se.StatusCode)
	assert.True(t, se.Retryable)
}

// --- Exists Tests ---

func TestExists_Found(t *testing.T) {
	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exists, err := sc.Exists(context.Background(), "stores/t1/images/file.webp")

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestExists_NotFound(t *testing.T) {
	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	exists, err := sc.Exists(context.Background(), "path")

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestExists_ServerError(t *testing.T) {
	sc, server := newTestClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := sc.Exists(context.Background(), "path")

	require.Error(t, err)
	se, ok := err.(*StorageError)
	require.True(t, ok)
	assert.Equal(t, 500, se.StatusCode)
	assert.True(t, se.Retryable)
}

// --- StorageError ---

func TestStorageError_ErrorMessage(t *testing.T) {
	se := &StorageError{Message: "upload: unauthorized", StatusCode: 401, Retryable: false}
	assert.Equal(t, "storage error (HTTP 401): upload: unauthorized", se.Error())
}

// --- IsRetryableStorageError ---

func TestIsRetryableStorageError(t *testing.T) {
	assert.True(t, IsRetryableStorageError(&StorageError{Retryable: true}))
	assert.False(t, IsRetryableStorageError(&StorageError{Retryable: false}))
	assert.False(t, IsRetryableStorageError(assert.AnError))
}
