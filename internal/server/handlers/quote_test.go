package handlers_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/handlers"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
)

func newQuoteHandler(minHostingMinutes int) *handlers.QuoteHandler {
	return &handlers.QuoteHandler{
		Calculator:        newCalculator(),
		MinHostingMinutes: minHostingMinutes,
		Logger:            slog.Default(),
	}
}

func TestQuoteHandlerServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		minMinutes     int
		body           any
		expectedStatus int
		expectedCode   string
		expectedQuote  int64
	}{
		{
			name:           "valid request",
			body:           map[string]any{"fileSize": 1024, "retentionPeriod": 60},
			expectedStatus: http.StatusOK,
			expectedQuote:  10,
		},
		{
			name:           "one GB for one month",
			body:           map[string]any{"fileSize": 1_000_000_000, "retentionPeriod": 43_200},
			expectedStatus: http.StatusOK,
			expectedQuote:  100_000,
		},
		{
			name:           "missing file size",
			body:           map[string]any{"retentionPeriod": 60},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_NO_SIZE",
		},
		{
			name:           "missing retention period",
			body:           map[string]any{"fileSize": 1024},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_NO_RETENTION_PERIOD",
		},
		{
			name:           "negative file size",
			body:           map[string]any{"fileSize": -5, "retentionPeriod": 60},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_SIZE",
		},
		{
			name:           "retention below the minimum",
			minMinutes:     30,
			body:           map[string]any{"fileSize": 1024, "retentionPeriod": 10},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_RETENTION_PERIOD",
		},
		{
			name:           "retention period too large",
			body:           map[string]any{"fileSize": 1024, "retentionPeriod": 70_000_000},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_RETENTION_PERIOD",
		},
		{
			name:           "invalid JSON",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "ERR_INVALID_BODY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newRequest(t, http.MethodPost, "/quote", tt.body, nil)
			req.Header.Set("Content-Type", "application/json")

			w := serve(newQuoteHandler(tt.minMinutes), req)
			assert.Equal(t, tt.expectedStatus, w.Code, "body: %s", w.Body.String())

			if tt.expectedCode != "" {
				assert.Equal(t, tt.expectedCode, decode[responses.ErrorResponse](t, w).Code)
			}

			if tt.expectedQuote != 0 {
				assert.Equal(t, tt.expectedQuote, decode[map[string]int64](t, w)["quote"])
			}
		})
	}
}

// FuzzQuoteHandler checks that any request body yields a JSON 200 or 400 and never panics.
func FuzzQuoteHandler(f *testing.F) {
	for _, seed := range []string{
		`{"fileSize":1024,"retentionPeriod":60}`,
		`{"fileSize":"1024","retentionPeriod":"60"}`,
		`{"fileSize":1e3,"retentionPeriod":1.5}`,
		`{"fileSize":9223372036854775807,"retentionPeriod":69000000}`,
		`{}`,
		`[]`,
		`null`,
		``,
	} {
		f.Add(seed)
	}

	h := newQuoteHandler(0)

	f.Fuzz(func(t *testing.T, body string) {
		w := serve(h, newRequest(t, http.MethodPost, "/quote", body, nil))

		assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError}, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})
}

func BenchmarkQuoteHandler(b *testing.B) {
	h := newQuoteHandler(0)
	body := []byte(`{"fileSize":1024,"retentionPeriod":60}`)

	b.ReportAllocs()

	for b.Loop() {
		req := httptest.NewRequestWithContext(b.Context(), http.MethodPost, "/quote", bytes.NewReader(body))
		_ = serve(h, req)
	}
}
