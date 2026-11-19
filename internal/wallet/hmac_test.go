package wallet_test

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"testing"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

var errHMACFailed = errors.New("hmac failed")

func TestCreateUploaderHMAC(t *testing.T) {
	var got sdkWallet.CreateHMACArgs

	mw := &mocks.MockWallet{
		CreateHMACFunc: func(_ context.Context, args sdkWallet.CreateHMACArgs, _ string) (*sdkWallet.CreateHMACResult, error) {
			got = args
			return &sdkWallet.CreateHMACResult{HMAC: [32]byte{0xab, 0xcd}}, nil
		},
	}

	hmac, err := newProvider(mw).CreateUploaderHMAC(t.Context(), "fileSize=1")
	require.NoError(t, err)
	assert.Equal(t, "abcd"+strings.Repeat("00", 30), hmac)

	assert.Equal(t, []byte("fileSize=1"), []byte(got.Data))
	assert.Equal(t, wallet.StorageUploadProtocol, got.ProtocolID.Protocol)
	assert.Equal(t, wallet.AnyonesKeyID, got.KeyID)
	assert.Equal(t, sdkWallet.CounterpartyTypeSelf, got.Counterparty.Type)
}

func TestCreateUploaderHMACErrors(t *testing.T) {
	_, err := wallet.NewProvider("", "", "mainnet", nil, slog.Default()).CreateUploaderHMAC(t.Context(), "x")
	require.ErrorIs(t, err, wallet.ErrWalletNotInitialized)

	mw := &mocks.MockWallet{
		CreateHMACFunc: func(context.Context, sdkWallet.CreateHMACArgs, string) (*sdkWallet.CreateHMACResult, error) {
			return nil, errHMACFailed
		},
	}

	_, err = newProvider(mw).CreateUploaderHMAC(t.Context(), "x")
	require.ErrorIs(t, err, errHMACFailed)
}

func TestVerifyUploaderHMAC(t *testing.T) {
	validHMAC := hex.EncodeToString(make([]byte, 32))

	tests := []struct {
		name    string
		hmac    string
		result  *sdkWallet.VerifyHMACResult
		err     error
		wantErr error
	}{
		{name: "valid", hmac: validHMAC, result: &sdkWallet.VerifyHMACResult{Valid: true}},
		{name: "does not verify", hmac: validHMAC, result: &sdkWallet.VerifyHMACResult{Valid: false}, wantErr: wallet.ErrInvalidHMAC},
		{name: "not hex", hmac: "zz", wantErr: wallet.ErrInvalidHMACFormat},
		{name: "wrong length", hmac: "abcd", wantErr: wallet.ErrInvalidHMACFormat},
		{name: "wallet error", hmac: validHMAC, err: errHMACFailed, wantErr: errHMACFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotData []byte

			mw := &mocks.MockWallet{
				VerifyHMACFunc: func(_ context.Context, args sdkWallet.VerifyHMACArgs, _ string) (*sdkWallet.VerifyHMACResult, error) {
					gotData = args.Data
					return tt.result, tt.err
				},
			}

			err := newProvider(mw).VerifyUploaderHMAC(t.Context(), "5", "obj", "2030-01-01T00:00:00Z", "key", tt.hmac)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "fileSize=5&objectID=obj&expiry=2030-01-01T00:00:00Z&uploader=key", string(gotData))
		})
	}
}

func TestVerifyUploaderHMACRequiresWallet(t *testing.T) {
	err := wallet.NewProvider("", "", "mainnet", nil, slog.Default()).VerifyUploaderHMAC(t.Context(), "", "", "", "", "")
	require.ErrorIs(t, err, wallet.ErrWalletNotInitialized)
}
