package pathutil

import (
	"fmt"
	"regexp"
	"strings"
)

var validTenantIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// SanitizeTenantID validates and returns a cleaned tenant ID.
// Allowed characters: a-z, A-Z, 0-9, hyphen, underscore.
// Rejects: empty, contains "..", "/", "\", or spaces.
//
// Returns plain error (not *mediastore.Error) — internal package constraint.
// Public API callers must wrap returned errors into the library's Error type.
func SanitizeTenantID(id string) (string, error) {
	if id == "" {
		return "", fmt.Errorf("tenant ID must not be empty")
	}
	if strings.Contains(id, "..") {
		return "", fmt.Errorf("tenant ID must not contain path traversal: %q", id)
	}
	if strings.Contains(id, "/") || strings.Contains(id, "\\") {
		return "", fmt.Errorf("tenant ID must not contain path separators: %q", id)
	}
	if strings.Contains(id, " ") {
		return "", fmt.Errorf("tenant ID must not contain spaces: %q", id)
	}
	if !validTenantIDPattern.MatchString(id) {
		return "", fmt.Errorf("tenant ID contains invalid characters: %q", id)
	}
	return id, nil
}

// BuildStoragePath constructs the storage path for a file.
// Output: "stores/{tenantID}/images/{filename}"
func BuildStoragePath(tenantID, filename string) string {
	return fmt.Sprintf("stores/%s/images/%s", tenantID, filename)
}

// BuildOriginalFilename constructs the filename for an original image.
// Output: "{hashPrefix}_{uuidPrefix}.webp"
func BuildOriginalFilename(hashPrefix, uuidPrefix string) string {
	return fmt.Sprintf("%s_%s.webp", hashPrefix, uuidPrefix)
}

// BuildVariantFilename creates a variant filename from a base filename.
// Input: "e3b0c44298fc_a1b2c3d4.webp", "thumb"
// Output: "e3b0c44298fc_a1b2c3d4_thumb.webp"
func BuildVariantFilename(baseFilename, variantName string) string {
	ext := ".webp"
	base := strings.TrimSuffix(baseFilename, ext)
	return fmt.Sprintf("%s_%s%s", base, variantName, ext)
}

// BuildCDNURL constructs the full CDN URL for a storage path.
// Input: "https://cdn.nafees.com", "stores/t1/images/abc.webp"
// Output: "https://cdn.nafees.com/stores/t1/images/abc.webp"
func BuildCDNURL(cdnBase, storagePath string) string {
	cdnBase = strings.TrimRight(cdnBase, "/")
	storagePath = strings.TrimLeft(storagePath, "/")
	return cdnBase + "/" + storagePath
}

// ExtractVariantPaths returns all variant paths for a given base storage path.
func ExtractVariantPaths(basePath string, variantNames []string) []string {
	dir, file := splitPath(basePath)
	paths := make([]string, 0, len(variantNames))
	for _, name := range variantNames {
		variantFile := BuildVariantFilename(file, name)
		if dir == "" {
			paths = append(paths, variantFile)
		} else {
			paths = append(paths, dir+"/"+variantFile)
		}
	}
	return paths
}

// splitPath splits a path into directory and filename.
func splitPath(path string) (string, string) {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "", path
	}
	return path[:idx], path[idx+1:]
}
