package responses_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	responses.WriteJSON(w, http.StatusCreated, map[string]int{"quote": 10})

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"quote":10}`, w.Body.String())
}

func TestWriteJSONUnencodableValue(t *testing.T) {
	w := httptest.NewRecorder()
	responses.WriteJSON(w, http.StatusOK, map[string]any{"bad": make(chan int)})

	assert.Equal(t, http.StatusOK, w.Code, "the status is already sent when encoding fails")
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		description string
		want        string
	}{
		{name: "code and description", code: "ERR_X", description: "Bad thing.", want: `{"status":"error","code":"ERR_X","description":"Bad thing."}`},
		{name: "description only", description: "Size mismatch", want: `{"status":"error","description":"Size mismatch"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			responses.WriteError(w, http.StatusBadRequest, tt.code, tt.description)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.JSONEq(t, tt.want, w.Body.String())
		})
	}
}
