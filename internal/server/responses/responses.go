package responses

import (
	"encoding/json"
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
	json.NewEncoder(w).Encode(v)
}

// WriteError writes an error response.
func WriteError(w http.ResponseWriter, status int, code, description string) {
	WriteJSON(w, status, ErrorResponse{
		Status:      "error",
		Code:        code,
		Description: description,
	})
}
