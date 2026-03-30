# ADR-001: Mediastore Library Architecture Decisions

**Status:** Proposed
**Date:** 2026-03-27
**Deciders:** Backend team lead, DevOps

---

## Context

Mediastore is a standalone Go library for managing product images in the Nafees JMS (Jewelry Management System). It was originally built as a Bunny.net-only library (103 tests, 95% coverage). A refactoring was performed to make it provider-agnostic while adding observability, cost optimizations, and io.Reader support.

The Go backend currently uses AWS S3 via a custom `pkg/s3/s3_service.go`, while this library targets Bunny.net. The refactoring needs to bridge this gap and establish the canonical architecture going forward.

**Forces at play:**
- Backend uses S3, library was built for Bunny.net
- Library must be reusable across projects
- 103 existing tests must not break (backward compatibility)
- Image processing requires full decode (cannot stream)
- External `cwebp` binary dependency limits deployment portability

---

## Decisions

### Decision 1: Provider Abstraction via Small Interfaces

**Decision:** Three small interfaces abstract storage, signing, and encoding.

```go
ObjectStorage  { Upload, Delete, Exists }     // 3 methods
URLSigner      { SignURL }                     // 1 method
ImageEncoder   { Encode, Extension, ContentType } // 3 methods
```

**Why this is the best practice:**

| Alternative | Problem |
|-------------|---------|
| Single `Provider` interface with all methods | Violates Interface Segregation; forces S3 to implement signing it handles differently |
| Concrete types only | No way to swap providers without rewriting |
| Plugin/driver pattern (like database/sql) | Overkill for 3 implementations; adds registry complexity |

**The Go stdlib pattern we follow:** `io.Reader` (1 method), `io.Writer` (1 method) — small interfaces composed by the caller.

**Consequences:**
- Each provider implements only what it needs
- S3 provider doesn't need to understand Bunny token signing
- Testing uses simple mocks (no mock frameworks needed)
- Adding Cloudflare R2 or MinIO requires only 2 new types

---

### Decision 2: Three Constructors (Backward Compat + Provider-Agnostic)

**Decision:** Keep `New(Config)` for backward compat, add `NewClient()` and `NewBunnyClient()`.

```go
New(cfg Config) (*Client, error)           // Legacy Bunny-only (unchanged)
NewClient(storage, signer, opts) (*Client, error)  // Provider-agnostic
NewBunnyClient(zone, key, cdn, cdnKey, opts) (*Client, error) // Bunny + options
```

**Why not break the API:**

The library has 103 existing tests using `New(Config)`. Breaking the API forces all consumers to rewrite simultaneously. The cost of maintaining one legacy constructor is negligible vs. the coordination cost of a breaking change.

**Migration path:**
1. **Existing code:** Keep using `New(Config)` — zero changes needed
2. **New Bunny users:** Use `NewBunnyClient()` with functional options
3. **S3/other providers:** Use `NewClient(s3Storage, s3Signer, opts...)`
4. **Future:** Deprecate `New(Config)` via `// Deprecated:` comment when all consumers migrate

**When to revisit:** When the last consumer of `New(Config)` migrates to `NewBunnyClient()`.

---

### Decision 3: Functional Options Pattern

**Decision:** Use `func(*clientOptions)` pattern for configuration.

```go
client, err := mediaforge.NewClient(storage, signer,
    mediaforge.WithMaxFileSize(5 * 1024 * 1024),
    mediaforge.WithVariants(mediaforge.DefaultVariants()),
    mediaforge.WithRetries(3, time.Second),
)
```

**Why this over alternatives:**

| Pattern | Problem |
|---------|---------|
| Builder pattern | Verbose, requires method chaining, hard to extend |
| Config struct only | Can't distinguish "not set" from "set to zero" |
| Environment variables | Not composable, testing-hostile |
| YAML/TOML config file | Not embeddable in library |

**Precedence chain:**
```
Explicit option → applyToConfig → applyDefaults → validateNonProvider
```

An explicit `WithMaxFileSize(5MB)` always wins over the 10MB default.

**Known tension:** `MaxRetries` uses `*int` (pointer) to distinguish "not set" from "set to 0". Other options use zero-value detection. This inconsistency is acceptable because MaxRetries=0 is a valid and common value, while MaxFileSize=0 is never valid.

---

### Decision 4: Retry in the Core, Not Per-Provider

**Decision:** The core `storage.WithRetry()` wraps all provider calls. Providers just return errors.

```
Provider.Upload() → error (with optional RetryableError interface)
    ↓
core: storage.WithRetry(ctx, retryCfg, shouldRetry, provider.Upload)
    ↓
shouldRetry(err) checks: RetryableError interface OR StorageError.Retryable
```

**Why not per-provider retry:**

| Approach | Problem |
|----------|---------|
| Provider handles its own retry | Duplicated retry logic in every provider; inconsistent backoff |
| No retry at all | Transient failures (HTTP 503, network blip) cause unnecessary errors |
| Middleware/decorator pattern | Adds complexity; retry config needs to be provider-aware |

**The chosen approach:**
- Core owns the retry loop (exponential backoff + jitter, configurable via `WithRetries`)
- Provider signals retryability via `RetryableError` interface (optional, duck-typed)
- S3 provider uses `smithy.APIError.ErrorCode()` for structured retry decisions
- Bunny provider uses HTTP status codes (5xx = retryable)

**Backoff formula:**
```
delay = min(baseDelay * 2^attempt, 30s) ± 25% jitter
```
Default: 3 retries, 1s base → attempts at ~0s, ~1s, ~2s, ~4s (max ~7s total).

---

### Decision 5: Encoder Interface (Not Binary Lock-in)

**Decision:** `cwebp` binary is the default encoder, behind an `ImageEncoder` interface.

**Why not lock to cwebp:**
- CI/CD environments may not have `cwebp` installed
- WebP encoding has pure Go alternatives (`golang.org/x/image/webp`)
- AVIF may be preferred in the future
- Testing needs deterministic output (mock encoder)

**Why keep cwebp as default:**
- Better compression quality than pure Go
- Faster encoding for large images
- Production-proven (already deployed)

**Fallback chain:**
```
WithEncoder(custom) → custom encoder
nil encoder         → CWebPEncoder{BinaryPath: cfg.CWebPPath}
```

**The cwebp check:**
- `New()` and `NewBunnyClient()` check binary at construction time (fast fail)
- `NewClient()` skips check if custom encoder provided
- `NewClient()` with nil encoder defers failure to first `Upload()` call

**Recommendation:** This deferred failure is acceptable. The alternative (requiring encoder always) would break the ergonomics of the simple case where cwebp is in PATH.

---

### Decision 6: Separate Module for S3 Provider

**Decision:** `providers/s3/` is a separate Go module with its own `go.mod`.

```
packages/mediaforge/
├── go.mod                    # core: uuid, x/image, testify only
└── providers/s3/
    └── go.mod                # s3: aws-sdk-go-v2 + core
```

**Why not one module:**

| Approach | Problem |
|----------|---------|
| Single module | Everyone gets aws-sdk-go-v2 (47+ transitive deps) even if using Bunny |
| Build tags | Complex, error-prone, IDE-unfriendly |
| Separate repo | Versioning overhead, harder to develop together |

**Separate module in same repo** gives us:
- Core stays at 3 direct dependencies (uuid, x/image, testify)
- S3 users opt-in to AWS SDK by importing the provider module
- Both modules develop together, tested together
- `replace` directive in S3 go.mod points to `../../` for local development

---

### Decision 7: Cost Optimization as Opt-In Features

**Decision:** Dedup, lazy variants, and smart variant skip are all disabled by default.

```go
WithDedup(true)            // HEAD request before each upload
WithLazyVariants(true)     // Upload original only
WithSmartVariantSkip(true) // Skip variants where source fits
```

**Why disabled by default:**
- **Dedup:** Adds latency (HEAD request per upload). Not useful if uploads are always unique.
- **Lazy variants:** Requires a separate `GenerateVariants()` call later. Breaking change if caller expects variants immediately.
- **Smart skip:** Changes the number of files stored. May confuse monitoring/billing if enabled unexpectedly.

**Why opt-in per-feature (not a single "optimize" flag):**
- Each feature has different trade-offs
- Some combinations are dangerous (lazy + dedup = minimal writes but no variants ever)
- Explicit is better than implicit

---

### Decision 8: Sequential Variant Upload (Not Concurrent)

**Decision:** Variants are uploaded one at a time, not concurrently.

**Why not concurrent:**

| Approach | Benefit | Risk |
|----------|---------|------|
| Sequential | Simple, predictable, cleanup on failure works | Slower for many variants |
| Concurrent (goroutines) | Faster for 4+ variants | Complex error handling, partial failure cleanup, memory spike |
| Worker pool | Bounded concurrency | Same complexity as concurrent, plus pool management |

**Current math:**
- 4 files per image (original + 3 variants)
- Each file ~50-300KB (WebP compressed)
- Upload time dominated by network, not CPU
- Sequential: ~4 × RTT. Concurrent: ~1 × RTT + overhead

**When to revisit:** If variant count exceeds 6 or RTT exceeds 500ms, concurrent upload with bounded goroutines (e.g., `errgroup` with limit) would be worth the complexity.

---

### Decision 9: Hooks for Observability (Not Middleware)

**Decision:** A `Hook` interface with Before/After callbacks, not a middleware chain.

```go
type Hook interface {
    BeforeUpload(ctx, key, size)
    AfterUpload(ctx, key, size, dur, err)
    BeforeDelete(ctx, key)
    AfterDelete(ctx, key, dur, err)
}
```

**Why not middleware/interceptor:**

| Pattern | Problem |
|---------|---------|
| `http.RoundTripper`-style wrapping | Storage operations aren't HTTP-only; S3 uses SDK, not raw HTTP |
| Event emitter | Requires registration, lifecycle management, goroutine safety |
| Aspect-oriented | Go doesn't have AOP; would need code generation |

**Why hooks are sufficient:**
- 4 callbacks cover all observable operations
- Caller has full context (key, size, duration, error)
- `NoopHook` exported for embedding (override only what you need)
- Zero overhead when not set (noopHook methods are inlined by compiler)

**Known limitation:** Hook interface requires all 4 methods. Mitigated by:
```go
type MyHook struct { mediaforge.NoopHook }
func (h *MyHook) AfterUpload(ctx, key, size, dur, err) { /* only this */ }
```

---

### Decision 10: Structured Errors with Categories

**Decision:** Single `*Error` type with `ErrorCategory` enum, not sentinel errors.

```go
type Error struct {
    Category  ErrorCategory  // config|validation|processing|storage|signing
    Msg       string
    Retryable bool
    Op        string
    Detail    string
    err       error          // wrapped cause
}
```

**Why not sentinel errors (`var ErrNotFound = errors.New(...)`):**
- Sentinel errors don't carry context (no operation name, no detail)
- Can't distinguish "validation failed in Upload" from "validation failed in Delete"
- Retryability is per-error, not per-type

**Why not error codes (int):**
- Strings are self-documenting
- `errors.Is()` matching by category is natural:
  ```go
  if errors.Is(err, &Error{Category: CategoryStorage}) { retry() }
  ```

**Error chain:**
```
ErrStorage("storage operation failed", retryable=true)
    .WithOp("Upload")
    .WithDetail("uploading variant thumb")
    .WithCause(originalHTTPError)
```

**`IsRetryable()` walks the full chain** — if any error in the chain is retryable, the whole chain is retryable. This handles wrapping correctly.

---

## Trade-off Summary

| Decision | What gets easier | What gets harder |
|----------|-----------------|------------------|
| Provider interfaces | Swapping backends, testing | Understanding 3 constructors |
| Functional options | Adding new config | Zero-value ambiguity |
| Core retry | Consistent behavior | Provider can't customize backoff |
| Encoder interface | CI/testing without cwebp | Extra abstraction layer |
| Separate S3 module | Minimal core deps | Module versioning coordination |
| Opt-in cost features | Safe defaults | Must read docs to optimize |
| Sequential uploads | Simple cleanup | Slower for many variants |
| Hook interface | Zero-overhead observability | Must implement all 4 methods |
| Structured errors | Rich error context | Custom error type to learn |

---

## Action Items

1. [ ] Add `// Deprecated:` comment to `New(Config)` after backend migrates to `NewBunnyClient()`
2. [ ] Implement `GenerateVariants()` method for lazy variant use case
3. [ ] Add `context.Context` to `URLSigner.SignURL()` in next major version
4. [ ] Consider `errgroup`-based concurrent upload when variant count grows
5. [ ] Write migration guide: `New(Config)` → `NewBunnyClient()` → `NewClient()`
6. [ ] Benchmark memory usage: current (all variants in memory) vs. process-per-variant

---

## Appendix: Architecture Diagram

```
                         ┌────────────────────┐
                         │   Consumer Code    │
                         │  (JMS Backend)     │
                         └────────┬───────────┘
                                  │
              ┌───────────────────┼───────────────────┐
              │                   │                   │
    ┌─────────┴──────┐  ┌────────┴────────┐  ┌───────┴───────┐
    │  Upload(ctx,   │  │  Delete(ctx,    │  │ GetSignedURL  │
    │  tenant, data) │  │  tenant, file)  │  │ (path, ttl)   │
    └───────┬────────┘  └───────┬─────────┘  └───────┬───────┘
            │                   │                     │
    ┌───────┴────────┐          │              ┌──────┴──────┐
    │  Processing    │          │              │  URLSigner  │
    │  Pipeline      │          │              │  interface  │
    │  (internal)    │          │              └──────┬──────┘
    │                │          │                     │
    │ validate       │    ┌─────┴──────┐      ┌──────┴──────┐
    │ hash (SHA-256) │    │            │      │ BunnySigner │
    │ decode         │    │ Object     │      │ S3 Signer   │
    │ resize         │    │ Storage    │      └─────────────┘
    │ encode (WebP)  │    │ interface  │
    └────────────────┘    │            │
                          └─────┬──────┘
                                │
                    ┌───────────┼───────────┐
                    │           │           │
              ┌─────┴────┐ ┌───┴────┐ ┌────┴─────┐
              │  Bunny   │ │  S3    │ │  Custom  │
              │  Adapter │ │Storage │ │  Impl    │
              └──────────┘ └────────┘ └──────────┘
```

**Data flow:**
```
Image bytes → Validate → Hash → Decode → Resize → Encode → Upload (sequential per variant)
```

**Error flow:**
```
Provider error → RetryableError check → WithRetry (exponential backoff) → wrapStorageError → *Error
```
