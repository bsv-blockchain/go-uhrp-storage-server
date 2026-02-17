package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/pricing"
)

func TestQuoteHandler_MissingFileSize(t *testing.T) {
	h := &QuoteHandler{
		Calculator:        pricing.NewCalculator(0.03),
		MinHostingMinutes: 0,
	}
	body := `{"retentionPeriod": 60}`
	req := httptest.NewRequest("POST", "/quote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestQuoteHandler_MissingRetention(t *testing.T) {
	h := &QuoteHandler{
		Calculator:        pricing.NewCalculator(0.03),
		MinHostingMinutes: 0,
	}
	body := `{"fileSize": 1024}`
	req := httptest.NewRequest("POST", "/quote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestQuoteHandler_RetentionTooLarge(t *testing.T) {
	h := &QuoteHandler{
		Calculator:        pricing.NewCalculator(0.03),
		MinHostingMinutes: 0,
	}
	body := `{"fileSize": 1024, "retentionPeriod": 70000000}`
	req := httptest.NewRequest("POST", "/quote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestQuoteHandler_ValidRequest(t *testing.T) {
	h := &QuoteHandler{
		Calculator:        pricing.NewCalculator(0.03),
		MinHostingMinutes: 0,
	}
	body := `{"fileSize": 1024, "retentionPeriod": 60}`
	req := httptest.NewRequest("POST", "/quote", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
