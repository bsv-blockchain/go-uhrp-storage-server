package pricing

import (
	"testing"
)

type mockExchangeRateProvider struct{}

func (m mockExchangeRateProvider) GetExchangeRate() (float64, error) {
	return 30.0, nil
}

func TestGetPrice_MinimumPrice(t *testing.T) {
	calc := NewCalculator(0.03, mockExchangeRateProvider{})
	// Very small file, short period should still return minimum of 10 sats
	price, err := calc.GetPrice(1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price < 10 {
		t.Errorf("expected minimum price of 10, got %d", price)
	}
}

func TestGetPrice_LargerFile(t *testing.T) {
	calc := NewCalculator(0.03, mockExchangeRateProvider{})
	// 1 GB for 1 month (43200 minutes)
	price, err := calc.GetPrice(1_000_000_000, 43200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price < 10 {
		t.Errorf("expected price >= 10, got %d", price)
	}
}
