package wallet_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
)

func TestProvider_InitWallet_Failure(t *testing.T) {
	t.Run("Missing Private Key", func(t *testing.T) {
		p := wallet.NewProvider("", "url", "mainnet", nil, slog.Default())
		err := p.InitWallet(context.Background())
		if err == nil {
			t.Error("expected error due to missing private key")
		}
	})

	t.Run("Invalid Private Key", func(t *testing.T) {
		p := wallet.NewProvider("invalid-hex", "url", "mainnet", nil, slog.Default())
		err := p.InitWallet(context.Background())
		if err == nil {
			t.Error("expected error due to invalid private key")
		}
	})
}
