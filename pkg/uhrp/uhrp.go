// Package uhrp provides helpers for UHRP (Universal Hash Resolution Protocol) URLs.
//
// A UHRP URL is the Base58Check encoding of a file's SHA-256 hash with the two-byte
// prefix 0xce00, exactly as StorageUtils.getURLForHash produces it in @bsv/sdk. These
// helpers delegate to go-sdk's storage package, so the URLs match the ones the server
// advertises on the overlay network.
package uhrp

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/bsv-blockchain/go-sdk/storage"
)

// ErrInvalidURL is returned when a string is not a valid UHRP URL. It wraps the go-sdk
// storage error that names the cause, so errors.Is also matches one of:
//   - storage.ErrInvalidBase58: the URL is not valid Base58
//   - storage.ErrInvalidURLLength: the URL does not decode to exactly 38 bytes
//   - storage.ErrInvalidURLPrefix: the 2-byte prefix is not 0xce00
//   - storage.ErrInvalidChecksum: the 4-byte checksum does not match
var ErrInvalidURL = errors.New("invalid UHRP URL")

// HashData computes SHA-256 of the given data and returns the hash bytes.
func HashData(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// GetURLForHash converts a 32-byte SHA-256 hash to a UHRP URL (Base58Check, no scheme prefix).
// It returns an empty string when hash is not 32 bytes long.
func GetURLForHash(hash []byte) string {
	url, err := storage.GetURLForHash(hash)
	if err != nil {
		return ""
	}

	return url
}

// GetHashFromURL extracts the SHA-256 hash from a UHRP URL. The URL may be bare or carry a
// uhrp: or web+uhrp: scheme (in any case), with or without "//", or just a leading "//",
// as the TypeScript SDK's normalizeURL accepts. Any failure (encoding, length, prefix or
// checksum) is returned wrapped in ErrInvalidURL, together with the go-sdk cause.
func GetHashFromURL(uhrpURL string) ([]byte, error) {
	hash, err := storage.GetHashFromURL(uhrpURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}

	return hash, nil
}
