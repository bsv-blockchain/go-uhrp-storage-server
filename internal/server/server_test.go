package server_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/config"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

type staticRate float64

func (r staticRate) GetExchangeRate(context.Context) (float64, error) { return float64(r), nil }

// newHandler builds the server's router with no wallet, a temp public dir and one uploaded file.
func newHandler(t *testing.T) http.Handler {
	t.Helper()

	publicDir := t.TempDir()

	store, err := storage.NewFileStore(publicDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	require.NoError(t, store.Write("hello.txt", []byte("hello")))

	cfg := &config.Config{HTTPPort: "0", HostingDomain: "localhost:8080"}
	calc := pricing.NewCalculator(0.03, staticRate(30))
	wp := wallet.NewProvider("", "", "mainnet", nil, slog.Default())

	srv := server.New(cfg, calc, store, wp, publicDir, slog.Default())
	assert.Equal(t, ":0", srv.HTTPServer.Addr)

	return srv.HTTPServer.Handler
}

func serve(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body)))

	return w
}

func requireNotFoundJSON(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp responses.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, responses.ErrorResponse{Status: "error", Code: "ERR_ROUTE_NOT_FOUND", Description: "Route not found."}, resp)
}

func TestServerUnknownRouteReturnsJSON404(t *testing.T) {
	requireNotFoundJSON(t, serve(t, newHandler(t), http.MethodGet, "/nonexistent", ""))
}

func TestServerServesUploadedFiles(t *testing.T) {
	w := serve(t, newHandler(t), http.MethodGet, "/cdn/hello.txt", "")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
	assert.Equal(t, "hello", w.Body.String())
}

func TestServerNeverListsTheCDNDirectory(t *testing.T) {
	h := newHandler(t)

	for _, path := range []string{"/cdn/", "/cdn/missing", "/cdn/../go.mod"} {
		t.Run(path, func(t *testing.T) {
			w := serve(t, h, http.MethodGet, path, "")

			assert.NotContains(t, w.Body.String(), "hello.txt", "directory contents must not leak")
			assert.NotEqual(t, http.StatusOK, w.Code)
		})
	}

	requireNotFoundJSON(t, serve(t, h, http.MethodGet, "/cdn/", ""))
}

func TestServerQuoteRoute(t *testing.T) {
	w := serve(t, newHandler(t), http.MethodPost, "/quote", `{"fileSize":1024,"retentionPeriod":60}`)

	require.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"quote":10}`, w.Body.String())
}

func TestServerCORSPreflight(t *testing.T) {
	w := serve(t, newHandler(t), http.MethodOptions, "/upload", "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Private-Network"))
}

func TestServerWithoutWalletDoesNotMountWalletRoutes(t *testing.T) {
	h := newHandler(t)

	for _, route := range []struct{ method, path string }{
		{http.MethodPost, "/upload"},
		{http.MethodGet, "/list"},
		{http.MethodPost, "/renew"},
		{http.MethodGet, "/find"},
		{http.MethodPost, "/.well-known/auth"},
	} {
		t.Run(route.path, func(t *testing.T) {
			requireNotFoundJSON(t, serve(t, h, route.method, route.path, "{}"))
		})
	}
}
