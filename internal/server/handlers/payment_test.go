package handlers_test

import (
	"net/http"
	"testing"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

func TestRequestPriceCalculator(t *testing.T) {
	pub := testIdentityKey(t)
	advertisement := sdkWallet.Output{Tags: []string{uhrpURLTag("test"), "size_1024", "expiry_time_2000000000"}}

	tests := []struct {
		name         string
		method       string
		path         string
		body         any
		withIdentity bool
		outputs      []sdkWallet.Output
		wantPrice    int
		wantErr      error
	}{
		{
			// USD = (1024 / 1e9) * (60 / 43200) * 0.03, far below the 10-satoshi minimum.
			name:      "upload price",
			method:    http.MethodPost,
			path:      "/upload",
			body:      handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 60},
			wantPrice: 10,
		},
		{
			name:      "large upload price",
			method:    http.MethodPost,
			path:      "/upload",
			body:      handlers.UploadRequest{FileSize: 1_000_000_000, RetentionPeriod: 43_200},
			wantPrice: 100_000,
		},
		{
			name:      "unparsable upload body is free",
			method:    http.MethodPost,
			path:      "/upload",
			body:      "not json",
			wantPrice: 0,
		},
		{
			name:         "renew price",
			method:       http.MethodPost,
			path:         "/renew",
			body:         handlers.RenewRequest{UhrpURL: "test", AdditionalMinutes: 60},
			withIdentity: true,
			outputs:      []sdkWallet.Output{advertisement},
			wantPrice:    10,
		},
		{
			name:    "renew without identity",
			method:  http.MethodPost,
			path:    "/renew",
			body:    handlers.RenewRequest{UhrpURL: "test", AdditionalMinutes: 60},
			wantErr: handlers.ErrMissingIdentityKey,
		},
		{
			name:         "renew of an unknown file",
			method:       http.MethodPost,
			path:         "/renew",
			body:         handlers.RenewRequest{UhrpURL: "test", AdditionalMinutes: 60},
			withIdentity: true,
			wantErr:      walletpkg.ErrAdvertisementNotFound,
		},
		{
			name:      "other route",
			method:    http.MethodGet,
			path:      "/list",
			wantPrice: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wp := newProvider(&mocks.MockWallet{ListOutputsFunc: listOutputs(tt.outputs...)})
			priceFor := handlers.RequestPriceCalculator(newCalculator(), wp)

			req := newRequest(t, tt.method, tt.path, tt.body, nil)
			if tt.withIdentity {
				req = newRequest(t, tt.method, tt.path, tt.body, pub)
			}

			price, err := priceFor(req)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPrice, price)
		})
	}
}

func TestRequestPriceCalculatorRestoresBody(t *testing.T) {
	priceFor := handlers.RequestPriceCalculator(newCalculator(), newProvider(&mocks.MockWallet{}))

	req := newRequest(t, http.MethodPost, "/upload", `{"fileSize":1024,"retentionPeriod":60}`, nil)

	_, err := priceFor(req)
	require.NoError(t, err)

	var body handlers.UploadRequest
	require.NoError(t, decodeBody(req, &body))
	assert.Equal(t, handlers.UploadRequest{FileSize: 1024, RetentionPeriod: 60}, body)
}
