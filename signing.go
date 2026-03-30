package mediaforge

import (
	"time"

	"github.com/Ibtkary/mediaforge/internal/pathutil"
)

// GetSignedURL generates signed CDN URLs for an asset and its variants.
// If ttl <= 0, the configured DefaultTokenTTL is used.
func (c *Client) GetSignedURL(storagePath string, variantNames []string, ttl time.Duration) (*SignedURLSet, error) {
	const op = "GetSignedURL"

	if storagePath == "" {
		return nil, ErrValidation("storage path must not be empty").WithOp(op)
	}

	if ttl <= 0 {
		ttl = c.config.DefaultTokenTTL
	}

	expires := c.clock.Now().Add(ttl)

	// Sign original
	originalURL, err := c.signer.SignURL(storagePath, expires)
	if err != nil {
		return nil, ErrSigning("failed to sign original URL").WithOp(op).WithCause(err)
	}

	// Sign variants
	variantPaths := pathutil.ExtractVariantPaths(storagePath, variantNames)
	variantURLs := make(map[string]string, len(variantNames))

	for i, vp := range variantPaths {
		signedURL, err := c.signer.SignURL(vp, expires)
		if err != nil {
			return nil, ErrSigning("failed to sign variant URL").WithOp(op).WithDetail(variantNames[i]).WithCause(err)
		}
		variantURLs[variantNames[i]] = signedURL
	}

	return &SignedURLSet{
		Original: originalURL,
		Variants: variantURLs,
	}, nil
}
