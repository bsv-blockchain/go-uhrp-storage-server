package wallet_test

import (
	"crypto/sha256"
	"log/slog"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

// testPrivateKey returns the deterministic private key 1, used only as a test fixture.
func testPrivateKey(t *testing.T) *ec.PrivateKey {
	t.Helper()

	priv, err := ec.PrivateKeyFromHex("0000000000000000000000000000000000000000000000000000000000000001")
	require.NoError(t, err)

	return priv
}

// uploaderIdentity returns the identity-key hex of a throwaway uploader key.
func uploaderIdentity(t *testing.T) string {
	t.Helper()

	priv, err := ec.PrivateKeyFromHex("0000000000000000000000000000000000000000000000000000000000000002")
	require.NoError(t, err)

	return priv.PubKey().ToDERHex()
}

// testSignature signs a 32-byte digest. go-wallet-toolbox installs a GoBDK
// signer that rejects messages that are not already a SHA-256 digest.
func testSignature(t *testing.T, priv *ec.PrivateKey) *ec.Signature {
	t.Helper()

	digest := sha256.Sum256([]byte("test"))

	sig, err := priv.Sign(digest[:])
	require.NoError(t, err)

	return sig
}

// newProvider returns a mainnet provider backed by the given mock wallet.
func newProvider(mw *mocks.MockWallet) *wallet.Provider {
	wp := wallet.NewProvider("", "", "mainnet", nil, slog.Default())
	wp.SetWallet(mw)

	return wp
}
