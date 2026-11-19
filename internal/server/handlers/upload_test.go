package handlers_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

type createHMACFunc func(ctx context.Context, args sdkWallet.CreateHMACArgs, originator string) (*sdkWallet.CreateHMACResult, error)

func fixedHMAC(context.Context, sdkWallet.CreateHMACArgs, string) (*sdkWallet.CreateHMACResult, error) {
	return &sdkWallet.CreateHMACResult{HMAC: [32]byte{1, 2, 3}}, nil
}

func TestUploadHandlerServeHTTP(t *testing.T) {
	pub := testIdentityKey(t)

	tests := []struct {
		name           string
		identityKey    *ec.PublicKey
		body           any
		createHMAC     createHMACFunc
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "unauthorized",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "ERR_UNAUTHORIZED",
		},
		{
			name:           "invalid body",
			identityKey:    pub,
			body:           "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_BODY",
		},
		{
			name:           "invalid file size",
			identityKey:    pub,
			body:           handlers.UploadRequest{FileSize: 0, RetentionPeriod: 60},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_SIZE",
		},
		{
			name:           "file too large",
			identityKey:    pub,
			body:           handlers.UploadRequest{FileSize: 11_000_000_001, RetentionPeriod: 60},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_SIZE",
		},
		{
			name:           "missing retention period",
			identityKey:    pub,
			body:           handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 0},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_NO_RETENTION_PERIOD",
		},
		{
			name:           "retention period too small",
			identityKey:    pub,
			body:           handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 5},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_RETENTION_PERIOD",
		},
		{
			name:        "HMAC failure",
			identityKey: pub,
			body:        handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 60},
			createHMAC: func(context.Context, sdkWallet.CreateHMACArgs, string) (*sdkWallet.CreateHMACResult, error) {
				return nil, errWalletUnavailable
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "ERR_HMAC",
		},
		{
			name:           "success",
			identityKey:    pub,
			body:           handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 60},
			createHMAC:     fixedHMAC,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handlers.UploadHandler{
				Calculator:        newCalculator(),
				WalletProvider:    newProvider(&mocks.MockWallet{CreateHMACFunc: tt.createHMAC}),
				HostingDomain:     "localhost:8080",
				MinHostingMinutes: 10,
				Logger:            slog.Default(),
			}

			w := serve(h, newRequest(t, http.MethodPost, "/upload", tt.body, tt.identityKey))
			require.Equal(t, tt.expectedStatus, w.Code, "body: %s", w.Body.String())
			assert.Equal(t, tt.expectedCode, decode[handlers.UploadResponse](t, w).Code)
		})
	}
}

func TestUploadHandlerPresignedURL(t *testing.T) {
	pub := testIdentityKey(t)

	tests := []struct {
		name   string
		domain string
		scheme string
	}{
		{name: "localhost uses http", domain: "localhost:8080", scheme: "http"},
		{name: "public host uses https", domain: "files.example.com", scheme: "https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handlers.UploadHandler{
				Calculator:     newCalculator(),
				WalletProvider: newProvider(&mocks.MockWallet{CreateHMACFunc: fixedHMAC}),
				HostingDomain:  tt.domain,
				Logger:         slog.Default(),
			}

			w := serve(h, newRequest(t, http.MethodPost, "/upload", handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 60}, pub))
			require.Equal(t, http.StatusOK, w.Code)

			resp := decode[handlers.UploadResponse](t, w)
			assert.Equal(t, "success", resp.Status)
			assert.Equal(t, int64(10), resp.Amount)

			uploadURL, err := url.Parse(resp.UploadURL)
			require.NoError(t, err)
			assert.Equal(t, tt.scheme, uploadURL.Scheme)
			assert.Equal(t, tt.domain, uploadURL.Host)
			assert.Equal(t, "/put", uploadURL.Path)

			q := uploadURL.Query()
			assert.Equal(t, "1024", q.Get("fileSize"))
			assert.Equal(t, pub.ToDERHex(), q.Get("uploader"))
			assert.NotEmpty(t, q.Get("objectID"))
			assert.NotEmpty(t, q.Get("expiry"))
			assert.True(t, strings.HasPrefix(q.Get("hmac"), "010203"))
		})
	}
}
