package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

func TestPutHandler_ServeHTTP(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "put_test")
	defer os.RemoveAll(tempDir)
	store := storage.NewFileStore(tempDir)

	tests := []struct {
		name               string
		query              string
		domain             string
		body               []byte
		mockVerifyHMACFunc func(ctx context.Context, args sdkWallet.VerifyHMACArgs, originator string) (*sdkWallet.VerifyHMACResult, error)
		mockPubKeyFunc     func(ctx context.Context, args sdkWallet.GetPublicKeyArgs, originator string) (*sdkWallet.GetPublicKeyResult, error)
		mockActionFunc     func(ctx context.Context, args sdkWallet.CreateActionArgs, originator string) (*sdkWallet.CreateActionResult, error)
		expectedStatus     int
	}{
		{
			name:           "Missing Parameters",
			query:          "uploader=test",
			body:           []byte("hello"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Size Mismatch",
			query:          "uploader=test&objectID=123&fileSize=10&expiry=2000000000&hmac=0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
			body:           []byte("hello"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Unauthorized - Invalid HMAC",
			domain: "https://example.com",
			query:  "uploader=test&objectID=123&fileSize=5&expiry=2000000000&hmac=0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
			body:   []byte("hello"),
			mockVerifyHMACFunc: func(ctx context.Context, args sdkWallet.VerifyHMACArgs, originator string) (*sdkWallet.VerifyHMACResult, error) {
				return &sdkWallet.VerifyHMACResult{Valid: false}, nil
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "Failure - Localhost",
			domain: "localhost:8080",
			query:  "uploader=test&objectID=test-obj&fileSize=5&expiry=2000000000&hmac=0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
			body:   []byte("hello"),
			mockVerifyHMACFunc: func(ctx context.Context, args sdkWallet.VerifyHMACArgs, originator string) (*sdkWallet.VerifyHMACResult, error) {
				return &sdkWallet.VerifyHMACResult{Valid: true}, nil
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := &mocks.MockWallet{
				VerifyHMACFunc:   tt.mockVerifyHMACFunc,
				GetPublicKeyFunc: tt.mockPubKeyFunc,
				CreateActionFunc: tt.mockActionFunc,
			}
			wp := walletpkg.NewProvider("", "", "")
			wp.SetWallet(mw)

			h := &handlers.PutHandler{
				Store:          store,
				WalletProvider: wp,
				HostingDomain:  tt.domain,
			}

			req := httptest.NewRequest("PUT", "/put?"+tt.query, bytes.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}
