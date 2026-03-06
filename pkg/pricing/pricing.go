package pricing

import (
	"fmt"
	"math"
	"time"
)

// ExchangeRateProvider defines the interface for fetching BSV/USD exchange rates.
type ExchangeRateProvider interface {
	GetExchangeRate() (float64, error)
}

// Calculator computes storage prices in satoshis.
type Calculator struct {
	PricePerGBMonth float64
	Provider        ExchangeRateProvider
}

// NewCalculator creates a new pricing calculator with the given oracle.
func NewCalculator(pricePerGBMonth float64, provider ExchangeRateProvider) *Calculator {
	return &Calculator{
		PricePerGBMonth: pricePerGBMonth,
		Provider:        provider,
	}
}

// GetPrice returns the satoshi price for storing fileSize bytes for retentionPeriod minutes.
func (c *Calculator) GetPrice(fileSize int64, retentionPeriod int64) (int64, error) {
	fileSizeGB := float64(fileSize) / 1_000_000_000

	retentionMonths := float64(retentionPeriod) / (60 * 24 * 30)

	usdPrice := fileSizeGB * retentionMonths * c.PricePerGBMonth

	var exchangeRate float64
	var err error
	maxRetries := 3
	for i := range maxRetries {
		exchangeRate, err = c.Provider.GetExchangeRate()
		if err == nil {
			break
		}
		fmt.Printf("Error fetching exchange rate (attempt %d/%d): %v\n", i+1, maxRetries, err)
		if i < maxRetries-1 {
			time.Sleep(250 * time.Millisecond) // wait before retrying because the error often occurs when there is too many requests
		}
	}
	if err != nil {
		return 0, fmt.Errorf("failed to fetch exchange rate after %d attempts: %w", maxRetries, err)
	}

	exchangeRateInSatoshis := 1.0 / (exchangeRate / 100_000_000)

	satPrice := int64(math.Max(10, math.Floor(usdPrice*exchangeRateInSatoshis)))
	return satPrice, nil
}
