// Package handlers implements the UHRP storage server's HTTP endpoints.
package handlers

import (
	"encoding/json"
	"net/http"
)

const (
	// maxFileSizeBytes is the largest file an upload may declare (11 GB).
	maxFileSizeBytes = 11_000_000_000

	// maxRetentionMinutes is the longest retention period a quote may ask for (about 130 years).
	maxRetentionMinutes = 69_000_000
)

// paginationRequest is the optional JSON body of GET /list and GET /find.
type paginationRequest struct {
	Limit  *uint32 `json:"limit"`
	Offset *uint32 `json:"offset"`
}

// decodePagination reads the optional limit and offset from a JSON request body.
// A missing or malformed body leaves both at zero, which selects the wallet defaults.
func decodePagination(r *http.Request) (limit, offset uint32) {
	if r.Body == nil {
		return 0, 0
	}

	var req paginationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return 0, 0
	}

	if req.Limit != nil {
		limit = *req.Limit
	}

	if req.Offset != nil {
		offset = *req.Offset
	}

	return limit, offset
}
