package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

type mockOracle struct{}

func (m mockOracle) GetExchangeRate() (float64, error) {
	return 30.0, nil
}

func TestQuoteHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		body           interface{}
		expectedStatus int
	}{
		{
			name:           "Valid Request",
			body:           map[string]interface{}{"fileSize": 1024, "retentionPeriod": 60},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing File Size",
			body:           map[string]interface{}{"retentionPeriod": 60},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing Retention Period",
			body:           map[string]interface{}{"fileSize": 1024},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Retention Period Too Large",
			body:           map[string]interface{}{"fileSize": 1024, "retentionPeriod": 70000000},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &handlers.QuoteHandler{
				Calculator:        pricing.NewCalculator(0.03, mockOracle{}),
				MinHostingMinutes: 0,
			}

			var bodyReader *bytes.Reader
			if s, ok := tt.body.(string); ok {
				bodyReader = bytes.NewReader([]byte(s))
			} else {
				bodyBytes, _ := json.Marshal(tt.body)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest("POST", "/quote", bodyReader)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d. Body: %s", tt.expectedStatus, w.Code, w.Body.String())
			}
		})
	}
}
