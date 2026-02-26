package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// QuoteHandler handles POST /quote requests.
type QuoteHandler struct {
	Calculator        *pricing.Calculator
	MinHostingMinutes int
}

type quoteRequest struct {
	FileSize        json.Number `json:"fileSize"`
	RetentionPeriod json.Number `json:"retentionPeriod"`
}

type quoteResponse struct {
	Quote int64 `json:"quote"`
}

func (h *QuoteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req quoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "Invalid request body.")
		return
	}

	fileSize, err := req.FileSize.Int64()
	if err != nil || req.FileSize.String() == "" {
		writeError(w, http.StatusBadRequest, "ERR_NO_SIZE", "Provide the size of the file you want to host.")
		return
	}

	retentionPeriod, err := req.RetentionPeriod.Int64()
	if err != nil || req.RetentionPeriod.String() == "" {
		writeError(w, http.StatusBadRequest, "ERR_NO_RETENTION_PERIOD", "Specify the number of minutes to host the file.")
		return
	}

	if fileSize <= 0 {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_SIZE", "The file size must be an integer.")
		return
	}

	if retentionPeriod < int64(h.MinHostingMinutes) {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_RETENTION_PERIOD",
			fmt.Sprintf("The retention period must be an integer and must be more than %d minutes", h.MinHostingMinutes))
		return
	}

	if retentionPeriod > 69_000_000 {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_RETENTION_PERIOD",
			"The retention period must be less than 69 million minutes (about 130 years)")
		return
	}

	satPrice, err := h.Calculator.GetPrice(fileSize, retentionPeriod)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "An internal error has occurred.")
		return
	}

	writeJSON(w, http.StatusOK, quoteResponse{Quote: satPrice})
}
