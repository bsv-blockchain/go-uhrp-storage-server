// Package pricing calculates UHRP storage prices in satoshis from a USD price per
// GB-month and the live BSV/USD exchange rate.
package pricing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"
)

const (
	// MinimumPriceSatoshis is the lowest price ever quoted, however small or short the upload.
	MinimumPriceSatoshis = 10

	bytesPerGB      = 1_000_000_000
	minutesPerMonth = 60 * 24 * 30
	satoshisPerBSV  = 100_000_000

	// exchangeRateAttempts is how many times GetPrice asks the provider for a rate.
	exchangeRateAttempts = 3

	// exchangeRateRetryDelay is the wait between attempts; rate APIs often fail under bursts of requests.
	exchangeRateRetryDelay = 250 * time.Millisecond
)

var (
	// ErrInvalidExchangeRate is returned when a provider reports a rate that is not a positive, finite number.
	ErrInvalidExchangeRate = errors.New("invalid exchange rate")

	// ErrPriceOverflow is returned when a price does not fit in an int64 number of satoshis.
	ErrPriceOverflow = errors.New("price exceeds the maximum representable satoshi amount")
)

// ExchangeRateProvider defines the interface for fetching BSV/USD exchange rates.
type ExchangeRateProvider interface {
	// GetExchangeRate returns the price of one BSV in USD.
	GetExchangeRate(ctx context.Context) (float64, error)
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
// The price is never below MinimumPriceSatoshis.
func (c *Calculator) GetPrice(ctx context.Context, fileSize, retentionPeriod int64) (int64, error) {
	exchangeRate, err := c.exchangeRate(ctx)
	if err != nil {
		return 0, err
	}

	fileSizeGB := float64(fileSize) / bytesPerGB
	retentionMonths := float64(retentionPeriod) / minutesPerMonth
	usdPrice := fileSizeGB * retentionMonths * c.PricePerGBMonth
	exchangeRateInSatoshis := 1.0 / (exchangeRate / satoshisPerBSV)

	satPrice := math.Max(MinimumPriceSatoshis, math.Floor(usdPrice*exchangeRateInSatoshis))
	if math.IsNaN(satPrice) || satPrice >= math.MaxInt64 {
		return 0, fmt.Errorf("%w: %d bytes for %d minutes", ErrPriceOverflow, fileSize, retentionPeriod)
	}

	return int64(satPrice), nil
}

// exchangeRate asks the provider for the BSV/USD rate, retrying transient failures.
func (c *Calculator) exchangeRate(ctx context.Context) (float64, error) {
	var err error

	for attempt := 1; attempt <= exchangeRateAttempts; attempt++ {
		var rate float64
		if rate, err = c.Provider.GetExchangeRate(ctx); err == nil {
			return validateExchangeRate(rate)
		}

		slog.WarnContext(ctx, "Failed to fetch exchange rate",
			"attempt", attempt, "maxAttempts", exchangeRateAttempts, "error", err)

		if attempt == exchangeRateAttempts {
			break
		}

		if waitErr := sleepContext(ctx, exchangeRateRetryDelay); waitErr != nil {
			return 0, fmt.Errorf("fetch exchange rate: %w", waitErr)
		}
	}

	return 0, fmt.Errorf("failed to fetch exchange rate after %d attempts: %w", exchangeRateAttempts, err)
}

// validateExchangeRate rejects rates that would make a price meaningless (zero, negative, NaN or infinite).
func validateExchangeRate(rate float64) (float64, error) {
	if !(rate > 0) || math.IsInf(rate, 0) {
		return 0, fmt.Errorf("%w: %v", ErrInvalidExchangeRate, rate)
	}

	return rate, nil
}

// sleepContext waits for d, returning early with the context's error if it is canceled first.
func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
