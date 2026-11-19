package pricing

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errRateUnavailable = errors.New("rate unavailable")

// staticRateProvider always returns the same rate.
type staticRateProvider float64

func (p staticRateProvider) GetExchangeRate(context.Context) (float64, error) {
	return float64(p), nil
}

// failingRateProvider always fails, counting its calls.
type failingRateProvider struct {
	calls atomic.Int32
}

func (p *failingRateProvider) GetExchangeRate(context.Context) (float64, error) {
	p.calls.Add(1)

	return 0, errRateUnavailable
}

func TestCalculatorGetPrice(t *testing.T) {
	tests := []struct {
		name            string
		fileSize        int64
		retentionPeriod int64
		want            int64
	}{
		{name: "tiny file pays the minimum", fileSize: 1, retentionPeriod: 1, want: MinimumPriceSatoshis},
		{name: "zero size pays the minimum", fileSize: 0, retentionPeriod: 60, want: MinimumPriceSatoshis},
		// 1 GB for one 30-day month at $0.03/GB-month and $30/BSV: 0.03 / 30 * 1e8 sats.
		{name: "one GB for one month", fileSize: 1_000_000_000, retentionPeriod: 43_200, want: 100_000},
		{name: "price scales with retention", fileSize: 1_000_000_000, retentionPeriod: 2 * 43_200, want: 200_000},
	}

	calc := NewCalculator(0.03, staticRateProvider(30))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calc.GetPrice(t.Context(), tt.fileSize, tt.retentionPeriod)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCalculatorGetPriceReturnsProviderError(t *testing.T) {
	provider := &failingRateProvider{}

	_, err := NewCalculator(0.03, provider).GetPrice(t.Context(), 1, 1)
	require.ErrorIs(t, err, errRateUnavailable)
	assert.Equal(t, int32(1), provider.calls.Load(), "retrying a failed lookup is the provider's job")
}

func TestCalculatorGetPriceRejectsInvalidRates(t *testing.T) {
	for _, rate := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := NewCalculator(0.03, staticRateProvider(rate)).GetPrice(t.Context(), 1, 1)
		require.ErrorIs(t, err, ErrInvalidExchangeRate, "rate %v", rate)
	}
}

func TestCalculatorGetPriceRejectsOverflow(t *testing.T) {
	calc := NewCalculator(1e12, staticRateProvider(0.0001))

	_, err := calc.GetPrice(t.Context(), math.MaxInt64, math.MaxInt64)
	require.ErrorIs(t, err, ErrPriceOverflow)
}

func FuzzCalculatorGetPrice(f *testing.F) {
	f.Add(int64(1024), int64(60), 30.0)
	f.Add(int64(0), int64(0), 1.0)
	f.Add(int64(-1), int64(-1), 0.5)
	f.Add(int64(math.MaxInt64), int64(math.MaxInt64), 1e-9)

	f.Fuzz(func(t *testing.T, fileSize, retentionPeriod int64, rate float64) {
		price, err := NewCalculator(0.03, staticRateProvider(rate)).GetPrice(t.Context(), fileSize, retentionPeriod)
		if err != nil {
			if !errors.Is(err, ErrInvalidExchangeRate) {
				require.ErrorIs(t, err, ErrPriceOverflow)
			}

			return
		}

		assert.GreaterOrEqual(t, price, int64(MinimumPriceSatoshis))
	})
}

func BenchmarkCalculatorGetPrice(b *testing.B) {
	calc := NewCalculator(0.03, staticRateProvider(30))
	ctx := b.Context()

	b.ReportAllocs()

	for b.Loop() {
		_, _ = calc.GetPrice(ctx, 1_000_000_000, 43_200)
	}
}
