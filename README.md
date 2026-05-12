# mediaforge

[![Go Reference](https://pkg.go.dev/badge/github.com/Ibtkary/mediaforge.svg)](https://pkg.go.dev/github.com/Ibtkary/mediaforge)
[![Go Report Card](https://goreportcard.com/badge/github.com/Ibtkary/mediaforge)](https://goreportcard.com/report/github.com/Ibtkary/mediaforge)
[![Tests](https://github.com/Ibtkary/mediaforge/actions/workflows/ci.yml/badge.svg)](https://github.com/Ibtkary/mediaforge/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-95%25-brightgreen)](https://github.com/Ibtkary/mediaforge)

**Provider-agnostic image processing and storage for Go.**

mediaforge takes raw images, validates them, generates optimized WebP variants (thumb, medium, large), uploads them to any cloud storage, and serves them via signed CDN URLs.

```go
client, _ := mediaforge.NewClient(storage, signer)
asset, _  := client.Upload(ctx, "store1", imageBytes)      // process + upload
urls, _   := client.GetSignedURL(asset.StoragePath, variants, time.Hour) // signed CDN URLs
```

## Features

- **Provider-agnostic** -- Bunny.net, AWS S3, Cloudflare R2, MinIO, or bring your own
- **Automatic variants** -- thumb (150px), medium (400px), large (800px), original (2000px)
- **WebP encoding** -- 80-90% smaller than JPEG with better quality
- **Signed URLs** -- time-limited CDN links for secure delivery
- **Cost optimization** -- content dedup, smart variant skip, lazy variants
- **Observability** -- structured logging (slog) + hooks for metrics/tracing
- **Minimal deps** -- core has only 3 dependencies (uuid, x/image, testify)
- **214 tests, 95%+ coverage**

## Install

```bash
go get github.com/Ibtkary/mediaforge
```

For AWS S3:

```bash
go get github.com/Ibtkary/mediaforge/providers/s3
```

For Cloudflare R2:

```bash
go get github.com/Ibtkary/mediaforge/providers/r2
```

### Prerequisites

- Go 1.21+
- `cwebp` binary in PATH ([install](https://developers.google.com/speed/webp/download)) -- not needed if using a custom encoder

## Quick Start

### With AWS S3

```go
import (
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/Ibtkary/mediaforge"
    s3provider "github.com/Ibtkary/mediaforge/providers/s3"
)

// Setup
awsCfg, _ := config.LoadDefaultConfig(ctx)
s3Client := s3.NewFromConfig(awsCfg)

storage := s3provider.NewStorage(s3Client, "my-bucket")
signer  := s3provider.NewSigner(s3.NewPresignClient(s3Client), "my-bucket")

client, err := mediaforge.NewClient(storage, signer)

// Upload
asset, err := client.Upload(ctx, "tenant1", imageBytes)
// asset.StoragePath = "stores/tenant1/images/e477ecfe_0cf31a90.webp"
// asset.Variants    = {thumb, medium, large}

// Get signed URLs
urls, err := client.GetSignedURL(asset.StoragePath,
    []string{"thumb", "medium", "large"}, time.Hour)
// urls.Original         = "https://..."
// urls.Variants["thumb"] = "https://..."

// Delete
results, err := client.Delete(ctx, "tenant1", "e477ecfe_0cf31a90.webp",
    []string{"thumb", "medium", "large"})
```

### With Cloudflare R2

R2 uses Cloudflare's S3-compatible API. Keep the bucket private and serve images
through presigned URLs.

```go
import (
    "context"
    "os"
    "time"

    "github.com/Ibtkary/mediaforge"
    r2provider "github.com/Ibtkary/mediaforge/providers/r2"
)

client, err := r2provider.NewClient(context.Background(), r2provider.Config{
    AccountID:       os.Getenv("R2_ACCOUNT_ID"),
    AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
    SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
    Bucket:          os.Getenv("R2_BUCKET"),
    // Endpoint is optional. Leave empty to use:
    // https://<account_id>.r2.cloudflarestorage.com
    Endpoint: os.Getenv("R2_ENDPOINT"),
}, mediaforge.WithVariants(mediaforge.DefaultVariants()))
if err != nil {
    return err
}

asset, err := client.Upload(ctx, "tenant1", imageBytes)
urls, err := client.GetSignedURL(asset.StoragePath,
    []string{"thumb", "medium", "large"}, time.Hour)
```

### With Bunny.net

```go
import "github.com/Ibtkary/mediaforge"

client, err := mediaforge.New(mediaforge.Config{
    StorageZone:    os.Getenv("BUNNY_ZONE"),
    StorageAPIKey:  os.Getenv("BUNNY_API_KEY"),
    CDNBaseURL:     os.Getenv("BUNNY_CDN_URL"),
    CDNSecurityKey: os.Getenv("BUNNY_CDN_KEY"),
})
```

## How It Works

```
Input (JPEG/PNG)                    Output (WebP variants)

  2000x1500                         original  2000x1500  ~45KB
  800KB JPEG    ──▶  mediaforge  ──▶  large      800x600  ~38KB
                    (validate,       medium     400x300  ~13KB
                     resize,         thumb      150x112   ~3KB
                     encode)
                                    Total: ~99KB (88% savings)
```

## Options

```go
client, _ := mediaforge.NewClient(storage, signer,
    // Variants
    mediaforge.WithVariants(mediaforge.DefaultVariants()),

    // Limits
    mediaforge.WithMaxFileSize(5 * 1024 * 1024),  // 5MB
    mediaforge.WithMinDimensions(50, 50),
    mediaforge.WithMaxDimensions(4000, 4000),

    // Retry
    mediaforge.WithRetries(3, time.Second),

    // Cost optimization
    mediaforge.WithDedup(true),            // skip if content hash exists
    mediaforge.WithSmartVariantSkip(true), // skip variants for small images
    mediaforge.WithLazyVariants(true),     // upload original only

    // Observability
    mediaforge.WithLogger(slog.Default()),
    mediaforge.WithHook(&myPrometheusHook{}),

    // Custom encoder (skip cwebp requirement)
    mediaforge.WithEncoder(&myPureGoEncoder{}),
)
```

## Providers

| Provider | Package | Storage | Signed URLs |
|----------|---------|---------|-------------|
| **AWS S3** | `mediaforge/providers/s3` | S3 PutObject | S3 Presigned URLs |
| **Cloudflare R2** | `mediaforge/providers/r2` | S3-compatible PutObject | S3-compatible Presigned URLs |
| **Bunny.net** | built-in | Bunny Storage API | Bunny CDN Tokens |
| **Custom** | implement `ObjectStorage` + `URLSigner` | your choice | your choice |

### Custom Provider

```go
type MyStorage struct{}

func (s *MyStorage) Upload(ctx context.Context, key string, r io.Reader, size int64) error { ... }
func (s *MyStorage) Delete(ctx context.Context, key string) error { ... }
func (s *MyStorage) Exists(ctx context.Context, key string) (bool, error) { ... }

type MySigner struct{}

func (s *MySigner) SignURL(key string, expires time.Time) (string, error) { ... }
```

## Observability

### Logging

```go
mediaforge.WithLogger(slog.Default())
```

Logs at key points: upload start/end, variant processing, dedup hits, errors.

### Hooks

```go
type Hook interface {
    BeforeUpload(ctx context.Context, key string, size int64)
    AfterUpload(ctx context.Context, key string, size int64, dur time.Duration, err error)
    BeforeDelete(ctx context.Context, key string)
    AfterDelete(ctx context.Context, key string, dur time.Duration, err error)
}
```

Embed `mediaforge.NoopHook` and override only what you need:

```go
type MetricsHook struct { mediaforge.NoopHook }

func (h *MetricsHook) AfterUpload(ctx context.Context, key string, size int64, dur time.Duration, err error) {
    uploadDuration.Observe(dur.Seconds())
    if err != nil { uploadErrors.Inc() }
}
```

## Error Handling

```go
asset, err := client.Upload(ctx, "tenant1", data)
if err != nil {
    var mediaErr *mediaforge.Error
    if errors.As(err, &mediaErr) {
        switch mediaErr.Category {
        case mediaforge.CategoryValidation:
            // bad image (wrong format, too large, too small)
        case mediaforge.CategoryStorage:
            // cloud storage error (check mediaforge.IsRetryable(err))
        case mediaforge.CategoryProcessing:
            // encoding failed (cwebp issue?)
        }
    }
}
```

## Documentation

- [Integration Guide](docs/integration-guide.md) -- full flow from upload to display
- [Architecture Decision Record](docs/ADR-001-architecture.md) -- design rationale

## License

MIT
