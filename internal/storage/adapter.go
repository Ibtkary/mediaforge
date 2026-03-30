package storage

import (
	"context"
	"io"
)

// BunnyAdapter wraps the existing StorageClient to satisfy the ObjectStorage interface.
// It bridges the io.Reader-based interface to the []byte-based StorageClient.
type BunnyAdapter struct {
	client *StorageClient
}

// NewBunnyAdapter creates a BunnyAdapter wrapping the given StorageClient.
func NewBunnyAdapter(client *StorageClient) *BunnyAdapter {
	return &BunnyAdapter{client: client}
}

// Upload reads all data from r and delegates to StorageClient.Upload.
func (a *BunnyAdapter) Upload(ctx context.Context, key string, r io.Reader, size int64) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return &StorageError{
			Message:    "reading upload data: " + err.Error(),
			StatusCode: 0,
			Retryable:  false,
		}
	}
	return a.client.Upload(ctx, key, data)
}

// Delete delegates to StorageClient.Delete.
func (a *BunnyAdapter) Delete(ctx context.Context, key string) error {
	return a.client.Delete(ctx, key)
}

// Exists delegates to StorageClient.Exists.
func (a *BunnyAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.client.Exists(ctx, key)
}
