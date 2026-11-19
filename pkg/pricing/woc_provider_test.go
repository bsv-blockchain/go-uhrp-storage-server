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

// rateServer is a local stand-in for the WhatsOnChain exchange-rate API.
type rateServer struct {
	status atomic.Int32
	body   atomic.Value
	hits   atomic.Int32
}

// answer sets the status and body the server responds with from now on.
func (s *rateServer) answer(status int, body string) {
	s.status.Store(int32(status)) //nolint:gosec // test HTTP status codes fit in an int32
	s.body.Store(body)
}

// newTestProvider points a provider at a local server that answers with status and body.
func newTestProvider(t *testing.T, status int, body string) (*WhatsOnChainProvider, *rateServer) {
	t.Helper()

	rs := &rateServer{}
	rs.answer(status, body)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rs.hits.Add(1)
		w.WriteHeader(int(rs.status.Load()))
		_, _ = w.Write([]byte(rs.body.Load().(string)))
	}))
	t.Cleanup(srv.Close)

	return newWhatsOnChainProvider(srv.URL, srv.Client()), rs
}

// ageCache makes the cached rate look as if it had been fetched age ago.
func ageCache(provider *WhatsOnChainProvider, age time.Duration) {
	provider.mu.Lock()
	provider.cachedAt = time.Now().Add(-age)
	provider.mu.Unlock()
}

func TestWhatsOnChainProviderGetExchangeRate(t *testing.T) {
	tests := []struct {
		name    string
		status  int
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
			provider, _ := newTestProvider(t, tt.status, tt.body)

			rate, err := provider.GetExchangeRate(t.Context())
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
	provider, _ := newTestProvider(t, http.StatusOK, "not json")

	_, err := provider.GetExchangeRate(t.Context())
	require.Error(t, err)
}

func TestWhatsOnChainProviderServesFreshRateFromCache(t *testing.T) {
	provider, rs := newTestProvider(t, http.StatusOK, `{"rate":20}`)

	rate, err := provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	require.InDelta(t, 20.0, rate, 1e-9)

	// Within the refresh interval the API is not asked again, so its new rate is not seen yet.
	rs.answer(http.StatusOK, `{"rate":25}`)

	for range 5 {
		rate, err = provider.GetExchangeRate(t.Context())
		require.NoError(t, err)
		assert.InDelta(t, 20.0, rate, 1e-9)
	}

	assert.Equal(t, int32(1), rs.hits.Load())
}

func TestWhatsOnChainProviderRefreshesAfterInterval(t *testing.T) {
	provider, rs := newTestProvider(t, http.StatusOK, `{"rate":20}`)

	_, err := provider.GetExchangeRate(t.Context())
	require.NoError(t, err)

	rs.answer(http.StatusOK, `{"rate":25}`)
	ageCache(provider, provider.refreshInterval+time.Second)

	rate, err := provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	assert.InDelta(t, 25.0, rate, 1e-9)
	assert.Equal(t, int32(2), rs.hits.Load())
}

func TestWhatsOnChainProviderFallsBackToRecentRate(t *testing.T) {
	provider, rs := newTestProvider(t, http.StatusOK, `{"rate":20}`)

	rate, err := provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	require.InDelta(t, 20.0, rate, 1e-9)

	// The API starts failing once the rate is due for a refresh: the cached rate is served while
	// it is younger than the cache TTL.
	rs.answer(http.StatusServiceUnavailable, "")
	ageCache(provider, provider.refreshInterval+time.Second)

	rate, err = provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	assert.InDelta(t, 20.0, rate, 1e-9)
	assert.Equal(t, int32(2), rs.hits.Load(), "a stale rate is only served after the API was asked")

	// Once the cached rate is older than the TTL, the failure is returned.
	ageCache(provider, provider.cacheTTL+time.Second)

	_, err = provider.GetExchangeRate(t.Context())
	require.ErrorIs(t, err, ErrUnexpectedStatus)
}

func TestNewWhatsOnChainProviderDefaults(t *testing.T) {
	provider := NewWhatsOnChainProvider()

	assert.Equal(t, whatsOnChainExchangeRateURL, provider.url)
	assert.Equal(t, whatsOnChainTimeout, provider.httpClient.Timeout)
	assert.Equal(t, whatsOnChainRefreshInterval, provider.refreshInterval)
	assert.Equal(t, whatsOnChainCacheTTL, provider.cacheTTL)
	assert.Less(t, provider.refreshInterval, provider.cacheTTL, "a stale rate must outlive the refresh interval to be a fallback")
}

func TestNewWhatsOnChainProviderWithURL(t *testing.T) {
	custom := NewWhatsOnChainProviderWithURL("http://127.0.0.1:9999/rate")
	assert.Equal(t, "http://127.0.0.1:9999/rate", custom.url)
	assert.Equal(t, whatsOnChainTimeout, custom.httpClient.Timeout)

	assert.Equal(t, whatsOnChainExchangeRateURL, NewWhatsOnChainProviderWithURL("").url)
}
