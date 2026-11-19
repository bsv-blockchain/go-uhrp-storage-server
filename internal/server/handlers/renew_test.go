package handlers_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"testing"

	"github.com/bsv-blockchain/go-sdk/chainhash"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/transaction"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

type listOutputsFunc func(ctx context.Context, args sdkWallet.ListOutputsArgs, originator string) (*sdkWallet.ListOutputsResult, error)

// listOutputs returns a ListOutputs mock that always answers with outputs.
func listOutputs(outputs ...sdkWallet.Output) listOutputsFunc {
	return func(context.Context, sdkWallet.ListOutputsArgs, string) (*sdkWallet.ListOutputsResult, error) {
		return &sdkWallet.ListOutputsResult{Outputs: outputs}, nil
	}
}

func TestRenewHandlerServeHTTP(t *testing.T) {
	priv, err := ec.PrivateKeyFromHex("0000000000000000000000000000000000000000000000000000000000000001")
	require.NoError(t, err)

	pub := priv.PubKey()

	// Sign a SHA-256 digest. Importing the wallet package installs go-wallet-toolbox's
	// GoBDK signer, which rejects messages that are not already 32 bytes.
	digest := sha256.Sum256([]byte("test"))

	dummySig, err := priv.Sign(digest[:])
	require.NoError(t, err)

	zeroHash, err := chainhash.NewHash(make([]byte, 32))
	require.NoError(t, err)

	advertisement := func(tags ...string) sdkWallet.Output {
		return sdkWallet.Output{
			Outpoint: transaction.Outpoint{Txid: *zeroHash, Index: 0},
			Tags: append([]string{
				"uhrp_url_" + hex.EncodeToString([]byte("uhrp:success")),
				"uploader_identity_key_" + pub.ToDERHex(),
				"object_identifier_" + hex.EncodeToString([]byte("test-obj")),
				"expiry_time_2000000000",
			}, tags...),
		}
	}

	tests := []struct {
		name           string
		identityKey    *ec.PublicKey
		body           any
		mock           *mocks.MockWallet
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "unauthorized",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "ERR_UNAUTHORIZED",
		},
		{
			name:           "invalid request body",
			identityKey:    pub,
			body:           "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_MISSING_FIELDS",
		},
		{
			name:           "missing URL",
			identityKey:    pub,
			body:           handlers.RenewRequest{UhrpURL: "", AdditionalMinutes: 10},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_MISSING_FIELDS",
		},
		{
			name:           "missing minutes",
			identityKey:    pub,
			body:           handlers.RenewRequest{UhrpURL: "uhrp:success", AdditionalMinutes: 0},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_MISSING_FIELDS",
		},
		{
			name:           "not found",
			identityKey:    pub,
			body:           handlers.RenewRequest{UhrpURL: "uhrp:notfound", AdditionalMinutes: 60},
			mock:           &mocks.MockWallet{ListOutputsFunc: listOutputs()},
			expectedStatus: http.StatusNotFound,
			expectedCode:   "ERR_NOT_FOUND",
		},
		{
			name:           "ambiguous advertisement",
			identityKey:    pub,
			body:           handlers.RenewRequest{UhrpURL: "uhrp:success", AdditionalMinutes: 60},
			mock:           &mocks.MockWallet{ListOutputsFunc: listOutputs(advertisement("size_1024"), advertisement("size_1024"))},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "ERR_FIND",
		},
		{
			name:           "advertisement without a size",
			identityKey:    pub,
			body:           handlers.RenewRequest{UhrpURL: "uhrp:success", AdditionalMinutes: 60},
			mock:           &mocks.MockWallet{ListOutputsFunc: listOutputs(advertisement())},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "ERR_INTERNAL_RENEW",
		},
		{
			name:        "valid routing with backend failure",
			identityKey: pub,
			body:        handlers.RenewRequest{UhrpURL: "uhrp:success", AdditionalMinutes: 60},
			mock: &mocks.MockWallet{
				ListOutputsFunc: listOutputs(advertisement("size_1024")),
				GetPublicKeyFunc: func(context.Context, sdkWallet.GetPublicKeyArgs, string) (*sdkWallet.GetPublicKeyResult, error) {
					return &sdkWallet.GetPublicKeyResult{PublicKey: pub}, nil
				},
				CreateActionFunc: func(context.Context, sdkWallet.CreateActionArgs, string) (*sdkWallet.CreateActionResult, error) {
					return &sdkWallet.CreateActionResult{
						SignableTransaction: &sdkWallet.SignableTransaction{
							Tx:        make([]byte, 100), // not a valid BEEF
							Reference: []byte("mock-ref"),
						},
					}, nil
				},
				CreateSignatureFunc: func(context.Context, sdkWallet.CreateSignatureArgs, string) (*sdkWallet.CreateSignatureResult, error) {
					return &sdkWallet.CreateSignatureResult{Signature: dummySig}, nil
				},
			},
			// A full BEEF is too complex to build here: the renewal fails once it parses the
			// signable transaction, which proves the handler reached the renewal logic.
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "ERR_RENEW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.mock
			if mock == nil {
				mock = &mocks.MockWallet{}
			}

			h := &handlers.RenewHandler{
				Calculator:     newCalculator(),
				WalletProvider: newProvider(mock),
				Logger:         slog.Default(),
			}

			w := serve(h, newRequest(t, http.MethodPost, "/renew", tt.body, tt.identityKey))
			require.Equal(t, tt.expectedStatus, w.Code, "body: %s", w.Body.String())
			assert.Equal(t, tt.expectedCode, decode[handlers.RenewResponse](t, w).Code)
		})
	}
}
