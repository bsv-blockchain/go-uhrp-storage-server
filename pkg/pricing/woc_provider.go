package pricing

import (
	"encoding/json"
	"net/http"
	"time"
)

// WhatsOnChainProvider is the default implementation that fetches rates from Whatsonchain.
type WhatsOnChainProvider struct {
	httpClient *http.Client
}

// NewWhatsOnChainProvider creates a new oracle instance targeting Whatsonchain APIs.
func NewWhatsOnChainProvider() *WhatsOnChainProvider {
	return &WhatsOnChainProvider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// exchangeRateResponse is the response structure for the Whatsonchain API.
type exchangeRateResponse struct {
	Rate float64 `json:"rate"`
}

// GetExchangeRate implements the ExchangeRateProvider interface.
func (o *WhatsOnChainProvider) GetExchangeRate() (float64, error) {
	resp, err := o.httpClient.Get("https://api.whatsonchain.com/v1/bsv/main/exchangerate")
	if err != nil {
		return 0.0, err
	}
	defer resp.Body.Close()

	var data exchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0.0, err
	}

	return data.Rate, nil
}
