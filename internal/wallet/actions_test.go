package wallet_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

func TestCreateAdvertisement(t *testing.T) {
	priv, _ := ec.PrivateKeyFromHex("0000000000000000000000000000000000000000000000000000000000000001")
	pub := priv.PubKey()
	dummySig, _ := priv.Sign([]byte("test"))

	mw := &mocks.MockWallet{
		CreateActionFunc: func(ctx context.Context, args sdkWallet.CreateActionArgs, originator string) (*sdkWallet.CreateActionResult, error) {
			return &sdkWallet.CreateActionResult{Tx: make([]byte, 100)}, nil
		},
		SignActionFunc: func(ctx context.Context, args sdkWallet.SignActionArgs, originator string) (*sdkWallet.SignActionResult, error) {
			return &sdkWallet.SignActionResult{Tx: make([]byte, 100)}, nil
		},
		GetPublicKeyFunc: func(ctx context.Context, args sdkWallet.GetPublicKeyArgs, originator string) (*sdkWallet.GetPublicKeyResult, error) {
			return &sdkWallet.GetPublicKeyResult{PublicKey: pub}, nil
		},
		CreateSignatureFunc: func(ctx context.Context, args sdkWallet.CreateSignatureArgs, originator string) (*sdkWallet.CreateSignatureResult, error) {
			return &sdkWallet.CreateSignatureResult{Signature: dummySig}, nil
		},
	}

	params := wallet.CreateAdParams{
		Hash:          make([]byte, 32),
		URL:           "uhrp:test",
		ObjectID:      "obj",
		ContentLength: 1024,
		ExpirySecs:    3600,
		Uploader:      "uploader",
	}

	wp := wallet.NewProvider("", "", "mainnet", nil, slog.Default())
	wp.SetWallet(mw)
	err := wp.CreateAdvertisement(context.Background(), overlay.NetworkMainnet, params)
	// We expect a parsing error because our mock BEEF is just zeros,
	// but this proves the logic reached overlayBroadcast.
	if err != nil && !strings.Contains(err.Error(), "error parsing signed transaction bytes into BEEF") &&
		!strings.Contains(err.Error(), "nil client") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRenewAdvertisement_Simple(t *testing.T) {
	priv, _ := ec.PrivateKeyFromHex("0000000000000000000000000000000000000000000000000000000000000001")
	pub := priv.PubKey()
	dummySig, _ := priv.Sign([]byte("test"))

	uhrpURL := "uhrp:test"
	uploaderKey := "02cbc1404c96562479633e721b0e01476d05ebecfc6797a7e9df533f81daed48ed"
	zeroHash, _ := chainhash.NewHash(make([]byte, 32))

	mw := &mocks.MockWallet{
		CreateActionFunc: func(ctx context.Context, args sdkWallet.CreateActionArgs, originator string) (*sdkWallet.CreateActionResult, error) {
			return &sdkWallet.CreateActionResult{
				SignableTransaction: &sdkWallet.SignableTransaction{
					Tx:        []byte{0}, // Invalid BEEF
					Reference: []byte("ref"),
				},
			}, nil
		},
		GetPublicKeyFunc: func(ctx context.Context, args sdkWallet.GetPublicKeyArgs, originator string) (*sdkWallet.GetPublicKeyResult, error) {
			return &sdkWallet.GetPublicKeyResult{PublicKey: pub}, nil
		},
		CreateSignatureFunc: func(ctx context.Context, args sdkWallet.CreateSignatureArgs, originator string) (*sdkWallet.CreateSignatureResult, error) {
			return &sdkWallet.CreateSignatureResult{Signature: dummySig}, nil
		},
	}

	output := &sdkWallet.Output{
		Outpoint: transaction.Outpoint{Txid: *zeroHash, Index: 0},
	}

	params := wallet.CreateAdParams{
		Hash:     make([]byte, 32),
		URL:      uhrpURL,
		Uploader: uploaderKey,
	}

	wp := wallet.NewProvider("", "", "mainnet", nil, slog.Default())
	wp.SetWallet(mw)
	err := wp.RenewAdvertisement(context.Background(), overlay.NetworkMainnet, output, nil, params)
	if err == nil {
		t.Error("expected error due to invalid BEEF, got nil")
	}
}
