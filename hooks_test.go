package mediaforge

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingHook records all hook calls for testing.
type recordingHook struct {
	mu      sync.Mutex
	uploads []hookCall
	deletes []hookCall
}

type hookCall struct {
	key  string
	size int64
	dur  time.Duration
	err  error
}

func (h *recordingHook) BeforeUpload(_ context.Context, key string, size int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.uploads = append(h.uploads, hookCall{key: key, size: size})
}

func (h *recordingHook) AfterUpload(_ context.Context, key string, size int64, dur time.Duration, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Find the last Before call and update it
	for i := len(h.uploads) - 1; i >= 0; i-- {
		if h.uploads[i].key == key && h.uploads[i].dur == 0 {
			h.uploads[i].dur = dur
			h.uploads[i].err = err
			break
		}
	}
}

func (h *recordingHook) BeforeDelete(_ context.Context, key string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.deletes = append(h.deletes, hookCall{key: key})
}

func (h *recordingHook) AfterDelete(_ context.Context, key string, dur time.Duration, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := len(h.deletes) - 1; i >= 0; i-- {
		if h.deletes[i].key == key && h.deletes[i].dur == 0 {
			h.deletes[i].dur = dur
			h.deletes[i].err = err
			break
		}
	}
}

func TestHook_UploadCalls(t *testing.T) {
	hook := &recordingHook{}
	ms := &mockStorage{}

	client, err := NewClient(ms, &mockSigner{},
		WithHook(hook),
		WithEncoder(&mockEncoder{}),
		WithClock(fixedClock{t: time.Now()}),
	)
	require.NoError(t, err)

	data := createTestJPEG(200, 200)
	_, err = client.Upload(context.Background(), "tenant1", data)
	require.NoError(t, err)

	hook.mu.Lock()
	defer hook.mu.Unlock()

	// 1 original + 3 variants = 4 upload hook calls
	assert.Equal(t, 4, len(hook.uploads), "expected 4 upload hook calls (original + 3 variants)")
}

func TestHook_DeleteCalls(t *testing.T) {
	hook := &recordingHook{}
	ms := &mockStorage{}

	client, err := NewClient(ms, &mockSigner{},
		WithHook(hook),
	)
	require.NoError(t, err)

	results, err := client.Delete(context.Background(), "tenant1", "abc_def.webp", []string{"thumb", "medium"})
	require.NoError(t, err)
	assert.Len(t, results, 3) // original + 2 variants

	hook.mu.Lock()
	defer hook.mu.Unlock()
	assert.Equal(t, 3, len(hook.deletes), "expected 3 delete hook calls")
}

func TestNoopHook_NoPanic(t *testing.T) {
	h := NoopHook{}
	ctx := context.Background()
	h.BeforeUpload(ctx, "key", 100)
	h.AfterUpload(ctx, "key", 100, time.Second, nil)
	h.BeforeDelete(ctx, "key")
	h.AfterDelete(ctx, "key", time.Second, nil)
}
