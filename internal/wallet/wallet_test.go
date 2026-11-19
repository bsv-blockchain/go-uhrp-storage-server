package wallet_test

import (
	"log/slog"
	"testing"

	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

func TestProviderInitWalletFailure(t *testing.T) {
	t.Run("missing private key", func(t *testing.T) {
		err := wallet.NewProvider("", "url", "mainnet", nil, slog.Default()).InitWallet(t.Context())
		require.ErrorIs(t, err, wallet.ErrMissingServerPrivateKey)
	})

	t.Run("invalid private key", func(t *testing.T) {
		err := wallet.NewProvider("invalid-hex", "url", "mainnet", nil, slog.Default()).InitWallet(t.Context())
		require.ErrorContains(t, err, "invalid server private key")
	})
}

func TestProviderInitWalletIsIdempotent(t *testing.T) {
	mw := &mocks.MockWallet{}

	p := wallet.NewProvider("", "", "mainnet", nil, slog.Default())
	p.SetWallet(mw)

	// An already-initialized provider returns early, before validating any configuration.
	require.NoError(t, p.InitWallet(t.Context()))
	assert.Same(t, mw, p.GetWallet())
}

func TestProviderOverlayNetwork(t *testing.T) {
	tests := []struct {
		network string
		want    overlay.Network
	}{
		{network: "mainnet", want: overlay.NetworkMainnet},
		{network: "testnet", want: overlay.NetworkTestnet},
		{network: "", want: overlay.NetworkMainnet},
		{network: "unknown", want: overlay.NetworkMainnet},
	}

	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			p := wallet.NewProvider("", "", tt.network, nil, slog.Default())
			assert.Equal(t, tt.want, p.OverlayNetwork())
		})
	}
}

func TestProviderGetWalletBeforeInit(t *testing.T) {
	assert.Nil(t, wallet.NewProvider("", "", "mainnet", nil, slog.Default()).GetWallet())
}
