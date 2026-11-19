package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// mockOracle prices BSV at $30.
type mockOracle struct{}

func (mockOracle) GetExchangeRate(context.Context) (float64, error) {
	return 30.0, nil
}

func newCalculator() *pricing.Calculator {
	return pricing.NewCalculator(0.03, mockOracle{})
}

// testIdentityKey returns the identity key the tests authenticate as: the public key of
// the throwaway private key 3.
func testIdentityKey(t *testing.T) *ec.PublicKey {
	t.Helper()

	priv, err := ec.PrivateKeyFromHex("0000000000000000000000000000000000000000000000000000000000000003")
	require.NoError(t, err)

	return priv.PubKey()
}

// newProvider returns a wallet provider backed by the given mock wallet.
func newProvider(mw *mocks.MockWallet) *walletpkg.Provider {
	wp := walletpkg.NewProvider("", "", "", nil, slog.Default())
	wp.SetWallet(mw)

	return wp
}

// newRequest builds a request whose body is raw when body is a string, and JSON otherwise.
// A non-nil identityKey is placed in the context, as the auth middleware would.
func newRequest(t *testing.T, method, target string, body any, identityKey *ec.PublicKey) *http.Request {
	t.Helper()

	var reader io.Reader

	switch b := body.(type) {
	case nil:
	case string:
		reader = bytes.NewReader([]byte(b))
	case []byte:
		reader = bytes.NewReader(b)
	default:
		encoded, err := json.Marshal(b)
		require.NoError(t, err)

		reader = bytes.NewReader(encoded)
	}

	req := httptest.NewRequestWithContext(t.Context(), method, target, reader)
	if identityKey != nil {
		req = req.WithContext(context.WithValue(req.Context(), middlewares.IdentityContextKey, identityKey))
	}

	return req
}

// serve runs handler on req and returns the recorded response.
func serve(handler http.Handler, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	return w
}

// decodeBody decodes a request's (possibly restored) JSON body into v.
func decodeBody(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// decode unmarshals a JSON response body into a value of type T.
func decode[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()

	var v T
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &v), "body: %s", w.Body.String())

	return v
}
