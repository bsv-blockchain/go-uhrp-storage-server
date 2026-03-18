package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
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

func TestUploadHandler_ServeHTTP(t *testing.T) {
	pubHex := "026210202604084f83b63a6a978f135bbf32d525701f5c64390757a419241d7c38"
	pub, _ := ec.PublicKeyFromString(pubHex)

	tests := []struct {
		name           string
		identityKey    *ec.PublicKey
		body           interface{}
		mockHMACFunc   func(ctx context.Context, args sdkWallet.CreateHMACArgs, originator string) (*sdkWallet.CreateHMACResult, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Unauthorized",
			identityKey:    nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Body",
			identityKey:    pub,
			body:           "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_BODY",
		},
		{
			name:           "Invalid File Size",
			identityKey:    pub,
			body:           handlers.UploadRequest{FileSize: 0, RetentionPeriod: 60},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_SIZE",
		},
		{
			name:           "Invalid Retention Period",
			identityKey:    pub,
			body:           handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 0},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_NO_RETENTION_PERIOD",
		},
		{
			name:           "Retention Period Too Small",
			identityKey:    pub,
			body:           handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 5},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_RETENTION_PERIOD",
		},
		{
			name:        "Success",
			identityKey: pub,
			body:        handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 60},
			mockHMACFunc: func(ctx context.Context, args sdkWallet.CreateHMACArgs, originator string) (*sdkWallet.CreateHMACResult, error) {
				return &sdkWallet.CreateHMACResult{HMAC: [32]byte{1, 2, 3}}, nil
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &mocks.MockWallet{
				CreateHMACFunc: tt.mockHMACFunc,
			}
			wp := walletpkg.NewProvider("", "", "")
			wp.SetWallet(mw)

			calc := pricing.NewCalculator(0.03, mockOracle{})
			h := &handlers.UploadHandler{
				Calculator:        calc,
				WalletProvider:    wp,
				HostingDomain:     "localhost:8080",
				MinHostingMinutes: 10,
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/upload", bytes.NewReader(bodyBytes))
			if tt.identityKey != nil {
				ctx := context.WithValue(req.Context(), middlewares.IdentityContextKey, tt.identityKey)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}

			if tt.expectedCode != "" {
				var resp handlers.UploadResponse
				json.Unmarshal(w.Body.Bytes(), &resp)
				if resp.Code != tt.expectedCode {
					t.Errorf("expected code %s, got %s", tt.expectedCode, resp.Code)
				}
			}
		})
	}
}
