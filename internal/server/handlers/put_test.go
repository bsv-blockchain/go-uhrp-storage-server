package handlers_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet/mocks"
)

const (
	testHMAC = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

	// testBody is the 5-byte upload the tests send.
	testBody = "hello"

	// validPutQuery is a complete presigned query for a testBody-sized file.
	validPutQuery = "uploader=test&objectID=test-obj&fileSize=5&expiry=2000000000&hmac=" + testHMAC
)

var errWalletUnavailable = errors.New("wallet unavailable")

type verifyHMACFunc func(ctx context.Context, args sdkWallet.VerifyHMACArgs, originator string) (*sdkWallet.VerifyHMACResult, error)

func hmacResult(valid bool) verifyHMACFunc {
	return func(context.Context, sdkWallet.VerifyHMACArgs, string) (*sdkWallet.VerifyHMACResult, error) {
		return &sdkWallet.VerifyHMACResult{Valid: valid}, nil
	}
}

func newStore(t *testing.T) *storage.FileStore {
	t.Helper()

	store, err := storage.NewFileStore(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	return store
}

// failingReader fails the test if the handler reads the request body.
type failingReader struct{ t *testing.T }

func (r failingReader) Read([]byte) (int, error) {
	r.t.Error("the request body must not be read")
	return 0, io.EOF
}

func TestPutHandlerServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		domain         string
		body           string
		verifyHMAC     verifyHMACFunc
		expectedStatus int
		expectedCode   string
		expectedDesc   string
		expectStored   bool
	}{
		{
			name:           "missing parameters",
			query:          "uploader=test",
			body:           testBody,
			expectedStatus: http.StatusBadRequest,
			expectedDesc:   "Missing required query parameters",
		},
		{
			name:           "non-numeric file size",
			query:          "uploader=test&objectID=123&fileSize=abc&expiry=2000000000&hmac=" + testHMAC,
			body:           testBody,
			expectedStatus: http.StatusBadRequest,
			expectedDesc:   "Invalid fileSize",
		},
		{
			name:           "file size above the maximum",
			query:          "uploader=test&objectID=123&fileSize=11000000001&expiry=2000000000&hmac=" + testHMAC,
			body:           testBody,
			expectedStatus: http.StatusBadRequest,
			expectedDesc:   "Invalid fileSize",
		},
		{
			name:           "invalid expiry",
			query:          "uploader=test&objectID=123&fileSize=5&expiry=tomorrow&hmac=" + testHMAC,
			body:           testBody,
			expectedStatus: http.StatusBadRequest,
			expectedDesc:   "Invalid expiry",
		},
		{
			name:           "invalid HMAC",
			domain:         "example.com",
			query:          "uploader=test&objectID=123&fileSize=5&expiry=2000000000&hmac=" + testHMAC,
			body:           testBody,
			verifyHMAC:     hmacResult(false),
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "ERR_UNAUTHORIZED",
		},
		{
			name:           "malformed HMAC",
			domain:         "example.com",
			query:          "uploader=test&objectID=123&fileSize=5&expiry=2000000000&hmac=zz",
			body:           testBody,
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   "ERR_UNAUTHORIZED",
		},
		{
			name:   "HMAC verification error",
			domain: "example.com",
			query:  validPutQuery,
			body:   testBody,
			verifyHMAC: func(context.Context, sdkWallet.VerifyHMACArgs, string) (*sdkWallet.VerifyHMACResult, error) {
				return nil, errWalletUnavailable
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "ERR_HMAC_VERIFY",
		},
		{
			name:           "body shorter than file size",
			query:          "uploader=test&objectID=123&fileSize=10&expiry=2000000000&hmac=" + testHMAC,
			body:           testBody,
			verifyHMAC:     hmacResult(true),
			expectedStatus: http.StatusBadRequest,
			expectedDesc:   "Size mismatch",
		},
		{
			name:           "body longer than file size",
			query:          "uploader=test&objectID=123&fileSize=2&expiry=2000000000&hmac=" + testHMAC,
			body:           testBody,
			verifyHMAC:     hmacResult(true),
			expectedStatus: http.StatusBadRequest,
			expectedDesc:   "Size mismatch",
		},
		{
			name:           "stored but not advertised on localhost",
			domain:         "localhost:8080",
			query:          validPutQuery,
			body:           testBody,
			verifyHMAC:     hmacResult(true),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "ERR_INTERNAL",
			expectStored:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStore(t)
			h := &handlers.PutHandler{
				Store:          store,
				WalletProvider: newProvider(&mocks.MockWallet{VerifyHMACFunc: tt.verifyHMAC}),
				HostingDomain:  tt.domain,
				Logger:         slog.Default(),
			}

			w := serve(h, newRequest(t, http.MethodPut, "/put?"+tt.query, tt.body, nil))
			require.Equal(t, tt.expectedStatus, w.Code, "body: %s", w.Body.String())

			resp := decode[responses.ErrorResponse](t, w)
			if tt.expectedCode != "" {
				assert.Equal(t, tt.expectedCode, resp.Code)
			}

			if tt.expectedDesc != "" {
				assert.Equal(t, tt.expectedDesc, resp.Description)
			}

			assert.Equal(t, tt.expectStored, store.Exists("test-obj"))
		})
	}
}

func TestPutHandlerVerifiesHMACBeforeReadingBody(t *testing.T) {
	h := &handlers.PutHandler{
		Store:          newStore(t),
		WalletProvider: newProvider(&mocks.MockWallet{VerifyHMACFunc: hmacResult(false)}),
		HostingDomain:  "example.com",
		Logger:         slog.Default(),
	}

	req := newRequest(t, http.MethodPut, "/put?"+validPutQuery, nil, nil)
	req.Body = io.NopCloser(failingReader{t: t})

	w := serve(h, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestPutHandlerReadsAtMostTheAuthorizedSize(t *testing.T) {
	h := &handlers.PutHandler{
		Store:          newStore(t),
		WalletProvider: newProvider(&mocks.MockWallet{VerifyHMACFunc: hmacResult(true)}),
		HostingDomain:  "example.com",
		Logger:         slog.Default(),
	}

	// A 5-byte upload followed by an endless stream: the handler must stop after one extra byte.
	body := &countingReader{r: io.MultiReader(strings.NewReader(testBody), endless{})}

	req := newRequest(t, http.MethodPut, "/put?"+validPutQuery, nil, nil)
	req.Body = io.NopCloser(body)

	w := serve(h, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Size mismatch", decode[responses.ErrorResponse](t, w).Description)
	assert.LessOrEqual(t, body.n, int64(6))
}

func TestPutHandlerRejectsExistingFile(t *testing.T) {
	store := newStore(t)
	require.NoError(t, store.Write("test-obj", []byte("first")))

	h := &handlers.PutHandler{
		Store:          store,
		WalletProvider: newProvider(&mocks.MockWallet{VerifyHMACFunc: hmacResult(true)}),
		HostingDomain:  "localhost:8080",
		Logger:         slog.Default(),
	}

	w := serve(h, newRequest(t, http.MethodPut, "/put?"+validPutQuery, testBody, nil))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "File exists", decode[responses.ErrorResponse](t, w).Description)

	data, err := store.Read("test-obj")
	require.NoError(t, err)
	assert.Equal(t, "first", string(data), "an existing file must not be overwritten")
}

// endless is a reader that never ends.
type endless struct{}

func (endless) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}

	return len(p), nil
}

// countingReader counts the bytes read through it.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)

	return n, err
}
