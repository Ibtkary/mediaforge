package mediaforge

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelete_Success(t *testing.T) {
	var mu sync.Mutex
	var deletedPaths []string

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		deletedPaths = append(deletedPaths, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))

	results, err := client.Delete(context.Background(), "tenant1", "abc123_d4e5f6g7.webp", []string{"thumb", "medium", "large"})

	require.NoError(t, err)
	require.Len(t, results, 4) // 1 original + 3 variants
	for _, r := range results {
		assert.True(t, r.Success)
		assert.NoError(t, r.Err)
	}
	assert.Len(t, deletedPaths, 4)
}

func TestDelete_InvalidTenantID(t *testing.T) {
	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	results, err := client.Delete(context.Background(), "", "file.webp", nil)

	require.Error(t, err)
	assert.Nil(t, results)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryValidation, mErr.Category)
	assert.Equal(t, "Delete", mErr.Op)
}

func TestDelete_PartialFailure(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		c := callCount
		mu.Unlock()

		// First DELETE succeeds, second fails
		if c == 1 {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	results, err := client.Delete(context.Background(), "tenant1", "file.webp", []string{"thumb"})

	require.NoError(t, err) // method error is nil
	require.Len(t, results, 2)

	assert.True(t, results[0].Success)
	assert.NoError(t, results[0].Err)

	assert.False(t, results[1].Success)
	assert.Error(t, results[1].Err)
}

func TestDelete_AllFail(t *testing.T) {
	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	results, err := client.Delete(context.Background(), "tenant1", "file.webp", []string{"thumb", "medium"})

	require.NoError(t, err) // method error nil — per-result errors
	require.Len(t, results, 3)
	for _, r := range results {
		assert.False(t, r.Success)
		assert.Error(t, r.Err)
	}
}

func TestDelete_NoVariants(t *testing.T) {
	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	results, err := client.Delete(context.Background(), "tenant1", "file.webp", nil)

	require.NoError(t, err)
	require.Len(t, results, 1) // just the original
	assert.True(t, results[0].Success)
}

func TestDelete_CorrectPaths(t *testing.T) {
	var mu sync.Mutex
	var paths []string

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))

	_, err := client.Delete(context.Background(), "store1", "abc_def.webp", []string{"thumb", "large"})

	require.NoError(t, err)
	require.Len(t, paths, 3)
	// Paths include the storage zone prefix from httptest
	assert.Contains(t, paths[0], "stores/store1/images/abc_def.webp")
	assert.Contains(t, paths[1], "stores/store1/images/abc_def_thumb.webp")
	assert.Contains(t, paths[2], "stores/store1/images/abc_def_large.webp")
}

func TestDelete_RetrySuccess(t *testing.T) {
	var mu sync.Mutex
	callCount := 0

	cfg := testConfig()
	cfg.MaxRetries = intPtr(2)
	client, err := New(cfg)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		c := callCount
		mu.Unlock()

		// First call 500, second call 200
		if c == 1 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(server.Close)
	client.setStorageBaseURL(server.URL)

	results, err := client.Delete(context.Background(), "t1", "file.webp", nil)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Success)
}

func TestDelete_ContextCancellation(t *testing.T) {
	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	results, err := client.Delete(ctx, "tenant1", "file.webp", []string{"thumb"})

	require.NoError(t, err) // method error is nil
	// At least some results should have errors from cancelled context
	hasError := false
	for _, r := range results {
		if r.Err != nil {
			hasError = true
		}
	}
	assert.True(t, hasError)
}
