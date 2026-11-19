package uhrp_test

import (
	"encoding/hex"
	"testing"

	base58 "github.com/bsv-blockchain/go-sdk/compat/base58"
	crypto "github.com/bsv-blockchain/go-sdk/primitives/hash"
	"github.com/bsv-blockchain/go-sdk/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/uhrp"
)

const (
	// helloWorldHash is SHA-256("hello world").
	helloWorldHash = "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	// helloWorldURL is the UHRP URL for helloWorldHash, as produced by @bsv/sdk's
	// StorageUtils.getURLForHash (checked against the TypeScript SDK).
	helloWorldURL = "XUUJPXCDnKd2iwUGSFTfTFyhQyxXjvg1ArF7byo58k7cyP5PYT6z"
)

func TestHashData(t *testing.T) {
	assert.Equal(t, helloWorldHash, hex.EncodeToString(uhrp.HashData([]byte("hello world"))))
}

func TestGetURLForHash(t *testing.T) {
	hash, err := hex.DecodeString(helloWorldHash)
	require.NoError(t, err)

	tests := []struct {
		name string
		hash []byte
		want string
	}{
		{name: "sha256 hash", hash: hash, want: helloWorldURL},
		{name: "too short", hash: hash[:31], want: ""},
		{name: "too long", hash: append(append([]byte{}, hash...), 0), want: ""},
		{name: "empty", hash: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, uhrp.GetURLForHash(tt.hash))
		})
	}
}

func TestGetHashFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "bare", url: helloWorldURL},
		{name: "uhrp scheme", url: "uhrp://" + helloWorldURL},
		{name: "web+uhrp scheme", url: "web+uhrp://" + helloWorldURL},
		{name: "upper-case scheme", url: "UHRP://" + helloWorldURL},
		{name: "uhrp scheme without slashes", url: "uhrp:" + helloWorldURL},
		{name: "web+uhrp scheme without slashes", url: "web+uhrp:" + helloWorldURL},
		{name: "slashes without a scheme", url: "//" + helloWorldURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := uhrp.GetHashFromURL(tt.url)
			require.NoError(t, err)
			assert.Equal(t, helloWorldHash, hex.EncodeToString(hash))
		})
	}
}

func TestGetHashFromURLInvalid(t *testing.T) {
	hash, err := hex.DecodeString(helloWorldHash)
	require.NoError(t, err)
	require.Equal(t, helloWorldURL, base58CheckURL(t, []byte{0xce, 0x00}, hash), "base58CheckURL encodes as a UHRP URL")

	tests := []struct {
		name  string
		url   string
		cause error
	}{
		{name: "empty", url: "", cause: storage.ErrInvalidURLLength},
		{name: "not base58", url: "0OIl", cause: storage.ErrInvalidBase58},
		{name: "legacy hex form", url: "uhrp://" + helloWorldHash, cause: storage.ErrInvalidBase58},
		{name: "bad checksum", url: helloWorldURL[:len(helloWorldURL)-1] + "1", cause: storage.ErrInvalidChecksum},
		{name: "bad prefix", url: base58CheckURL(t, []byte{0xce, 0x01}, hash), cause: storage.ErrInvalidURLPrefix},
		{name: "scheme only", url: "uhrp://", cause: storage.ErrInvalidURLLength},
		{name: "too long", url: helloWorldURL + "1", cause: storage.ErrInvalidURLLength},
		{name: "arbitrary text", url: "invalid", cause: storage.ErrInvalidBase58},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := uhrp.GetHashFromURL(tt.url)
			require.ErrorIs(t, err, uhrp.ErrInvalidURL)
			require.ErrorIs(t, err, tt.cause)
			assert.Nil(t, hash)
		})
	}
}

// base58CheckURL encodes prefix and hash the way a UHRP URL is encoded, with a valid checksum.
func base58CheckURL(t *testing.T, prefix, hash []byte) string {
	t.Helper()

	payload := append(append([]byte{}, prefix...), hash...)

	return base58.Encode(append(payload, crypto.Sha256d(payload)[:4]...))
}

func TestURLRoundTrip(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("a"), []byte("hello world"), make([]byte, 4096)} {
		hash := uhrp.HashData(data)

		got, err := uhrp.GetHashFromURL(uhrp.GetURLForHash(hash))
		require.NoError(t, err)
		assert.Equal(t, hash, got)
	}
}
