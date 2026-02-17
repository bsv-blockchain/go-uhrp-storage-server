package pricing

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

// Calculator computes storage prices in satoshis.
type Calculator struct {
	PricePerGBMonth float64
	httpClient      *http.Client
}

// NewCalculator creates a new pricing calculator.
func NewCalculator(pricePerGBMonth float64) *Calculator {
	return &Calculator{
		PricePerGBMonth: pricePerGBMonth,
		httpClient:      &http.Client{Timeout: 10 * time.Second},
	}
}

type exchangeRateResponse struct {
	Rate float64 `json:"rate"`
}

// GetPrice returns the satoshi price for storing fileSize bytes for retentionPeriod minutes.
func (c *Calculator) GetPrice(fileSize int64, retentionPeriod int64) (int64, error) {
	// File size in GB
	fileSizeGB := float64(fileSize) / 1_000_000_000

	// Retention period in months (minutes -> months)
	retentionMonths := float64(retentionPeriod) / (60 * 24 * 30)

	// USD price
	usdPrice := fileSizeGB * retentionMonths * c.PricePerGBMonth

	// Get exchange rate
	exchangeRate := c.fetchExchangeRate()

	// Convert USD to satoshis: 1 BSV = exchangeRate USD, 1 BSV = 100_000_000 sats
	exchangeRateInSatoshis := 1.0 / (exchangeRate / 100_000_000)

	satPrice := int64(math.Max(10, math.Floor(usdPrice*exchangeRateInSatoshis)))
	return satPrice, nil
}

func (c *Calculator) fetchExchangeRate() float64 {
	resp, err := c.httpClient.Get("https://api.whatsonchain.com/v1/bsv/main/exchangerate")
	if err != nil {
		fmt.Printf("Exchange rate failed, using fallback rate of 30: %v\n", err)
		return 30
	}
	defer resp.Body.Close()

	var data exchangeRateResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Rate == 0 {
		fmt.Printf("Invalid rate response, using fallback rate of 30: %v\n", err)
		return 30
	}
	return data.Rate
}
