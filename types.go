package mediaforge

import "time"

// UploadSession represents an active upload session.
type UploadSession struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// VariantInfo describes a generated image variant.
type VariantInfo struct {
	Path      string `json:"path"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	SizeBytes int64  `json:"size_bytes"`
}

// PendingAsset represents an uploaded asset that has not yet been committed.
type PendingAsset struct {
	StoragePath   string                `json:"storage_path"`
	ContentHash   string                `json:"content_hash"`
	FileSizeBytes int64                 `json:"file_size_bytes"`
	OriginalMIME  string                `json:"original_mime"`
	Width         int                   `json:"width"`
	Height        int                   `json:"height"`
	Variants      map[string]VariantInfo `json:"variants"`
	SessionID     string                `json:"session_id"`
	TenantID      string                `json:"tenant_id"`
	UploadedAt    time.Time             `json:"uploaded_at"`
	ExpiresAt     time.Time             `json:"expires_at"`
}

// Asset represents a committed, active asset.
type Asset struct {
	PendingAsset
	OwnerType   string    `json:"owner_type"`
	OwnerID     string    `json:"owner_id"`
	State       string    `json:"state"`
	CommittedAt time.Time `json:"committed_at"`
}

// SignedURLSet contains signed URLs for original and all variants.
type SignedURLSet struct {
	Original string            `json:"original"`
	Variants map[string]string `json:"variants"`
}

// DeleteResult represents the outcome of deleting a single path.
type DeleteResult struct {
	Path    string `json:"path"`
	Success bool   `json:"success"`
	Err     error  `json:"-"`
}

// AssetPaths holds the storage path for an asset and all its variants.
type AssetPaths struct {
	StoragePath  string   `json:"storage_path"`
	VariantPaths []string `json:"variant_paths"`
}
