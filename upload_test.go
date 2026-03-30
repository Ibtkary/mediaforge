package mediaforge

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpload_Success(t *testing.T) {
	skipIfNoCWebP(t)

	var mu sync.Mutex
	var paths []string

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))

	data := createTestJPEG(500, 400)
	result, err := client.Upload(context.Background(), "tenant1", data)

	require.NoError(t, err)
	require.NotNil(t, result)

	// 1 original + 3 variants = 4 PUT calls
	assert.Len(t, paths, 4)

	// PendingAsset fields
	assert.NotEmpty(t, result.StoragePath)
	assert.NotEmpty(t, result.ContentHash)
	assert.Equal(t, "tenant1", result.TenantID)
	assert.Equal(t, "image/jpeg", result.OriginalMIME)
	assert.Equal(t, 500, result.Width)
	assert.Equal(t, 400, result.Height)
	assert.True(t, result.FileSizeBytes > 0)
	assert.NotEmpty(t, result.SessionID)
	assert.False(t, result.UploadedAt.IsZero())
	assert.False(t, result.ExpiresAt.IsZero())
	assert.True(t, result.ExpiresAt.After(result.UploadedAt))
	require.Len(t, result.Variants, 3)
	assert.Contains(t, result.Variants, "thumb")
	assert.Contains(t, result.Variants, "medium")
	assert.Contains(t, result.Variants, "large")
}

func TestUpload_InvalidTenantID(t *testing.T) {
	skipIfNoCWebP(t)

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	_, err := client.Upload(context.Background(), "", createTestJPEG(200, 200))

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryValidation, mErr.Category)
	assert.Equal(t, "Upload", mErr.Op)
}

func TestUpload_TenantIDPathTraversal(t *testing.T) {
	skipIfNoCWebP(t)

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	_, err := client.Upload(context.Background(), "../evil", createTestJPEG(200, 200))

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryValidation, mErr.Category)
}

func TestUpload_InvalidImage(t *testing.T) {
	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	_, err := client.Upload(context.Background(), "tenant1", []byte("not-an-image"))

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryValidation, mErr.Category)
}

func TestUpload_ImageTooLarge(t *testing.T) {
	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	// Override max file size to something tiny
	client.config.MaxFileSize = 10

	_, err := client.Upload(context.Background(), "tenant1", createTestJPEG(200, 200))

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryValidation, mErr.Category)
}

func TestUpload_StorageFailure(t *testing.T) {
	skipIfNoCWebP(t)

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	_, err := client.Upload(context.Background(), "tenant1", createTestJPEG(200, 200))

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryStorage, mErr.Category)
	assert.True(t, mErr.Retryable)
}

func TestUpload_StorageUnauthorized(t *testing.T) {
	skipIfNoCWebP(t)

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	_, err := client.Upload(context.Background(), "tenant1", createTestJPEG(200, 200))

	require.Error(t, err)
	var mErr *Error
	require.True(t, errors.As(err, &mErr))
	assert.Equal(t, CategoryStorage, mErr.Category)
	assert.False(t, mErr.Retryable)
}

func TestUpload_VariantFailure_Cleanup(t *testing.T) {
	skipIfNoCWebP(t)

	var mu sync.Mutex
	callCount := 0
	var deletePaths []string

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.Method == http.MethodDelete {
			deletePaths = append(deletePaths, r.URL.Path)
			w.WriteHeader(http.StatusOK)
			return
		}

		// PUT: first call (original) succeeds, second call (first variant) fails
		callCount++
		if callCount <= 1 {
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))

	_, err := client.Upload(context.Background(), "tenant1", createTestJPEG(200, 200))

	require.Error(t, err)
	// Verify cleanup attempted: at least the original DELETE
	assert.GreaterOrEqual(t, len(deletePaths), 1)
}

func TestUpload_ContextCancellation(t *testing.T) {
	skipIfNoCWebP(t)

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusCreated)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := client.Upload(ctx, "tenant1", createTestJPEG(200, 200))
	require.Error(t, err)
}

func TestUpload_FilenameFormat(t *testing.T) {
	skipIfNoCWebP(t)

	var capturedPath string

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capturedPath == "" {
			capturedPath = r.URL.Path // capture first PUT (original)
		}
		w.WriteHeader(http.StatusCreated)
	}))

	result, err := client.Upload(context.Background(), "tenant1", createTestJPEG(200, 200))

	require.NoError(t, err)

	// Filename format: {16 hex}_{8 hex}.webp
	filenamePattern := regexp.MustCompile(`[a-f0-9]{16}_[a-f0-9]{8}\.webp$`)
	assert.Regexp(t, filenamePattern, result.StoragePath)
}

func TestUpload_StoragePaths(t *testing.T) {
	skipIfNoCWebP(t)

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	result, err := client.Upload(context.Background(), "my-store", createTestJPEG(200, 200))

	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(result.StoragePath, "stores/my-store/images/"))
	for _, vi := range result.Variants {
		assert.True(t, strings.HasPrefix(vi.Path, "stores/my-store/images/"))
	}
}

func TestUpload_ContentHashDeterministic(t *testing.T) {
	skipIfNoCWebP(t)

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	data := createTestJPEG(200, 200)
	r1, err1 := client.Upload(context.Background(), "t1", data)
	r2, err2 := client.Upload(context.Background(), "t1", data)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, r1.ContentHash, r2.ContentHash)
}

func TestUpload_PNGInput(t *testing.T) {
	skipIfNoCWebP(t)

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	result, err := client.Upload(context.Background(), "tenant1", createTestPNG(300, 200))

	require.NoError(t, err)
	assert.Equal(t, "image/png", result.OriginalMIME)
}

func TestUpload_PendingAssetMetadata(t *testing.T) {
	skipIfNoCWebP(t)

	client, _ := testClientWithServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	before := time.Now()
	result, err := client.Upload(context.Background(), "tenant1", createTestJPEG(500, 400))
	after := time.Now()

	require.NoError(t, err)

	// SessionID is a UUID
	assert.Len(t, result.SessionID, 36) // UUID with hyphens
	assert.Contains(t, result.SessionID, "-")

	// Timestamps
	assert.True(t, result.UploadedAt.After(before) || result.UploadedAt.Equal(before))
	assert.True(t, result.UploadedAt.Before(after) || result.UploadedAt.Equal(after))
	assert.True(t, result.ExpiresAt.After(result.UploadedAt))

	// Variant info
	for name, vi := range result.Variants {
		assert.NotEmpty(t, vi.Path, "variant %s path empty", name)
		assert.True(t, vi.Width > 0, "variant %s width zero", name)
		assert.True(t, vi.Height > 0, "variant %s height zero", name)
		assert.True(t, vi.SizeBytes > 0, "variant %s size zero", name)
	}
}
