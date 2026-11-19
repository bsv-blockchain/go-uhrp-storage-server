package uhrp_test

import (
	"encoding/hex"
	"testing"

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
	tests := []struct {
		name string
		url  string
	}{
		{name: "empty", url: ""},
		{name: "not base58", url: "0OIl"},
		{name: "legacy hex form", url: "uhrp://" + helloWorldHash},
		{name: "bad checksum", url: helloWorldURL[:len(helloWorldURL)-1] + "1"},
		{name: "scheme only", url: "uhrp://"},
		{name: "arbitrary text", url: "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := uhrp.GetHashFromURL(tt.url)
			require.ErrorIs(t, err, uhrp.ErrInvalidURL)
			assert.Nil(t, hash)
		})
	}
}

func TestURLRoundTrip(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("a"), []byte("hello world"), make([]byte, 4096)} {
		hash := uhrp.HashData(data)

		got, err := uhrp.GetHashFromURL(uhrp.GetURLForHash(hash))
		require.NoError(t, err)
		assert.Equal(t, hash, got)
	}
}
