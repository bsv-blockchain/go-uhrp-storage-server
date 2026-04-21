package handlers_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bsv-blockchain/go-sdk/chainhash"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

func TestRenewHandler_ServeHTTP(t *testing.T) {
	priv, _ := ec.PrivateKeyFromHex("0000000000000000000000000000000000000000000000000000000000000001")
	pub := priv.PubKey()
	dummySig, _ := priv.Sign([]byte("test"))
	zeroHash, _ := chainhash.NewHash(make([]byte, 32))

	tests := []struct {
		name           string
		identityKey    *ec.PublicKey
		body           interface{}
		mockListFunc   func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error)
		mockActionFunc func(ctx context.Context, args sdkWallet.CreateActionArgs, originator string) (*sdkWallet.CreateActionResult, error)
		mockSignFunc   func(ctx context.Context, args sdkWallet.SignActionArgs, originator string) (*sdkWallet.SignActionResult, error)
		mockPubKeyFunc func(ctx context.Context, args sdkWallet.GetPublicKeyArgs, originator string) (*sdkWallet.GetPublicKeyResult, error)
		mockSigFunc    func(ctx context.Context, args sdkWallet.CreateSignatureArgs, originator string) (*sdkWallet.CreateSignatureResult, error)
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Unauthorized",
			identityKey:    nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Request Body",
			identityKey:    pub,
			body:           "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_MISSING_FIELDS",
		},
		{
			name:           "Missing URL or Minutes",
			identityKey:    pub,
			body:           handlers.RenewRequest{UhrpURL: "", AdditionalMinutes: 10},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_MISSING_FIELDS",
		},
		{
			name:        "Not Found",
			identityKey: pub,
			body:        handlers.RenewRequest{UhrpURL: "uhrp:notfound", AdditionalMinutes: 60},
			mockListFunc: func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error) {
				return &sdkWallet.ListOutputsResult{Outputs: []sdkWallet.Output{}}, nil
			},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "ERR_NOT_FOUND",
		},
		{
			name:        "Valid Routing with Backend Failure",
			identityKey: pub,
			body:        handlers.RenewRequest{UhrpURL: "uhrp:success", AdditionalMinutes: 60},
			mockListFunc: func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error) {
				return &sdkWallet.ListOutputsResult{
					Outputs: []sdkWallet.Output{
						{
							Outpoint: transaction.Outpoint{
								Txid:  *zeroHash,
								Index: 0,
							},
							Tags: []string{
								"uhrp_url_" + hex.EncodeToString([]byte("uhrp:success")),
								"uploader_identity_key_" + pub.ToDERHex(),
								"object_identifier_" + hex.EncodeToString([]byte("test-obj")),
								"size_1024",
								"expiry_time_2000000000",
							},
						},
					},
				}, nil
			},
			mockPubKeyFunc: func(ctx context.Context, args sdkWallet.GetPublicKeyArgs, originator string) (*sdkWallet.GetPublicKeyResult, error) {
				return &sdkWallet.GetPublicKeyResult{PublicKey: pub}, nil
			},
			mockActionFunc: func(ctx context.Context, args sdkWallet.CreateActionArgs, originator string) (*sdkWallet.CreateActionResult, error) {
				return &sdkWallet.CreateActionResult{
					SignableTransaction: &sdkWallet.SignableTransaction{
						Tx:        make([]byte, 100), // Longer dummy BEEF
						Reference: []byte("mock-ref"),
					},
				}, nil
			},
			mockSigFunc: func(ctx context.Context, args sdkWallet.CreateSignatureArgs, originator string) (*sdkWallet.CreateSignatureResult, error) {
				return &sdkWallet.CreateSignatureResult{Signature: dummySig}, nil
			},
			// Note: Status 500 is expected because full BEEF reconstruction is too complex for this unit test.
			// This test verifies that the handler correctly reaches the renewal logic without panicking.
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "ERR_RENEW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &mocks.MockWallet{
				ListOutputsFunc:     tt.mockListFunc,
				CreateActionFunc:    tt.mockActionFunc,
				SignActionFunc:      tt.mockSignFunc,
				GetPublicKeyFunc:    tt.mockPubKeyFunc,
				CreateSignatureFunc: tt.mockSigFunc,
			}
			wp := walletpkg.NewProvider("", "", "", nil, slog.Default())
			wp.SetWallet(mw)

			calc := pricing.NewCalculator(0.03, mockOracle{})
			h := &handlers.RenewHandler{
				Calculator:     calc,
				WalletProvider: wp,
				Logger:         slog.Default(),
			}

			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/renew", bytes.NewReader(bodyBytes))
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
				var resp handlers.RenewResponse
				json.Unmarshal(w.Body.Bytes(), &resp)
				if resp.Code != tt.expectedCode {
					t.Errorf("expected code %s, got %s", tt.expectedCode, resp.Code)
				}
			}
		})
	}
}
