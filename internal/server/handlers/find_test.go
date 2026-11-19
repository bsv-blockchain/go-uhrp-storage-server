package handlers_test

import (
	"log/slog"
	"net/http"
	"net/url"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

func TestFindHandlerServeHTTP(t *testing.T) {
	pub := testIdentityKey(t)

	tests := []struct {
		name           string
		uhrpURL        string
		identityKey    *ec.PublicKey
		listOutputs    listOutputsFunc
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "unauthorized - no identity key",
			uhrpURL:        "uhrp:test",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "ERR_UNAUTHORIZED",
		},
		{
			name:           "bad request - no uhrpUrl",
			identityKey:    pub,
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_NO_UHRP_URL",
		},
		{
			name:           "not found - no advertisement",
			uhrpURL:        "uhrp:notfound",
			identityKey:    pub,
			listOutputs:    listOutputs(),
			expectedStatus: http.StatusNotFound,
			expectedCode:   "ERR_NOT_FOUND",
		},
		{
			name:        "success - valid find",
			uhrpURL:     "uhrp:success",
			identityKey: pub,
			listOutputs: listOutputs(sdkWallet.Output{Tags: []string{
				uhrpURLTag("uhrp:success"),
				"object_identifier_test-obj",
				"size_1024",
				"content_type_image/png",
				"expiry_time_2000000000",
			}}),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handlers.FindHandler{
				WalletProvider: newProvider(&mocks.MockWallet{ListOutputsFunc: tt.listOutputs}),
				Logger:         slog.Default(),
			}

			target := "/find"
			if tt.uhrpURL != "" {
				target += "?uhrpUrl=" + url.QueryEscape(tt.uhrpURL)
			}

			w := serve(h, newRequest(t, http.MethodGet, target, nil, tt.identityKey))
			require.Equal(t, tt.expectedStatus, w.Code, "body: %s", w.Body.String())

			resp := decode[handlers.FindResponse](t, w)
			assert.Equal(t, tt.expectedCode, resp.Code)

			if tt.expectedStatus == http.StatusOK {
				require.NotNil(t, resp.Data)
				assert.Equal(t, "success", resp.Status)
			}
		})
	}
}

func TestFindHandlerResponseData(t *testing.T) {
	h := &handlers.FindHandler{
		WalletProvider: newProvider(&mocks.MockWallet{ListOutputsFunc: listOutputs(sdkWallet.Output{Tags: []string{
			uhrpURLTag("uhrp:success"),
			"object_identifier_test-obj", // not hex: kept as-is
			"size_1024",
			"content_type_image/png",
			"expiry_time_2000000000",
		}})}),
		Logger: slog.Default(),
	}

	w := serve(h, newRequest(t, http.MethodGet, "/find?uhrpUrl=uhrp:success", nil, testIdentityKey(t)))
	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t,
		`{"status":"success","data":{"name":"test-obj","size":"1024","mimeType":"image/png","expiryTime":2000000000}}`,
		w.Body.String())
}
