package processing

import (
	"crypto/sha256"
	"encoding/hex"
)

// ContentHash returns the SHA-256 hex digest of data (64-char lowercase hex string).
func ContentHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HashPrefix returns the first n characters of hash.
// If n >= len(hash) or n <= 0, it returns the full hash.
func HashPrefix(hash string, n int) string {
	if n <= 0 || n >= len(hash) {
		return hash
	}
	return hash[:n]
}
