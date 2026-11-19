// Package responses writes the server's JSON responses, including the standard error shape.
package responses

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Status      string `json:"status"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status line is already sent, so an encoding or write error (usually a client
	// that disconnected) cannot be reported to the client; it is only logged.
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("Failed to write JSON response", "status", status, "error", err)
	}
}

// WriteError writes an error response.
func WriteError(w http.ResponseWriter, status int, code, description string) {
	WriteJSON(w, status, ErrorResponse{
		Status:      "error",
		Code:        code,
		Description: description,
	})
}
