package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const bunnyBaseURL = "https://storage.bunnycdn.com"

// StorageError represents an error from the Bunny Storage API.
type StorageError struct {
	Message    string
	StatusCode int
	Retryable  bool
}

func (e *StorageError) Error() string {
	return fmt.Sprintf("storage error (HTTP %d): %s", e.StatusCode, e.Message)
}

// IsRetryable returns whether the error is retryable.
func (e *StorageError) IsRetryable() bool {
	return e.Retryable
}

// StorageClient communicates with the Bunny.net Storage API.
type StorageClient struct {
	httpClient    *http.Client
	zone          string
	apiKey        string
	baseURL       string
	uploadTimeout time.Duration
	deleteTimeout time.Duration
}

// NewStorageClient creates a new StorageClient.
func NewStorageClient(zone, apiKey string, uploadTimeout, deleteTimeout time.Duration) *StorageClient {
	return &StorageClient{
		httpClient:    &http.Client{},
		zone:          zone,
		apiKey:        apiKey,
		baseURL:       bunnyBaseURL,
		uploadTimeout: uploadTimeout,
		deleteTimeout: deleteTimeout,
	}
}

// SetBaseURL overrides the base URL (used for testing with httptest).
func (sc *StorageClient) SetBaseURL(url string) {
	sc.baseURL = url
}

// Upload puts a file to Bunny Storage at the given path.
func (sc *StorageClient) Upload(ctx context.Context, path string, data []byte) error {
	ctx, cancel := context.WithTimeout(ctx, sc.uploadTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/%s/%s", sc.baseURL, sc.zone, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}

	req.Header.Set("AccessKey", sc.apiKey)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusCreated {
		return mapStorageError("upload", resp.StatusCode)
	}
	return nil
}

// Delete removes a file from Bunny Storage.
func (sc *StorageClient) Delete(ctx context.Context, path string) error {
	ctx, cancel := context.WithTimeout(ctx, sc.deleteTimeout)
	defer cancel()

	url := fmt.Sprintf("%s/%s/%s", sc.baseURL, sc.zone, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("creating delete request: %w", err)
	}

	req.Header.Set("AccessKey", sc.apiKey)

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return mapStorageError("delete", resp.StatusCode)
	}
	return nil
}

// Exists checks if a file exists in Bunny Storage.
func (sc *StorageClient) Exists(ctx context.Context, path string) (bool, error) {
	url := fmt.Sprintf("%s/%s/%s", sc.baseURL, sc.zone, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, fmt.Errorf("creating exists request: %w", err)
	}

	req.Header.Set("AccessKey", sc.apiKey)

	resp, err := sc.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("exists request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, mapStorageError("exists", resp.StatusCode)
	}
}

func mapStorageError(op string, statusCode int) *StorageError {
	retryable := statusCode >= 500
	var msg string
	switch statusCode {
	case http.StatusUnauthorized:
		msg = "unauthorized"
	case http.StatusBadRequest:
		msg = "bad request"
	case http.StatusNotFound:
		msg = "not found"
	default:
		if retryable {
			msg = "server error"
		} else {
			msg = fmt.Sprintf("unexpected status %d", statusCode)
		}
	}
	return &StorageError{
		Message:    fmt.Sprintf("%s: %s", op, msg),
		StatusCode: statusCode,
		Retryable:  retryable,
	}
}

// IsRetryableStorageError checks if an error is a retryable StorageError.
func IsRetryableStorageError(err error) bool {
	se, ok := err.(*StorageError)
	if !ok {
		return false
	}
	return se.IsRetryable()
}
