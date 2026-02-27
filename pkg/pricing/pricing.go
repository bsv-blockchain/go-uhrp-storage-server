package pricing

import (
	"math"
)

// ExchangeRateProvider defines the interface for fetching BSV/USD exchange rates.
type ExchangeRateProvider interface {
	GetExchangeRate() float64
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
	// File size in GB
	fileSizeGB := float64(fileSize) / 1_000_000_000

	// Retention period in months (minutes -> months)
	retentionMonths := float64(retentionPeriod) / (60 * 24 * 30)

	// USD price
	usdPrice := fileSizeGB * retentionMonths * c.PricePerGBMonth

	// Get exchange rate via Oracle
	exchangeRate := c.Provider.GetExchangeRate()

	// Convert USD to satoshis: 1 BSV = exchangeRate USD, 1 BSV = 100_000_000 sats
	exchangeRateInSatoshis := 1.0 / (exchangeRate / 100_000_000)

	satPrice := int64(math.Max(10, math.Floor(usdPrice*exchangeRateInSatoshis)))
	return satPrice, nil
}
