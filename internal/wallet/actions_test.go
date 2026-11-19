package wallet_test

import (
	"context"
	"encoding/hex"
	"log/slog"
	"testing"

	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

func TestCreateAdvertisement(t *testing.T) {
	priv := testPrivateKey(t)
	pub := priv.PubKey()
	dummySig := testSignature(t, priv)

	var created sdkWallet.CreateActionArgs

	mw := &mocks.MockWallet{
		CreateActionFunc: func(_ context.Context, args sdkWallet.CreateActionArgs, _ string) (*sdkWallet.CreateActionResult, error) {
			created = args
			return &sdkWallet.CreateActionResult{Tx: make([]byte, 100)}, nil
		},
		GetPublicKeyFunc: func(context.Context, sdkWallet.GetPublicKeyArgs, string) (*sdkWallet.GetPublicKeyResult, error) {
			return &sdkWallet.GetPublicKeyResult{PublicKey: pub}, nil
		},
		CreateSignatureFunc: func(context.Context, sdkWallet.CreateSignatureArgs, string) (*sdkWallet.CreateSignatureResult, error) {
			return &sdkWallet.CreateSignatureResult{Signature: dummySig}, nil
		},
	}

	params := wallet.CreateAdParams{
		Hash:          make([]byte, 32),
		URL:           "https://files.example.com/cdn/obj",
		ObjectID:      "obj",
		ContentType:   "text/plain",
		ContentLength: 1024,
		ExpirySecs:    3600,
		Uploader:      uploaderIdentity(t),
	}

	// The mock transaction is not a valid BEEF, so broadcasting fails after the action is created.
	err := newProvider(mw).CreateAdvertisement(t.Context(), overlay.NetworkMainnet, params)
	require.ErrorContains(t, err, "error parsing signed transaction bytes into BEEF")

	require.Len(t, created.Outputs, 1)
	output := created.Outputs[0]
	assert.Equal(t, wallet.BasketName, output.Basket)
	assert.Equal(t, uint64(1), output.Satoshis)
	assert.Equal(t, []string{wallet.BaseAdvertisementLabel}, created.Labels)
	assert.Contains(t, output.Tags, "object_identifier_"+hex.EncodeToString([]byte("obj")))
	assert.Contains(t, output.Tags, "uploader_identity_key_"+params.Uploader)
	assert.Contains(t, output.Tags, "expiry_time_3600")
	assert.Contains(t, output.Tags, "content_type_text/plain")
	assert.Contains(t, output.Tags, "size_1024")
	assert.Contains(t, output.Tags, "name_file")
}

func TestCreateAdvertisementRejectsNegativeValues(t *testing.T) {
	wp := newProvider(&mocks.MockWallet{})

	for _, params := range []wallet.CreateAdParams{
		{Hash: make([]byte, 32), ExpirySecs: -1},
		{Hash: make([]byte, 32), ContentLength: -1},
	} {
		err := wp.CreateAdvertisement(t.Context(), overlay.NetworkMainnet, params)
		require.ErrorIs(t, err, wallet.ErrNegativeValue)
	}
}

func TestAdvertisementOperationsRequireWallet(t *testing.T) {
	wp := wallet.NewProvider("", "", "mainnet", nil, slog.Default())

	require.ErrorIs(t, wp.CreateAdvertisement(t.Context(), overlay.NetworkMainnet, wallet.CreateAdParams{}), wallet.ErrWalletNotInitialized)
	require.ErrorIs(t, wp.RenewAdvertisement(t.Context(), overlay.NetworkMainnet, &sdkWallet.Output{}, nil, wallet.CreateAdParams{}),
		wallet.ErrWalletNotInitialized)
}

func TestRenewAdvertisementRequiresMatchedOutput(t *testing.T) {
	err := newProvider(&mocks.MockWallet{}).RenewAdvertisement(t.Context(), overlay.NetworkMainnet, nil, nil, wallet.CreateAdParams{})
	require.ErrorIs(t, err, wallet.ErrNoMatchedOutput)
}

func TestRenewAdvertisementRejectsNonPushDropOutput(t *testing.T) {
	err := newProvider(&mocks.MockWallet{}).RenewAdvertisement(
		t.Context(), overlay.NetworkMainnet, &sdkWallet.Output{LockingScript: []byte{0x51}}, nil, wallet.CreateAdParams{},
	)
	require.ErrorIs(t, err, wallet.ErrInvalidPushDropScript)
}

func TestRenewAdvertisementWithInvalidBEEF(t *testing.T) {
	priv := testPrivateKey(t)
	pub := priv.PubKey()
	dummySig := testSignature(t, priv)

	zeroHash, err := chainhash.NewHash(make([]byte, 32))
	require.NoError(t, err)

	mw := &mocks.MockWallet{
		CreateActionFunc: func(context.Context, sdkWallet.CreateActionArgs, string) (*sdkWallet.CreateActionResult, error) {
			return &sdkWallet.CreateActionResult{
				SignableTransaction: &sdkWallet.SignableTransaction{
					Tx:        []byte{0}, // Invalid BEEF
					Reference: []byte("ref"),
				},
			}, nil
		},
		GetPublicKeyFunc: func(context.Context, sdkWallet.GetPublicKeyArgs, string) (*sdkWallet.GetPublicKeyResult, error) {
			return &sdkWallet.GetPublicKeyResult{PublicKey: pub}, nil
		},
		CreateSignatureFunc: func(context.Context, sdkWallet.CreateSignatureArgs, string) (*sdkWallet.CreateSignatureResult, error) {
			return &sdkWallet.CreateSignatureResult{Signature: dummySig}, nil
		},
	}

	wp := newProvider(mw)

	// Build a real advertisement locking script to renew.
	lockingScript := advertisementLockingScript(t, wp, mw)

	output := &sdkWallet.Output{
		Outpoint:      transaction.Outpoint{Txid: *zeroHash, Index: 0},
		LockingScript: lockingScript,
		Satoshis:      1,
	}

	params := wallet.CreateAdParams{
		Hash:       make([]byte, 32),
		URL:        "uhrp:test",
		ExpirySecs: 7200,
		Uploader:   uploaderIdentity(t),
	}

	err = wp.RenewAdvertisement(t.Context(), overlay.NetworkMainnet, output, nil, params)
	require.ErrorContains(t, err, "error parsing signable transaction")
}

// advertisementLockingScript creates an advertisement through the provider and returns its locking script.
func advertisementLockingScript(t *testing.T, wp *wallet.Provider, mw *mocks.MockWallet) []byte {
	t.Helper()

	var script []byte

	original := mw.CreateActionFunc
	mw.CreateActionFunc = func(_ context.Context, args sdkWallet.CreateActionArgs, _ string) (*sdkWallet.CreateActionResult, error) {
		script = args.Outputs[0].LockingScript
		return &sdkWallet.CreateActionResult{Tx: []byte{0}}, nil
	}

	// Broadcasting the dummy transaction fails; only the locking script it was built with matters.
	_ = wp.CreateAdvertisement(t.Context(), overlay.NetworkMainnet, wallet.CreateAdParams{
		Hash:          make([]byte, 32),
		URL:           "uhrp:test",
		ExpirySecs:    3600,
		ContentLength: 1024,
	})
	require.NotEmpty(t, script)

	mw.CreateActionFunc = original

	return script
}
