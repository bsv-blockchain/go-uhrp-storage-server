package handlers_test

import (
	"context"
	"encoding/hex"
	"log/slog"
	"net/http"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

func uhrpURLTag(url string) string {
	return "uhrp_url_" + hex.EncodeToString([]byte(url))
}

func TestListHandlerServeHTTP(t *testing.T) {
	pub := testIdentityKey(t)

	tests := []struct {
		name           string
		identityKey    *ec.PublicKey
		listOutputs    listOutputsFunc
		expectedStatus int
		expectedURLs   []string
	}{
		{
			name:           "unauthorized",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "empty list",
			identityKey:    pub,
			listOutputs:    listOutputs(),
			expectedStatus: http.StatusOK,
			expectedURLs:   []string{},
		},
		{
			name:        "multiple items in wallet order",
			identityKey: pub,
			listOutputs: listOutputs(
				sdkWallet.Output{Tags: []string{uhrpURLTag("uhrp://second"), "expiry_time_2000000000"}},
				sdkWallet.Output{Tags: []string{uhrpURLTag("uhrp://first"), "expiry_time_2000000000"}},
			),
			expectedStatus: http.StatusOK,
			expectedURLs:   []string{"uhrp://second", "uhrp://first"},
		},
		{
			name:        "wallet failure",
			identityKey: pub,
			listOutputs: func(context.Context, sdkWallet.ListOutputsArgs, string) (*sdkWallet.ListOutputsResult, error) {
				return nil, errWalletUnavailable
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handlers.ListHandler{
				WalletProvider: newProvider(&mocks.MockWallet{ListOutputsFunc: tt.listOutputs}),
				Logger:         slog.Default(),
			}

			w := serve(h, newRequest(t, http.MethodGet, "/list", nil, tt.identityKey))
			require.Equal(t, tt.expectedStatus, w.Code, "body: %s", w.Body.String())

			if tt.expectedStatus != http.StatusOK {
				return
			}

			resp := decode[handlers.ListResponse](t, w)
			assert.Equal(t, "success", resp.Status)

			urls := make([]string, 0, len(resp.Uploads))
			for _, u := range resp.Uploads {
				urls = append(urls, u.UhrpURL)
			}

			assert.Equal(t, tt.expectedURLs, urls)
		})
	}
}

func TestListHandlerPassesPagination(t *testing.T) {
	var gotLimit, gotOffset uint32

	mock := &mocks.MockWallet{
		ListOutputsFunc: func(_ context.Context, args sdkWallet.ListOutputsArgs, _ string) (*sdkWallet.ListOutputsResult, error) {
			gotLimit, gotOffset = *args.Limit, *args.Offset
			return &sdkWallet.ListOutputsResult{}, nil
		},
	}

	h := &handlers.ListHandler{WalletProvider: newProvider(mock), Logger: slog.Default()}

	w := serve(h, newRequest(t, http.MethodGet, "/list", `{"limit":5,"offset":10}`, testIdentityKey(t)))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, uint32(5), gotLimit)
	assert.Equal(t, uint32(10), gotOffset)

	// A malformed body falls back to the wallet defaults.
	w = serve(h, newRequest(t, http.MethodGet, "/list", `not json`, testIdentityKey(t)))
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, uint32(200), gotLimit)
	assert.Equal(t, uint32(0), gotOffset)
}
