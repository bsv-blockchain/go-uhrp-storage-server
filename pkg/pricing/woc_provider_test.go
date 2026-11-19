package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/mrz1836/go-whatsonchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// exchangeRatePath is where go-whatsonchain asks for the mainnet BSV/USD rate, under its base URL.
	exchangeRatePath = "/bsv/main/exchangerate"

	testAPIKey = "woc-test-key" //nolint:gosec // G101: a made-up key, only ever sent to a local test server
)

var errFakeRateClient = errors.New("fake WhatsOnChain failure")

// rateResponse is one answer from rateServer.
type rateResponse struct {
	status     int
	body       string
	retryAfter string // the Retry-After header, when not empty
}

// rateServer is a local stand-in for the WhatsOnChain API. It answers with its responses in
// order, repeating the last one, and records each request.
type rateServer struct {
	mu        sync.Mutex
	responses []rateResponse
	hits      int
	paths     []string
	apiKeys   []string
	onRequest func()
}

// newRateServer starts a rateServer and returns it with its base URL, the value for WithBaseURL.
func newRateServer(t *testing.T, responses ...rateResponse) (*rateServer, string) {
	t.Helper()
	require.NotEmpty(t, responses)

	rs := &rateServer{responses: responses}
	srv := httptest.NewServer(http.HandlerFunc(rs.serveHTTP))
	t.Cleanup(srv.Close)

	return rs, srv.URL + "/"
}

func (s *rateServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	resp := s.responses[min(s.hits, len(s.responses)-1)]
	s.hits++
	s.paths = append(s.paths, r.URL.Path)
	s.apiKeys = append(s.apiKeys, r.Header.Get("woc-api-key"))
	onRequest := s.onRequest
	s.mu.Unlock()

	if onRequest != nil {
		onRequest()
	}

	if resp.retryAfter != "" {
		w.Header().Set("Retry-After", resp.retryAfter)
	}

	w.WriteHeader(resp.status)
	_, _ = w.Write([]byte(resp.body))
}

// requests returns how many requests the server has answered, their paths and their API key headers.
func (s *rateServer) requests() (int, []string, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.hits, append([]string(nil), s.paths...), append([]string(nil), s.apiKeys...)
}

// newHTTPTestProvider creates a provider through the real go-whatsonchain client, pointed at
// baseURL and backing off for only a millisecond between attempts. No API key comes from the
// environment unless the test sets one.
func newHTTPTestProvider(t *testing.T, baseURL string, opts ...whatsonchain.ClientOption) *WhatsOnChainProvider {
	t.Helper()

	opts = append([]whatsonchain.ClientOption{
		whatsonchain.WithBaseURL(baseURL),
		whatsonchain.WithBackoff(time.Millisecond, time.Millisecond, 1, 0),
	}, opts...)

	provider, err := NewWhatsOnChainProvider(t.Context(), opts...)
	require.NoError(t, err)

	return provider
}

func TestWhatsOnChainProviderGetExchangeRate(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		want     float64
		wantErr  error
		wantHits int
	}{
		{name: "valid rate", status: http.StatusOK, body: `{"currency":"USD","rate":15.17,"time":1}`, want: 15.17, wantHits: 1},
		{name: "server error is retried", status: http.StatusInternalServerError, body: "boom", wantErr: whatsonchain.ErrRequestFailed, wantHits: 3},
		{name: "rate limit is retried", status: http.StatusTooManyRequests, wantErr: whatsonchain.ErrRequestFailed, wantHits: 3},
		{name: "bad request is not retried", status: http.StatusBadRequest, body: "bad", wantErr: whatsonchain.ErrRequestFailed, wantHits: 1},
		{name: "not found without a body", status: http.StatusNotFound, wantErr: whatsonchain.ErrExchangeRateNotFound, wantHits: 1},
		{name: "not found with a JSON body", status: http.StatusNotFound, body: `{"error":"not found"}`, wantErr: whatsonchain.ErrExchangeRateNotFound, wantHits: 1},
		{name: "empty body", status: http.StatusOK, wantErr: whatsonchain.ErrExchangeRateNotFound, wantHits: 1},
		{name: "body without a rate", status: http.StatusOK, body: `{}`, wantErr: whatsonchain.ErrExchangeRateNotFound, wantHits: 1},
		{name: "zero rate", status: http.StatusOK, body: `{"rate":0}`, wantErr: whatsonchain.ErrExchangeRateNotFound, wantHits: 1},
		{name: "negative rate", status: http.StatusOK, body: `{"rate":-2}`, wantErr: whatsonchain.ErrExchangeRateNotFound, wantHits: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(whatsonchain.EnvAPIKey, "")

			rs, baseURL := newRateServer(t, rateResponse{status: tt.status, body: tt.body})
			provider := newHTTPTestProvider(t, baseURL)

			rate, err := provider.GetExchangeRate(t.Context())

			hits, paths, _ := rs.requests()
			assert.Equal(t, tt.wantHits, hits)

			for _, path := range paths {
				assert.Equal(t, exchangeRatePath, path)
			}

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Contains(t, err.Error(), "fetch exchange rate from WhatsOnChain")

				return
			}

			require.NoError(t, err)
			assert.InDelta(t, tt.want, rate, 1e-9)
		})
	}
}

func TestWhatsOnChainProviderMalformedBody(t *testing.T) {
	t.Setenv(whatsonchain.EnvAPIKey, "")

	_, baseURL := newRateServer(t, rateResponse{status: http.StatusOK, body: "not json"})

	_, err := newHTTPTestProvider(t, baseURL).GetExchangeRate(t.Context())

	var syntaxErr *json.SyntaxError
	require.ErrorAs(t, err, &syntaxErr)
}

func TestWhatsOnChainProviderRetryRecovers(t *testing.T) {
	t.Setenv(whatsonchain.EnvAPIKey, "")

	rs, baseURL := newRateServer(t,
		rateResponse{status: http.StatusServiceUnavailable},
		rateResponse{status: http.StatusOK, body: `{"rate":20}`},
	)

	rate, err := newHTTPTestProvider(t, baseURL).GetExchangeRate(t.Context())
	require.NoError(t, err)
	assert.InDelta(t, 20.0, rate, 1e-9)

	hits, _, _ := rs.requests()
	assert.Equal(t, 2, hits)
}

func TestWhatsOnChainProviderRetryAfter(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		retryAfter string
	}{
		{name: "rate limit", status: http.StatusTooManyRequests, retryAfter: "60"},
		{name: "unavailable", status: http.StatusServiceUnavailable, retryAfter: "120"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(whatsonchain.EnvAPIKey, "")

			rs, baseURL := newRateServer(t, rateResponse{status: tt.status, retryAfter: tt.retryAfter})

			// The server's own backoff: a Retry-After past its 2-second maximum is not waited out,
			// so the failure comes back at once (and a cached rate, if any, is served sooner).
			provider, err := NewWhatsOnChainProvider(t.Context(), whatsonchain.WithBaseURL(baseURL))
			require.NoError(t, err)

			start := time.Now()
			_, err = provider.GetExchangeRate(t.Context())

			require.ErrorIs(t, err, whatsonchain.ErrRequestFailed)
			assert.Less(t, time.Since(start), 5*time.Second)

			hits, _, _ := rs.requests()
			assert.Equal(t, 1, hits, "no retry when Retry-After asks for longer than the maximum backoff")
		})
	}
}

func TestWhatsOnChainProviderStopsWhenContextIsCancelled(t *testing.T) {
	t.Setenv(whatsonchain.EnvAPIKey, "")

	rs, baseURL := newRateServer(t, rateResponse{status: http.StatusServiceUnavailable})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	rs.mu.Lock()
	rs.onRequest = cancel
	rs.mu.Unlock()

	// A backoff far longer than the test: the call only returns quickly if the wait ends on cancellation.
	provider := newHTTPTestProvider(t, baseURL, whatsonchain.WithBackoff(time.Minute, time.Minute, 1, 0))

	start := time.Now()
	_, err := provider.GetExchangeRate(ctx)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(start), 10*time.Second)

	hits, _, _ := rs.requests()
	assert.Equal(t, 1, hits, "no retry after cancellation")
}

func TestWhatsOnChainProviderAPIKeyHeader(t *testing.T) {
	tests := []struct {
		name   string
		envKey string
		opts   []whatsonchain.ClientOption
		want   string
	}{
		{name: "no key configured", want: ""},
		{name: "key from an option", opts: []whatsonchain.ClientOption{whatsonchain.WithAPIKey(testAPIKey)}, want: testAPIKey},
		{name: "key from the environment", envKey: testAPIKey, want: testAPIKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(whatsonchain.EnvAPIKey, tt.envKey)

			rs, baseURL := newRateServer(t, rateResponse{status: http.StatusOK, body: `{"rate":20}`})

			_, err := newHTTPTestProvider(t, baseURL, tt.opts...).GetExchangeRate(t.Context())
			require.NoError(t, err)

			_, _, apiKeys := rs.requests()
			assert.Equal(t, []string{tt.want}, apiKeys)
		})
	}
}

// fakeRateClient is an exchangeRateGetter whose answer the test controls.
type fakeRateClient struct {
	mu    sync.Mutex
	rate  *whatsonchain.ExchangeRate
	err   error
	calls int
}

// newFakeRateClient returns a fake answering with rate.
func newFakeRateClient(rate float64) *fakeRateClient {
	return &fakeRateClient{rate: &whatsonchain.ExchangeRate{Currency: "USD", Rate: rate}}
}

func (f *fakeRateClient) GetExchangeRate(context.Context) (*whatsonchain.ExchangeRate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls++
	if f.err != nil {
		return nil, f.err
	}

	return f.rate, nil
}

// answer makes the fake answer with rate from now on.
func (f *fakeRateClient) answer(rate float64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.rate = &whatsonchain.ExchangeRate{Currency: "USD", Rate: rate}
	f.err = nil
}

// fail makes the fake fail from now on.
func (f *fakeRateClient) fail() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.err = errFakeRateClient
}

func (f *fakeRateClient) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls
}

// ageCache makes the cached rate look as if it had been fetched age ago.
func ageCache(provider *WhatsOnChainProvider, age time.Duration) {
	provider.mu.Lock()
	provider.cachedAt = time.Now().Add(-age)
	provider.mu.Unlock()
}

func TestWhatsOnChainProviderServesFreshRateFromCache(t *testing.T) {
	client := newFakeRateClient(20)
	provider := newWhatsOnChainProvider(client)

	rate, err := provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	require.InDelta(t, 20.0, rate, 1e-9)

	// Within the refresh interval the API is not asked again, so its new rate is not seen yet.
	client.answer(25)

	for range 5 {
		rate, err = provider.GetExchangeRate(t.Context())
		require.NoError(t, err)
		assert.InDelta(t, 20.0, rate, 1e-9)
	}

	assert.Equal(t, 1, client.callCount())
}

func TestWhatsOnChainProviderRefreshesAfterInterval(t *testing.T) {
	client := newFakeRateClient(20)
	provider := newWhatsOnChainProvider(client)

	_, err := provider.GetExchangeRate(t.Context())
	require.NoError(t, err)

	client.answer(25)
	ageCache(provider, provider.refreshInterval+time.Second)

	rate, err := provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	assert.InDelta(t, 25.0, rate, 1e-9)
	assert.Equal(t, 2, client.callCount())
}

func TestWhatsOnChainProviderFallsBackToRecentRate(t *testing.T) {
	client := newFakeRateClient(20)
	provider := newWhatsOnChainProvider(client)

	rate, err := provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	require.InDelta(t, 20.0, rate, 1e-9)

	// The API starts failing once the rate is due for a refresh: the cached rate is served while
	// it is younger than the cache TTL.
	client.fail()
	ageCache(provider, provider.refreshInterval+time.Second)

	rate, err = provider.GetExchangeRate(t.Context())
	require.NoError(t, err)
	assert.InDelta(t, 20.0, rate, 1e-9)
	assert.Equal(t, 2, client.callCount(), "a stale rate is only served after the API was asked")

	// Once the cached rate is older than the TTL, the failure is returned.
	ageCache(provider, provider.cacheTTL+time.Second)

	_, err = provider.GetExchangeRate(t.Context())
	require.ErrorIs(t, err, errFakeRateClient)
	assert.Contains(t, err.Error(), "fetch exchange rate from WhatsOnChain")
}

func TestWhatsOnChainProviderNeverCachesInvalidRate(t *testing.T) {
	t.Run("nothing cached yet", func(t *testing.T) {
		client := newFakeRateClient(0)
		provider := newWhatsOnChainProvider(client)

		_, err := provider.GetExchangeRate(t.Context())
		require.ErrorIs(t, err, ErrInvalidExchangeRate)

		// The zero rate was not cached, so the next failure has nothing to fall back to.
		client.fail()

		_, err = provider.GetExchangeRate(t.Context())
		require.ErrorIs(t, err, errFakeRateClient)
		assert.Equal(t, 2, client.callCount())
	})

	t.Run("a valid rate is cached", func(t *testing.T) {
		client := newFakeRateClient(20)
		provider := newWhatsOnChainProvider(client)

		_, err := provider.GetExchangeRate(t.Context())
		require.NoError(t, err)

		// An invalid refresh counts as a failure: the cached rate is served and kept as it was.
		client.answer(0)
		ageCache(provider, provider.refreshInterval+time.Second)

		rate, err := provider.GetExchangeRate(t.Context())
		require.NoError(t, err)
		assert.InDelta(t, 20.0, rate, 1e-9)

		ageCache(provider, provider.cacheTTL+time.Second)

		_, err = provider.GetExchangeRate(t.Context())
		require.ErrorIs(t, err, ErrInvalidExchangeRate)
	})
}

func TestWhatsOnChainProviderNilResponse(t *testing.T) {
	client := &fakeRateClient{}

	_, err := newWhatsOnChainProvider(client).GetExchangeRate(t.Context())
	require.ErrorIs(t, err, whatsonchain.ErrExchangeRateNotFound)
}

// wocClient returns the go-whatsonchain client behind provider.
func wocClient(t *testing.T, provider *WhatsOnChainProvider) whatsonchain.ClientInterface {
	t.Helper()

	client, ok := provider.client.(whatsonchain.ClientInterface)
	require.True(t, ok, "the provider wraps a %T", provider.client)

	return client
}

func TestNewWhatsOnChainProviderDefaults(t *testing.T) {
	t.Setenv(whatsonchain.EnvAPIKey, "")

	provider, err := NewWhatsOnChainProvider(t.Context())
	require.NoError(t, err)

	client := wocClient(t, provider)
	assert.Equal(t, whatsonchain.NetworkMain, client.Network())
	assert.Equal(t, "https://api.whatsonchain.com/v1/", client.BaseURL())
	assert.Equal(t, whatsOnChainTimeout, client.RequestTimeout())
	assert.Equal(t, whatsOnChainRetryCount, client.RequestRetryCount())
	assert.Empty(t, client.APIKey())

	// The server relies on go-whatsonchain's default backoff (about 250 ms, then 500 ms, plus up to
	// 100 ms of jitter); this fails if a release changes it, so the change is a deliberate choice.
	initial, maxTimeout, factor, jitter := client.BackoffConfig()
	assert.Equal(t, 250*time.Millisecond, initial)
	assert.Equal(t, 2*time.Second, maxTimeout)
	assert.InDelta(t, 2.0, factor, 1e-9)
	assert.Equal(t, 100*time.Millisecond, jitter)

	assert.Equal(t, whatsOnChainRefreshInterval, provider.refreshInterval)
	assert.Equal(t, whatsOnChainCacheTTL, provider.cacheTTL)
	assert.Less(t, provider.refreshInterval, provider.cacheTTL, "a stale rate must outlive the refresh interval to be a fallback")
}

func TestNewWhatsOnChainProviderOptionsOverrideDefaults(t *testing.T) {
	t.Setenv(whatsonchain.EnvAPIKey, "")

	provider, err := NewWhatsOnChainProvider(t.Context(),
		whatsonchain.WithNetwork(whatsonchain.NetworkTest),
		whatsonchain.WithBaseURL("http://127.0.0.1:9999/v1"),
		whatsonchain.WithRequestTimeout(time.Second),
		whatsonchain.WithRequestRetryCount(0),
		whatsonchain.WithAPIKey(testAPIKey),
	)
	require.NoError(t, err)

	client := wocClient(t, provider)
	assert.Equal(t, whatsonchain.NetworkTest, client.Network())
	assert.Equal(t, "http://127.0.0.1:9999/v1/", client.BaseURL())
	assert.Equal(t, time.Second, client.RequestTimeout())
	assert.Equal(t, 0, client.RequestRetryCount())
	assert.Equal(t, testAPIKey, client.APIKey())
}

func TestNewWhatsOnChainProviderEmptyOptionsKeepDefaults(t *testing.T) {
	t.Setenv(whatsonchain.EnvAPIKey, "")

	// The server passes its WHATS_ON_CHAIN_API_KEY and WHATS_ON_CHAIN_API_URL settings as is, unset or not.
	provider, err := NewWhatsOnChainProvider(t.Context(), whatsonchain.WithAPIKey(""), whatsonchain.WithBaseURL(""))
	require.NoError(t, err)

	client := wocClient(t, provider)
	assert.Equal(t, "https://api.whatsonchain.com/v1/", client.BaseURL())
	assert.Empty(t, client.APIKey())
}

func TestNewWhatsOnChainProviderInvalidNetwork(t *testing.T) {
	_, err := NewWhatsOnChainProvider(t.Context(), whatsonchain.WithNetwork("bogus"))
	require.ErrorIs(t, err, whatsonchain.ErrInvalidNetwork)
}
