# Mediastore — دليل الاستخدام والتكامل

> دليل شامل لتركيب واستخدام مكتبة mediaforge في أي مشروع Go.

---

## جدول المحتويات

- [الصورة الكبيرة](#الصورة-الكبيرة)
- [الرحلة الكاملة لصورة](#الرحلة-الكاملة-لصورة)
- [التركيب](#التركيب)
- [الإعداد السريع](#الإعداد-السريع)
- [الاستخدام خطوة بخطوة](#الاستخدام-خطوة-بخطوة)
- [تكامل مع Backend حقيقي](#تكامل-مع-backend-حقيقي)
- [مرجع الـ API](#مرجع-الـ-api)

---

## الصورة الكبيرة

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   📱 App     │     │   🖥️ API     │     │  📦 Media-   │     │   ☁️ Cloud   │
│   (Flutter)  │────▶│   (Go)       │────▶│    store     │────▶│   Storage    │
│              │     │              │     │   Library    │     │  (S3/Bunny)  │
└──────────────┘     └──────────────┘     └──────────────┘     └──────────────┘
                                                │
                                    ┌───────────┼───────────┐
                                    │           │           │
                                 Validate    Process     Upload
                                 (MIME,      (resize,    (original
                                  size,       WebP        + thumb
                                  dims)       encode)     + medium
                                                          + large)
```

**ببساطة:** المكتبة بتاخد صورة → تتحقق منها → تحولها لـ WebP → تعمل نسخ بأحجام مختلفة → ترفعها على الـ cloud → ترجعلك URLs موقعة لعرضها.

---

## الرحلة الكاملة لصورة

من لحظة ما المستخدم يختار صورة لحد ما تظهر في التطبيق:

```
                         رحلة الصورة الكاملة
═══════════════════════════════════════════════════════════════════

  📱 المستخدم يختار صورة
       │
       ▼
  ┌─────────────────────────────────────────────────────────────┐
  │  1. الرفع (Upload)                                         │
  │                                                             │
  │  📱 App ──HTTP POST──▶ 🖥️ API ──Upload()──▶ 📦 Mediastore │
  │                                                             │
  │  الصورة تمر بـ 5 مراحل داخل المكتبة:                       │
  │                                                             │
  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
  │  │ Validate │─▶│  Hash    │─▶│ Decode   │─▶│ Resize   │   │
  │  │          │  │ SHA-256  │  │ JPEG/PNG │  │ 4 sizes  │   │
  │  │ MIME ✓   │  │          │  │ ──▶      │  │          │   │
  │  │ Size ✓   │  │ بصمة    │  │ image.   │  │ original │   │
  │  │ Dims ✓   │  │ فريدة   │  │ Image    │  │ large    │   │
  │  └──────────┘  └──────────┘  └──────────┘  │ medium   │   │
  │                                             │ thumb    │   │
  │                                             └────┬─────┘   │
  │                                                  │         │
  │                                             ┌────▼─────┐   │
  │                                             │ Encode   │   │
  │                                             │ ──▶ WebP │   │
  │                                             │ (أصغر   │   │
  │                                             │  حجماً)  │   │
  │                                             └────┬─────┘   │
  │                                                  │         │
  │  ☁️ ◀── Upload 4 files ◀─────────────────────────┘         │
  │                                                             │
  │  النتيجة: PendingAsset                                      │
  │  ├── StoragePath: "stores/store1/images/a1b2c3..._d4e5.webp"│
  │  ├── ContentHash: "e3b0c44298fc1c14..."                     │
  │  ├── Variants: {thumb, medium, large}                       │
  │  └── SessionID: "uuid-..."                                  │
  └─────────────────────────────────────────────────────────────┘
       │
       ▼
  ┌─────────────────────────────────────────────────────────────┐
  │  2. الحفظ في قاعدة البيانات                                │
  │                                                             │
  │  🖥️ API بياخد الـ PendingAsset ويحفظ بياناته في DB:        │
  │                                                             │
  │  INSERT INTO product_images (                               │
  │      product_id, storage_path, content_hash,                │
  │      width, height, mime, variants                          │
  │  ) VALUES (...)                                             │
  │                                                             │
  │  ملاحظة: الـ DB بيحفظ الـ path بس، مش الصورة نفسها.        │
  └─────────────────────────────────────────────────────────────┘
       │
       ▼
  ┌─────────────────────────────────────────────────────────────┐
  │  3. العرض (Display)                                        │
  │                                                             │
  │  📱 App محتاج يعرض الصورة:                                 │
  │                                                             │
  │  📱 ──"هات صور المنتج"──▶ 🖥️ API                           │
  │                               │                             │
  │                     GetSignedURL(path, ["thumb"], 1h)       │
  │                               │                             │
  │                               ▼                             │
  │                     📦 Mediastore يولد URLs موقعة:           │
  │                     ┌──────────────────────────────────┐    │
  │                     │ SignedURLSet {                    │    │
  │                     │   Original: "https://cdn.../     │    │
  │                     │     abc.webp?token=x&expires=t"  │    │
  │                     │   Variants: {                    │    │
  │                     │     "thumb": "https://cdn.../    │    │
  │                     │       abc_thumb.webp?token=..."  │    │
  │                     │   }                              │    │
  │                     │ }                                │    │
  │                     └──────────────────────────────────┘    │
  │                               │                             │
  │  📱 ◀──── JSON response ◀─────┘                             │
  │                                                             │
  │  📱 App يعرض الصورة مباشرة من CDN:                          │
  │  Image.network(signedUrl)                                   │
  └─────────────────────────────────────────────────────────────┘
       │
       ▼
  ┌─────────────────────────────────────────────────────────────┐
  │  4. الحذف (Delete) — لو المستخدم حذف المنتج                │
  │                                                             │
  │  📱 ──"احذف المنتج"──▶ 🖥️ API                               │
  │                            │                                │
  │              Delete(ctx, "store1", "abc.webp",              │
  │                     ["thumb", "medium", "large"])            │
  │                            │                                │
  │              يحذف 4 ملفات: original + 3 variants            │
  │                            │                                │
  │              ▼                                              │
  │  ☁️ Storage: 4 files deleted                                │
  │  🗄️ DB: DELETE FROM product_images WHERE ...                │
  └─────────────────────────────────────────────────────────────┘
```

---

## التركيب

### مع Bunny.net

```bash
go get github.com/Ibtkary/mediaforge
```

### مع AWS S3

```bash
go get github.com/Ibtkary/mediaforge
go get github.com/Ibtkary/mediaforge/providers/s3
```

### مع Cloudflare R2

```bash
go get github.com/Ibtkary/mediaforge
go get github.com/Ibtkary/mediaforge/providers/r2
```

### المتطلبات

| المتطلب | مطلوب؟ | ملاحظة |
|---------|--------|--------|
| Go 1.25+ | نعم | |
| `cwebp` binary | نعم* | `brew install webp` أو `apt install webp`. غير مطلوب لو استخدمت encoder مخصص |

---

## الإعداد السريع

### طريقة 1: مع Bunny.net (أبسط طريقة)

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/Ibtkary/mediaforge"
)

func main() {
    client, err := mediaforge.New(mediaforge.Config{
        StorageZone:    os.Getenv("BUNNY_ZONE"),
        StorageAPIKey:  os.Getenv("BUNNY_API_KEY"),
        CDNBaseURL:     os.Getenv("BUNNY_CDN_URL"),
        CDNSecurityKey: os.Getenv("BUNNY_CDN_KEY"),
    })
    if err != nil {
        log.Fatal(err)
    }

    // --- رفع صورة ---
    imageData, _ := os.ReadFile("product.jpg")
    asset, err := client.Upload(context.Background(), "store1", imageData)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("uploaded: %s (%d variants)", asset.StoragePath, len(asset.Variants))

    // --- عرض صورة ---
    urls, err := client.GetSignedURL(asset.StoragePath, []string{"thumb", "medium", "large"}, 0)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("original: %s", urls.Original)
    log.Printf("thumb:    %s", urls.Variants["thumb"])
}
```

### طريقة 2: مع AWS S3 (provider-agnostic)

```go
package main

import (
    "context"
    "log"
    "os"

    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/Ibtkary/mediaforge"
    s3provider "github.com/Ibtkary/mediaforge/providers/s3"
)

func main() {
    // 1. أنشئ S3 client
    awsCfg, _ := config.LoadDefaultConfig(context.Background())
    s3Client := s3.NewFromConfig(awsCfg)
    bucket := os.Getenv("S3_BUCKET")

    // 2. أنشئ الـ storage و signer
    storage := s3provider.NewStorage(s3Client, bucket)
    signer := s3provider.NewSigner(s3.NewPresignClient(s3Client), bucket)

    // 3. أنشئ mediaforge client
    client, err := mediaforge.NewClient(storage, signer,
        mediaforge.WithMaxFileSize(5 * 1024 * 1024), // 5MB max
        mediaforge.WithVariants(mediaforge.DefaultVariants()),
    )
    if err != nil {
        log.Fatal(err)
    }

    // 4. استخدمه بنفس الطريقة!
    imageData, _ := os.ReadFile("product.jpg")
    asset, err := client.Upload(context.Background(), "store1", imageData)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("uploaded to S3: %s", asset.StoragePath)
}
```

### طريقة 3: مع Cloudflare R2 (private bucket + presigned URLs)

> R2 متوافق مع S3 API. الـ bucket يفضل private، والمكتبة تولد presigned URLs مؤقتة للعرض.

```go
package main

import (
    "context"
    "log"
    "os"
    "time"

    "github.com/Ibtkary/mediaforge"
    r2provider "github.com/Ibtkary/mediaforge/providers/r2"
)

func main() {
    client, err := r2provider.NewClient(context.Background(), r2provider.Config{
        AccountID:       os.Getenv("R2_ACCOUNT_ID"),
        AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
        SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
        Bucket:          os.Getenv("R2_BUCKET"),
        // اختياري. اتركه فارغاً لاستخدام:
        // https://<account_id>.r2.cloudflarestorage.com
        Endpoint: os.Getenv("R2_ENDPOINT"),
    },
        mediaforge.WithMaxFileSize(10 * 1024 * 1024),
        mediaforge.WithVariants(mediaforge.DefaultVariants()),
        mediaforge.WithDedup(true),
        mediaforge.WithSmartVariantSkip(true),
    )
    if err != nil {
        log.Fatal(err)
    }

    imageData, _ := os.ReadFile("product.jpg")
    asset, err := client.Upload(context.Background(), "store1", imageData)
    if err != nil {
        log.Fatal(err)
    }

    urls, err := client.GetSignedURL(asset.StoragePath, []string{"thumb", "medium", "large"}, time.Hour)
    if err != nil {
        log.Fatal(err)
    }
    log.Printf("uploaded to R2: %s", asset.StoragePath)
    log.Printf("signed original: %s", urls.Original)
}
```

---

## الاستخدام خطوة بخطوة

### الخطوة 1: الرفع (Upload)

```go
// من []byte
asset, err := client.Upload(ctx, "store1", imageBytes)

// من io.Reader (مثلاً: multipart form)
asset, err := client.UploadReader(ctx, "store1", file)
```

**النتيجة: `PendingAsset`**

```
PendingAsset {
    StoragePath:   "stores/store1/images/e477ecfe_0cf31a90.webp"
    ContentHash:   "e477ecfedf5a8ec6a1b2c3d4e5f67890..."
    FileSizeBytes: 45_230
    OriginalMIME:  "image/jpeg"      ← الصيغة الأصلية
    Width:         800                ← بعد الـ resize
    Height:        600
    Variants: {
        "thumb":  {Path: "...thumb.webp",  W:150, H:112, Size:3_200}
        "medium": {Path: "...medium.webp", W:400, H:300, Size:12_800}
        "large":  {Path: "...large.webp",  W:800, H:600, Size:38_400}
    }
    SessionID:     "a1b2c3d4-..."
    TenantID:      "store1"
    UploadedAt:    2026-03-27T10:00:00Z
    ExpiresAt:     2026-03-27T12:00:00Z  ← ساعتين (PendingTTL)
}
```

**إيه اللي حصل جوه المكتبة:**

```
product.jpg (800KB, 2000x1500, JPEG)
    │
    ├── Validate ✓  (MIME=image/jpeg, size < 10MB, dims 100-8000)
    ├── Hash        (SHA-256 → "e477ecfe...")
    ├── Decode      (JPEG → image.Image)
    │
    ├── Original    (resize 2000x1500 → 2000x1500, encode WebP) → 45KB
    ├── Thumb       (resize → 150x112, encode WebP)              → 3KB
    ├── Medium      (resize → 400x300, encode WebP)              → 13KB
    └── Large       (resize → 800x600, encode WebP)              → 38KB
                                                          Total: ~99KB
                                              (vs 800KB JPEG = 88% savings!)
```

### الخطوة 2: احفظ في الـ Database

```go
// المكتبة مش بتحفظ في DB — أنت اللي بتعمل ده
// احفظ الـ PendingAsset في جدول product_images مثلاً:

db.Exec(`
    INSERT INTO product_images
        (product_id, storage_path, content_hash, width, height, original_mime, variants_json)
    VALUES ($1, $2, $3, $4, $5, $6, $7)`,
    productID,
    asset.StoragePath,
    asset.ContentHash,
    asset.Width,
    asset.Height,
    asset.OriginalMIME,
    variantsJSON, // json.Marshal(asset.Variants)
)
```

### الخطوة 3: العرض (Signed URLs)

```go
// لما الـ App يطلب صور المنتج:
urls, err := client.GetSignedURL(
    storagePath,                         // من الـ DB
    []string{"thumb", "medium", "large"},
    1 * time.Hour,                       // صلاحية الرابط
)

// ابعت الـ URLs في الـ API response
json.NewEncoder(w).Encode(map[string]any{
    "original": urls.Original,
    "thumb":    urls.Variants["thumb"],
    "medium":   urls.Variants["medium"],
    "large":    urls.Variants["large"],
})
```

**الـ Flutter app يعرض الصورة:**

```dart
// في product list → استخدم thumb (أسرع تحميل)
Image.network(imageUrls['thumb'])

// في product detail → استخدم medium
Image.network(imageUrls['medium'])

// في full screen view → استخدم original
Image.network(imageUrls['original'])
```

### الخطوة 4: الحذف

```go
results, err := client.Delete(ctx, "store1", "e477ecfe_0cf31a90.webp",
    []string{"thumb", "medium", "large"},
)
// results: [{Path: "...webp", Success: true}, ...]

// واحذف من الـ DB كمان
db.Exec("DELETE FROM product_images WHERE storage_path = $1", storagePath)
```

---

## تكامل مع Backend حقيقي

### هيكل الملفات المقترح

```
internal/
├── features/
│   └── product/
│       ├── handler/
│       │   └── image_handler.go    ← HTTP handlers
│       ├── service/
│       │   └── image_service.go    ← Business logic + mediaforge
│       └── repository/
│           └── image_repo.go       ← DB operations
├── infrastructure/
│   └── media/
│       └── client.go               ← Mediastore client setup
```

### infrastructure/media/client.go

```go
package media

import (
    "os"
    "github.com/Ibtkary/mediaforge"
)

func NewClient() (*mediaforge.Client, error) {
    return mediaforge.New(mediaforge.Config{
        StorageZone:    os.Getenv("BUNNY_ZONE"),
        StorageAPIKey:  os.Getenv("BUNNY_API_KEY"),
        CDNBaseURL:     os.Getenv("BUNNY_CDN_URL"),
        CDNSecurityKey: os.Getenv("BUNNY_CDN_KEY"),
    })
}
```

### service/image_service.go

```go
package service

type ImageService struct {
    media *mediaforge.Client
    repo  ImageRepository
}

func (s *ImageService) UploadProductImage(ctx context.Context, productID, tenantID string, data []byte) (*ProductImage, error) {
    // 1. رفع الصورة
    asset, err := s.media.Upload(ctx, tenantID, data)
    if err != nil {
        return nil, fmt.Errorf("upload failed: %w", err)
    }

    // 2. حفظ في DB
    img := &ProductImage{
        ProductID:   productID,
        StoragePath: asset.StoragePath,
        ContentHash: asset.ContentHash,
        Width:       asset.Width,
        Height:      asset.Height,
        Variants:    asset.Variants,
    }
    if err := s.repo.Create(ctx, img); err != nil {
        return nil, fmt.Errorf("save failed: %w", err)
    }

    return img, nil
}

func (s *ImageService) GetProductImageURLs(ctx context.Context, storagePath string) (*mediaforge.SignedURLSet, error) {
    return s.media.GetSignedURL(storagePath, []string{"thumb", "medium", "large"}, 1*time.Hour)
}

func (s *ImageService) DeleteProductImage(ctx context.Context, tenantID, storagePath string) error {
    // 1. استخرج اسم الملف من الـ path
    filename := filepath.Base(storagePath)

    // 2. احذف من Storage
    results, err := s.media.Delete(ctx, tenantID, filename, []string{"thumb", "medium", "large"})
    if err != nil {
        return err
    }

    // 3. تحقق من النتائج
    for _, r := range results {
        if !r.Success {
            log.Printf("warning: failed to delete %s: %v", r.Path, r.Err)
        }
    }

    // 4. احذف من DB
    return s.repo.DeleteByPath(ctx, storagePath)
}
```

### handler/image_handler.go

```go
func (h *ImageHandler) Upload(w http.ResponseWriter, r *http.Request) {
    // 1. اقرأ الصورة من الـ request
    file, _, err := r.FormFile("image")
    if err != nil {
        http.Error(w, "missing image", http.StatusBadRequest)
        return
    }
    defer file.Close()

    data, err := io.ReadAll(file)
    if err != nil {
        http.Error(w, "read error", http.StatusBadRequest)
        return
    }

    // 2. ارفع عبر الـ service
    img, err := h.service.UploadProductImage(r.Context(), productID, tenantID, data)
    if err != nil {
        // المكتبة بترجع errors واضحة:
        var mediaErr *mediaforge.Error
        if errors.As(err, &mediaErr) {
            switch mediaErr.Category {
            case mediaforge.CategoryValidation:
                http.Error(w, mediaErr.Msg, http.StatusBadRequest)     // صورة مش صالحة
            case mediaforge.CategoryStorage:
                http.Error(w, "upload failed", http.StatusBadGateway)  // مشكلة في الـ cloud
            default:
                http.Error(w, "internal error", http.StatusInternalServerError)
            }
            return
        }
        http.Error(w, "internal error", http.StatusInternalServerError)
        return
    }

    // 3. رد بالنتيجة
    json.NewEncoder(w).Encode(img)
}
```

---

## الـ Variants — إيه ومتى تستخدم كل واحد؟

```
┌─────────────────────────────────────────────────────────────┐
│                        Original (2000px)                    │
│                                                             │
│  📐 2000 × 1500px  |  📁 ~45KB WebP  |  🎯 Full screen   │
│                                                             │
│  متى: عرض الصورة بالحجم الكامل (zoom, download)            │
├─────────────────────────────────────────────────────────────┤
│                        Large (800px)                        │
│                                                             │
│  📐 800 × 600px   |  📁 ~38KB WebP  |  🎯 Product detail  │
│                                                             │
│  متى: صفحة تفاصيل المنتج                                   │
├─────────────────────────────────────────────────────────────┤
│                        Medium (400px)                       │
│                                                             │
│  📐 400 × 300px   |  📁 ~13KB WebP  |  🎯 Product card    │
│                                                             │
│  متى: كروت المنتجات في الـ grid                             │
├─────────────────────────────────────────────────────────────┤
│                        Thumb (150px)                        │
│                                                             │
│  📐 150 × 112px   |  📁 ~3KB WebP   |  🎯 Thumbnail       │
│                                                             │
│  متى: القوائم، نتائج البحث، السلة                           │
└─────────────────────────────────────────────────────────────┘

  💡 الحجم مهم!
  صورة JPEG أصلية: 800KB
  بعد mediaforge:  99KB total (4 variants)
  التوفير: 88% ✨
```

---

## الخيارات المتقدمة

### توفير التكاليف

```go
client, _ := mediaforge.NewClient(storage, signer,
    // لو المستخدم رفع نفس الصورة مرتين → مش هيرفعها تاني
    mediaforge.WithDedup(true),

    // صور صغيرة (100x100) مش محتاجة thumb/medium/large
    mediaforge.WithSmartVariantSkip(true),

    // ارفع الأصلي بس، ولّد الـ variants لاحقاً
    mediaforge.WithLazyVariants(true),
)
```

### المراقبة (Observability)

```go
client, _ := mediaforge.NewClient(storage, signer,
    // Structured logging
    mediaforge.WithLogger(slog.Default()),

    // Metrics/Tracing hooks
    mediaforge.WithHook(&MyPrometheusHook{}),
)
```

### تخصيص الـ Variants

```go
client, _ := mediaforge.NewClient(storage, signer,
    mediaforge.WithVariants([]mediaforge.VariantConfig{
        {Name: "icon",   MaxWidth: 64,   MaxHeight: 64,   Quality: 75},
        {Name: "card",   MaxWidth: 300,  MaxHeight: 300,  Quality: 85},
        {Name: "hero",   MaxWidth: 1200, MaxHeight: 1200, Quality: 92},
    }),
)
```

---

## مرجع الـ API

### Constructors

| Constructor | الاستخدام | يحتاج cwebp؟ |
|-------------|----------|--------------|
| `New(Config)` | Bunny.net فقط (legacy) | نعم |
| `NewClient(storage, signer, opts...)` | أي provider | لو مفيش custom encoder |
| `NewBunnyClient(zone, key, cdn, cdnKey, opts...)` | Bunny + options | لو مفيش custom encoder |
| `r2.NewClient(ctx, Config, opts...)` | Cloudflare R2 + options | لو مفيش custom encoder |

### Methods

| Method | المدخلات | المخرجات | الوظيفة |
|--------|---------|---------|---------|
| `Upload(ctx, tenantID, []byte)` | Tenant ID + بيانات الصورة | `*PendingAsset` | رفع صورة |
| `UploadReader(ctx, tenantID, io.Reader)` | Tenant ID + reader | `*PendingAsset` | رفع من stream |
| `Delete(ctx, tenantID, filename, variants)` | Tenant + اسم الملف + أسماء الـ variants | `[]DeleteResult` | حذف صورة + variants |
| `GetSignedURL(path, variants, ttl)` | Storage path + variants + مدة الصلاحية | `*SignedURLSet` | توليد URLs موقعة |

### Error Categories

| Category | المعنى | Retryable؟ | مثال |
|----------|--------|-----------|------|
| `config` | إعداد خاطئ | لا | "StorageZone is required" |
| `validation` | صورة مش صالحة | لا | "image exceeds maximum file size" |
| `processing` | فشل في المعالجة | لا | "cwebp encoding failed" |
| `storage` | مشكلة في الـ cloud | ممكن | "HTTP 503 Service Unavailable" |
| `signing` | فشل في التوقيع | لا | "CDN security key not configured" |

### Options

| Option | Default | الوصف |
|--------|---------|-------|
| `WithMaxFileSize(bytes)` | 10MB | الحد الأقصى لحجم الصورة |
| `WithVariants([]VariantConfig)` | thumb/medium/large | أحجام الـ variants |
| `WithRetries(n, delay)` | 3 retries, 1s | عدد المحاولات والتأخير |
| `WithTokenTTL(duration)` | 1 hour | صلاحية الـ signed URLs |
| `WithDedup(bool)` | false | تفعيل فحص التكرار |
| `WithLazyVariants(bool)` | false | رفع الأصلي فقط |
| `WithSmartVariantSkip(bool)` | false | تخطي variants غير مفيدة |
| `WithEncoder(ImageEncoder)` | cwebp binary | encoder مخصص |
| `WithLogger(*slog.Logger)` | discard | structured logging |
| `WithHook(Hook)` | no-op | observability callbacks |
| `WithClock(Clock)` | system time | injectable clock (للتيست) |

---

## ملخص سريع

```
التركيب:  go get github.com/Ibtkary/mediaforge
الرفع:    asset, err := client.Upload(ctx, "tenant", imageBytes)
العرض:    urls, err  := client.GetSignedURL(asset.StoragePath, variants, ttl)
الحذف:    results, _ := client.Delete(ctx, "tenant", filename, variants)
```

**3 أسطر بس عشان تدير صور كاملة!**
