package mediaforge

import (
	"context"

	"github.com/Ibtkary/mediaforge/internal/pathutil"
	"github.com/Ibtkary/mediaforge/internal/storage"
)

// Delete removes an original image and its variants from storage.
// Individual file errors are captured in each DeleteResult.Err; the method
// error is only returned for setup failures (e.g. invalid tenant ID).
func (c *Client) Delete(ctx context.Context, tenantID, filename string, variantNames []string) ([]DeleteResult, error) {
	const op = "Delete"

	tenantID, err := pathutil.SanitizeTenantID(tenantID)
	if err != nil {
		return nil, ErrValidation("invalid tenant ID").WithOp(op).WithCause(err)
	}

	originalPath := pathutil.BuildStoragePath(tenantID, filename)
	variantPaths := pathutil.ExtractVariantPaths(originalPath, variantNames)

	allPaths := make([]string, 0, 1+len(variantPaths))
	allPaths = append(allPaths, originalPath)
	allPaths = append(allPaths, variantPaths...)

	retryCfg := c.buildRetryConfig()
	results := make([]DeleteResult, 0, len(allPaths))

	for _, path := range allPaths {
		c.logger.Info("deleting", "path", path, "tenant", tenantID)
		c.hook.BeforeDelete(ctx, path)
		delStart := c.clock.Now()
		delErr := storage.WithRetry(ctx, retryCfg, c.shouldRetry, func() error {
			return c.obj.Delete(ctx, path)
		})
		c.hook.AfterDelete(ctx, path, c.clock.Now().Sub(delStart), delErr)
		if delErr != nil {
			c.logger.Error("delete failed", "path", path, "error", delErr)
		}
		results = append(results, DeleteResult{
			Path:    path,
			Success: delErr == nil,
			Err:     delErr,
		})
	}

	return results, nil
}
