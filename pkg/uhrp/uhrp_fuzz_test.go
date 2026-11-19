package uhrp_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/uhrp"
)

// FuzzGetHashFromURL checks that parsing never panics, that every failure wraps ErrInvalidURL,
// and that every accepted URL is a 32-byte hash that encodes back to the same URL.
func FuzzGetHashFromURL(f *testing.F) {
	for _, seed := range []string{
		helloWorldURL,
		"uhrp://" + helloWorldURL,
		"web+uhrp://" + helloWorldURL,
		"uhrp://" + helloWorldHash,
		"",
		"uhrp://",
		"0OIl",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, url string) {
		hash, err := uhrp.GetHashFromURL(url)
		if err != nil {
			require.ErrorIs(t, err, uhrp.ErrInvalidURL)
			return
		}

		require.Len(t, hash, 32)

		bare := url
		for _, prefix := range []string{"web+uhrp://", "uhrp://"} {
			if strings.HasPrefix(strings.ToLower(bare), prefix) {
				bare = bare[len(prefix):]
				break
			}
		}

		assert.Equal(t, bare, uhrp.GetURLForHash(hash))
	})
}

// FuzzURLRoundTrip checks that any data's UHRP URL decodes back to its SHA-256 hash.
func FuzzURLRoundTrip(f *testing.F) {
	f.Add([]byte("hello world"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		hash := uhrp.HashData(data)

		got, err := uhrp.GetHashFromURL(uhrp.GetURLForHash(hash))
		require.NoError(t, err)
		assert.Equal(t, hash, got)
	})
}
