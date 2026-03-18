package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

func TestRequestPriceCalculator(t *testing.T) {
	pubHex := "026210202604084f83b63a6a978f135bbf32d525701f5c64390757a419241d7c38"
	pub, _ := ec.PublicKeyFromString(pubHex)

	calc := pricing.NewCalculator(0.03, mockOracle{})
	wp := walletpkg.NewProvider("", "", "", slog.Default())

	fn := handlers.RequestPriceCalculator(calc, wp)

	t.Run("Upload Price", func(t *testing.T) {
		reqBody := handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 60}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/upload", bytes.NewReader(bodyBytes))

		price, err := fn(req)
		if err != nil {
			t.Fatal(err)
		}
		// Expected Satoshis (Calculated):
		// USD = (1024 / 1e9) * (60 / 43200) * 0.03 = 0.00000000004266...
		// Satoshis = USD * (1e8 / 30) = 0.00000000004266... * 3333333.33 = 0.000142...
		// Minimum is 10.
		if price != 10 {
			t.Errorf("expected price 10, got %d", price)
		}
	})

	t.Run("Renew Price", func(t *testing.T) {
		mw := &mocks.MockWallet{
			ListOutputsFunc: func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error) {
				return &sdkWallet.ListOutputsResult{
					Outputs: []sdkWallet.Output{
						{
							Tags: []string{"uhrp_url_test", "size_1024", "expiry_time_2000000000"},
						},
					},
				}, nil
			},
		}
		wp.SetWallet(mw)

		reqBody := handlers.RenewRequest{UhrpURL: "test", AdditionalMinutes: 60}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/renew", bytes.NewReader(bodyBytes))
		ctx := context.WithValue(req.Context(), middlewares.IdentityContextKey, pub)
		req = req.WithContext(ctx)

		price, err := fn(req)
		if err != nil {
			t.Fatal(err)
		}
		if price != 10 {
			t.Errorf("expected price 10, got %d", price)
		}
	})

	t.Run("Other Route", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/list", nil)
		price, err := fn(req)
		if err != nil {
			t.Fatal(err)
		}
		if price != 0 {
			t.Errorf("expected price 0, got %d", price)
		}
	})
}
