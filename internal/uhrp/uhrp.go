package uhrp

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const uhrpPrefix = "uhrp://"

// HashData computes SHA-256 of the given data and returns the hash bytes.
func HashData(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// GetURLForHash converts a SHA-256 hash to a UHRP URL.
// The reference implementation uses StorageUtils.getURLForHash from @bsv/sdk.
func GetURLForHash(hash []byte) string {
	return uhrpPrefix + hex.EncodeToString(hash)
}

// GetHashFromURL extracts the hash bytes from a UHRP URL.
func GetHashFromURL(uhrpURL string) ([]byte, error) {
	if len(uhrpURL) <= len(uhrpPrefix) {
		return nil, fmt.Errorf("invalid UHRP URL: %s", uhrpURL)
	}
	hashStr := uhrpURL[len(uhrpPrefix):]
	return hex.DecodeString(hashStr)
}
