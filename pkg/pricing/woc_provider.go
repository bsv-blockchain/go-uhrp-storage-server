package pricing

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// WhatsOnChainProvider is the default implementation that fetches rates from Whatsonchain.
type WhatsOnChainProvider struct {
	httpClient *http.Client

	mu          sync.RWMutex
	cachedRate  float64
	cachedAt    time.Time
	cacheTTL    time.Duration
}

// NewWhatsOnChainProvider creates a new oracle instance targeting Whatsonchain APIs.
func NewWhatsOnChainProvider() *WhatsOnChainProvider {
	return &WhatsOnChainProvider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cacheTTL:   15 * time.Minute,
	}
}

// exchangeRateResponse is the response structure for the Whatsonchain API.
type exchangeRateResponse struct {
	Rate float64 `json:"rate"`
}

// GetExchangeRate implements the ExchangeRateProvider interface.
// Returns cached rate on API failure if a recent rate is available.
func (o *WhatsOnChainProvider) GetExchangeRate() (float64, error) {
	resp, err := o.httpClient.Get("https://api.whatsonchain.com/v1/bsv/main/exchangerate")
	if err != nil {
		return o.fallback(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return o.fallback(fmt.Errorf("unexpected status %d", resp.StatusCode))
	}

	var data exchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return o.fallback(err)
	}

	if data.Rate <= 0 {
		return o.fallback(fmt.Errorf("invalid rate: %f", data.Rate))
	}

	o.mu.Lock()
	o.cachedRate = data.Rate
	o.cachedAt = time.Now()
	o.mu.Unlock()

	return data.Rate, nil
}

// fallback returns a cached rate if available, otherwise propagates the error.
func (o *WhatsOnChainProvider) fallback(fetchErr error) (float64, error) {
	o.mu.RLock()
	rate := o.cachedRate
	age := time.Since(o.cachedAt)
	o.mu.RUnlock()

	if rate > 0 && age < o.cacheTTL {
		return rate, nil
	}

	return 0, fetchErr
}
