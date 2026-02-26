package handlers

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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, description string) {
	writeJSON(w, status, ErrorResponse{
		Status:      "error",
		Code:        code,
		Description: description,
	})
}
