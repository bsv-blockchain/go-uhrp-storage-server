package pricing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	// whatsOnChainExchangeRateURL is WhatsOnChain's BSV/USD exchange-rate endpoint.
	whatsOnChainExchangeRateURL = "https://api.whatsonchain.com/v1/bsv/main/exchangerate"

	whatsOnChainTimeout = 10 * time.Second

	// whatsOnChainRefreshInterval is how long a fetched rate is served without asking the API again.
	whatsOnChainRefreshInterval = 5 * time.Minute

	// whatsOnChainCacheTTL is the age past which a cached rate is never served, even while the API is down.
	whatsOnChainCacheTTL = 15 * time.Minute

	// maxExchangeRateResponseBytes bounds how much of a response body is decoded.
	maxExchangeRateResponseBytes = 64 << 10
)

// ErrUnexpectedStatus is returned when the exchange-rate API answers with a non-200 status.
var ErrUnexpectedStatus = errors.New("unexpected exchange rate response status")

// WhatsOnChainProvider is the default implementation that fetches rates from Whatsonchain.
type WhatsOnChainProvider struct {
	httpClient *http.Client
	url        string

	mu              sync.RWMutex
	cachedRate      float64
	cachedAt        time.Time
	refreshInterval time.Duration
	cacheTTL        time.Duration
}

// NewWhatsOnChainProvider creates a new oracle instance targeting Whatsonchain APIs.
func NewWhatsOnChainProvider() *WhatsOnChainProvider {
	return NewWhatsOnChainProviderWithURL("")
}

// NewWhatsOnChainProviderWithURL creates a provider for any endpoint that answers like WhatsOnChain's
// exchange-rate API (a JSON object with a "rate" field, in USD per BSV). An empty url selects WhatsOnChain.
func NewWhatsOnChainProviderWithURL(url string) *WhatsOnChainProvider {
	if url == "" {
		url = whatsOnChainExchangeRateURL
	}

	return newWhatsOnChainProvider(url, &http.Client{Timeout: whatsOnChainTimeout})
}

// newWhatsOnChainProvider creates a provider for the given endpoint and client (tests point it at a local server).
func newWhatsOnChainProvider(url string, client *http.Client) *WhatsOnChainProvider {
	return &WhatsOnChainProvider{
		httpClient:      client,
		url:             url,
		refreshInterval: whatsOnChainRefreshInterval,
		cacheTTL:        whatsOnChainCacheTTL,
	}
}

// exchangeRateResponse is the response structure for the Whatsonchain API.
type exchangeRateResponse struct {
	Rate float64 `json:"rate"`
}

// GetExchangeRate implements the ExchangeRateProvider interface.
// A rate fetched within the refresh interval is served from the cache. Otherwise the API is asked
// again, and if it fails, a cached rate younger than the cache TTL is served instead of the error.
func (o *WhatsOnChainProvider) GetExchangeRate(ctx context.Context) (float64, error) {
	if rate, ok := o.cached(o.refreshInterval); ok {
		return rate, nil
	}

	rate, err := o.fetch(ctx)
	if err != nil {
		if cached, ok := o.cached(o.cacheTTL); ok {
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

// fetch requests the current rate from the API.
func (o *WhatsOnChainProvider) fetch(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.url, nil)
	if err != nil {
		return 0, fmt.Errorf("build exchange rate request: %w", err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("fetch exchange rate: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	var data exchangeRateResponse
	if err = json.NewDecoder(io.LimitReader(resp.Body, maxExchangeRateResponseBytes)).Decode(&data); err != nil {
		return 0, fmt.Errorf("decode exchange rate: %w", err)
	}

	return validateExchangeRate(data.Rate)
}

// cached returns the cached rate when there is one younger than maxAge.
func (o *WhatsOnChainProvider) cached(maxAge time.Duration) (float64, bool) {
	o.mu.RLock()
	rate := o.cachedRate
	age := time.Since(o.cachedAt)
	o.mu.RUnlock()

	return rate, rate > 0 && age < maxAge
}
