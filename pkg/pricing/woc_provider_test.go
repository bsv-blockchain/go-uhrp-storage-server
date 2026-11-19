package pricing

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProvider points a provider at a local server that answers with status and body.
func newTestProvider(t *testing.T, status *atomic.Int32, body *atomic.Value) *WhatsOnChainProvider {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(int(status.Load()))
		_, _ = w.Write([]byte(body.Load().(string)))
	}))
	t.Cleanup(srv.Close)

	return newWhatsOnChainProvider(srv.URL, srv.Client())
}

func TestWhatsOnChainProviderGetExchangeRate(t *testing.T) {
	tests := []struct {
		name    string
		status  int32
		body    string
		want    float64
		wantErr error
	}{
		{name: "valid rate", status: http.StatusOK, body: `{"currency":"USD","rate":15.17,"time":1}`, want: 15.17},
		{name: "server error", status: http.StatusInternalServerError, body: "boom", wantErr: ErrUnexpectedStatus},
		{name: "rate limited", status: http.StatusTooManyRequests, body: "", wantErr: ErrUnexpectedStatus},
		{name: "zero rate", status: http.StatusOK, body: `{"rate":0}`, wantErr: ErrInvalidExchangeRate},
		{name: "negative rate", status: http.StatusOK, body: `{"rate":-2}`, wantErr: ErrInvalidExchangeRate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status atomic.Int32

			var body atomic.Value

			status.Store(tt.status)
			body.Store(tt.body)

			rate, err := newTestProvider(t, &status, &body).GetExchangeRate(t.Context())
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)
			assert.InDelta(t, tt.want, rate, 1e-9)
		})
	}
}

func TestWhatsOnChainProviderMalformedBody(t *testing.T) {
	var status atomic.Int32

	var body atomic.Value

	status.Store(http.StatusOK)
	body.Store("not json")

	_, err := newTestProvider(t, &status, &body).GetExchangeRate(t.Context())
	require.Error(t, err)
}

func TestWhatsOnChainProviderFallsBackToRecentRate(t *testing.T) {
	var status atomic.Int32

	var body atomic.Value

	status.Store(http.StatusOK)
	body.Store(`{"rate":20}`)

	provider := newTestProvider(t, &status, &body)

	rate, err := provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	require.InDelta(t, 20.0, rate, 1e-9)

	// The API starts failing: the cached rate is served while it is fresh.
	status.Store(http.StatusServiceUnavailable)

	rate, err = provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	assert.InDelta(t, 20.0, rate, 1e-9)

	// Once the cached rate is older than the TTL, the failure is returned.
	provider.mu.Lock()
	provider.cachedAt = time.Now().Add(-2 * provider.cacheTTL)
	provider.mu.Unlock()

	_, err = provider.GetExchangeRate(t.Context())
	require.ErrorIs(t, err, ErrUnexpectedStatus)
}

func TestNewWhatsOnChainProviderDefaults(t *testing.T) {
	provider := NewWhatsOnChainProvider()

	assert.Equal(t, whatsOnChainExchangeRateURL, provider.url)
	assert.Equal(t, whatsOnChainTimeout, provider.httpClient.Timeout)
	assert.Equal(t, whatsOnChainCacheTTL, provider.cacheTTL)
}
