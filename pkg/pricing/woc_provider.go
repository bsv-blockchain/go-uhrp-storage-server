package pricing

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mrz1836/go-whatsonchain"
)

const (
	// whatsOnChainTimeout bounds each request to the API.
	whatsOnChainTimeout = 10 * time.Second

	// whatsOnChainRetryCount is how many times a failed request (network error, 429 or 5xx) is
	// retried, so a rate lookup makes at most three attempts.
	whatsOnChainRetryCount = 2

	// The backoff between attempts: about 250 ms and then 500 ms, plus up to 100 ms of jitter.
	// The library's default (2 ms to 10 ms) is too short to wait out a rate limit.
	whatsOnChainBackoffInitial = 250 * time.Millisecond
	whatsOnChainBackoffMax     = 2 * time.Second
	whatsOnChainBackoffFactor  = 2.0
	whatsOnChainBackoffJitter  = 100 * time.Millisecond

	// whatsOnChainRefreshInterval is how long a fetched rate is served without asking the API again.
	whatsOnChainRefreshInterval = 5 * time.Minute

	// whatsOnChainCacheTTL is the age past which a cached rate is never served, even while the API is down.
	whatsOnChainCacheTTL = 15 * time.Minute
)

// exchangeRateGetter is the part of the WhatsOnChain client the provider uses.
type exchangeRateGetter interface {
	GetExchangeRate(ctx context.Context) (*whatsonchain.ExchangeRate, error)
}

// WhatsOnChainProvider is the default ExchangeRateProvider: it reads the BSV/USD rate from
// WhatsOnChain through go-whatsonchain and caches it.
type WhatsOnChainProvider struct {
	client exchangeRateGetter

	mu              sync.RWMutex
	cachedRate      float64
	cachedAt        time.Time
	refreshInterval time.Duration
	cacheTTL        time.Duration
}

// NewWhatsOnChainProvider creates a provider backed by a go-whatsonchain client. The client
// reads mainnet (the BSV/USD rate is a market price), times out each request after 10 seconds,
// and retries a failed request twice with exponential backoff. opts are applied after these
// defaults, so they override them: for example whatsonchain.WithAPIKey for higher rate limits,
// or whatsonchain.WithBaseURL for a WhatsOnChain-compatible mirror or a local stub.
//
// whatsonchain.WithHTTPClient replaces the library's retrying client, so with it a failed
// request is not retried.
func NewWhatsOnChainProvider(ctx context.Context, opts ...whatsonchain.ClientOption) (*WhatsOnChainProvider, error) {
	defaults := []whatsonchain.ClientOption{
		whatsonchain.WithNetwork(whatsonchain.NetworkMain),
		whatsonchain.WithRequestTimeout(whatsOnChainTimeout),
		whatsonchain.WithRequestRetryCount(whatsOnChainRetryCount),
		whatsonchain.WithBackoff(
			whatsOnChainBackoffInitial, whatsOnChainBackoffMax, whatsOnChainBackoffFactor, whatsOnChainBackoffJitter,
		),
	}

	client, err := whatsonchain.NewClient(ctx, append(defaults, opts...)...)
	if err != nil {
		return nil, fmt.Errorf("create WhatsOnChain client: %w", err)
	}

	return newWhatsOnChainProvider(client), nil
}

// newWhatsOnChainProvider wraps client in the rate cache (tests pass a fake client).
func newWhatsOnChainProvider(client exchangeRateGetter) *WhatsOnChainProvider {
	return &WhatsOnChainProvider{
		client:          client,
		refreshInterval: whatsOnChainRefreshInterval,
		cacheTTL:        whatsOnChainCacheTTL,
	}
}

// GetExchangeRate implements the ExchangeRateProvider interface.
// A rate fetched within the refresh interval is served from the cache. Otherwise the API is asked
// again, and if it fails, a cached rate younger than the cache TTL is served instead of the error.
func (o *WhatsOnChainProvider) GetExchangeRate(ctx context.Context) (float64, error) {
	if rate, _, ok := o.cached(o.refreshInterval); ok {
		return rate, nil
	}

	rate, err := o.fetch(ctx)
	if err != nil {
		if cached, age, ok := o.cached(o.cacheTTL); ok {
			slog.WarnContext(ctx, "Serving a cached BSV/USD exchange rate after a failed refresh",
				"error", err, "cacheAge", age.Round(time.Second).String())

			return cached, nil
		}

		return 0, err
	}

	o.mu.Lock()
	o.cachedRate = rate
	o.cachedAt = time.Now()
	o.mu.Unlock()

	return rate, nil
}

// fetch asks WhatsOnChain for the current rate. A zero or non-finite rate (a 404 with a JSON body
// decodes to a zero rate) is an error, so it is never cached.
func (o *WhatsOnChainProvider) fetch(ctx context.Context) (float64, error) {
	resp, err := o.client.GetExchangeRate(ctx)
	if err == nil && resp == nil {
		err = whatsonchain.ErrExchangeRateNotFound
	}

	if err != nil {
		return 0, fmt.Errorf("fetch exchange rate from WhatsOnChain: %w", err)
	}

	rate, err := validateExchangeRate(resp.Rate)
	if err != nil {
		return 0, fmt.Errorf("fetch exchange rate from WhatsOnChain: %w", err)
	}

	return rate, nil
}

// cached returns the cached rate and its age when there is one younger than maxAge.
func (o *WhatsOnChainProvider) cached(maxAge time.Duration) (float64, time.Duration, bool) {
	o.mu.RLock()
	rate := o.cachedRate
	age := time.Since(o.cachedAt)
	o.mu.RUnlock()

	return rate, age, rate > 0 && age < maxAge
}
