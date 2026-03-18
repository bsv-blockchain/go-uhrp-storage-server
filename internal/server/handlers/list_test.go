package handlers_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

func TestListHandler_ServeHTTP(t *testing.T) {
	pubHex := "026210202604084f83b63a6a978f135bbf32d525701f5c64390757a419241d7c38"
	pub, _ := ec.PublicKeyFromString(pubHex)

	tests := []struct {
		name           string
		identityKey    *ec.PublicKey
		mockListFunc   func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error)
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "Unauthorized",
			identityKey:    nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:        "Success - Empty List",
			identityKey: pub,
			mockListFunc: func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error) {
				return &sdkWallet.ListOutputsResult{Outputs: []sdkWallet.Output{}}, nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:        "Success - Multiple Items",
			identityKey: pub,
			mockListFunc: func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error) {
				return &sdkWallet.ListOutputsResult{
					Outputs: []sdkWallet.Output{
						{
							Tags: []string{
								"uhrp_url_756872703a2f2f6851776361524338353174757a427067785a6b714455437732345a7662775337426e6447546248356e6364685567797a50",
								"expiry_time_2000000000",
							},
						},
						{
							Tags: []string{
								"uhrp_url_756872703a2f2f524535714b4a4b4467616d4453737133695839453239714e6636517442644848456d7168326564354d7748637963694a4e",
								"expiry_time_2000000000",
							},
						},
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &mocks.MockWallet{
				ListOutputsFunc: tt.mockListFunc,
			}
			wp := walletpkg.NewProvider("", "", "", slog.Default())
			wp.SetWallet(mw)

			h := &handlers.ListHandler{WalletProvider: wp, Logger: slog.Default()}

			req := httptest.NewRequest("GET", "/list", nil)
			if tt.identityKey != nil {
				ctx := context.WithValue(req.Context(), middlewares.IdentityContextKey, tt.identityKey)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var resp handlers.ListResponse
				json.Unmarshal(w.Body.Bytes(), &resp)
				if len(resp.Uploads) != tt.expectedCount {
					t.Errorf("expected %d uploads, got %d", tt.expectedCount, len(resp.Uploads))
				}
			}
		})
	}
}
