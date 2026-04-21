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

func TestFindHandler_ServeHTTP(t *testing.T) {
	pubHex := "026210202604084f83b63a6a978f135bbf32d525701f5c64390757a419241d7c38"
	pub, _ := ec.PublicKeyFromString(pubHex)

	tests := []struct {
		name           string
		uhrpURL        string
		identityKey    *ec.PublicKey
		mockListFunc   func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Unauthorized - No Identity Key",
			uhrpURL:        "uhrp:test",
			identityKey:    nil,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "ERR_UNAUTHORIZED",
		},
		{
			name:           "Bad Request - No uhrpUrl",
			uhrpURL:        "",
			identityKey:    pub,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_NO_UHRP_URL",
		},
		{
			name:        "Not Found - No Advertisement",
			uhrpURL:     "uhrp:notfound",
			identityKey: pub,
			mockListFunc: func(ctx context.Context, args sdkWallet.ListOutputsArgs, walletName string) (*sdkWallet.ListOutputsResult, error) {
				return &sdkWallet.ListOutputsResult{Outputs: []sdkWallet.Output{}}, nil
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "ERR_NOT_FOUND",
		},
		{
			name:        "Success - Valid Find",
			uhrpURL:     "uhrp:success",
			identityKey: pub,
			mockListFunc: func(ctx context.Context, args sdkWallet.ListOutputsArgs, walletName string) (*sdkWallet.ListOutputsResult, error) {
				return &sdkWallet.ListOutputsResult{
					Outputs: []sdkWallet.Output{
						{
							Tags: []string{
								"uhrp_url_756872703a73756363657373", // hex for uhrp:success
								"object_identifier_test-obj",
								"size_1024",
								"content_type_image/png",
								"expiry_time_2000000000",
							},
						},
					},
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &mocks.MockWallet{
				ListOutputsFunc: tt.mockListFunc,
			}
			wp := walletpkg.NewProvider("", "", "", nil, slog.Default())
			wp.SetWallet(mw)

			h := &handlers.FindHandler{WalletProvider: wp, Logger: slog.Default()}

			url := "/find"
			if tt.uhrpURL != "" {
				url += "?uhrpUrl=" + tt.uhrpURL
			}
			req := httptest.NewRequest("GET", url, nil)

			if tt.identityKey != nil {
				ctx := context.WithValue(req.Context(), middlewares.IdentityContextKey, tt.identityKey)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedCode != "" {
				var resp handlers.FindResponse
				json.Unmarshal(w.Body.Bytes(), &resp)
				if resp.Code != tt.expectedCode {
					t.Errorf("expected code %s, got %s", tt.expectedCode, resp.Code)
				}
			}
		})
	}
}
