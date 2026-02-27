package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// QuoteHandler handles POST /quote requests.
type QuoteHandler struct {
	Calculator        *pricing.Calculator
	MinHostingMinutes int
}

type quoteRequest struct {
	FileSize        json.Number `json:"fileSize" swaggertype:"integer" example:"1024"`
	RetentionPeriod json.Number `json:"retentionPeriod" swaggertype:"integer" example:"60"`
}

type quoteResponse struct {
	Quote int64 `json:"quote"`
}

// ServeHTTP handles the /quote endpoint request.
// @Summary Calculate storage price
// @Description Get a price quote in satoshis for hosting a file of a specific size for a specific duration.
// @Accept json
// @Produce json
// @Param request body quoteRequest true "File size and retention period details"
// @Success 200 {object} quoteResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /quote [post]
func (h *QuoteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req quoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "Invalid request body.")
		return
	}

	fileSize, err := req.FileSize.Int64()
	if err != nil || req.FileSize.String() == "" {
		responses.WriteError(w, http.StatusBadRequest, "ERR_NO_SIZE", "Provide the size of the file you want to host.")
		return
	}

	retentionPeriod, err := req.RetentionPeriod.Int64()
	if err != nil || req.RetentionPeriod.String() == "" {
		responses.WriteError(w, http.StatusBadRequest, "ERR_NO_RETENTION_PERIOD", "Specify the number of minutes to host the file.")
		return
	}

	if fileSize <= 0 {
		responses.WriteError(w, http.StatusBadRequest, "ERR_INVALID_SIZE", "The file size must be an integer.")
		return
	}

	if retentionPeriod < int64(h.MinHostingMinutes) {
		responses.WriteError(w, http.StatusBadRequest, "ERR_INVALID_RETENTION_PERIOD",
			fmt.Sprintf("The retention period must be an integer and must be more than %d minutes", h.MinHostingMinutes))
		return
	}

	if retentionPeriod > 69_000_000 {
		responses.WriteError(w, http.StatusBadRequest, "ERR_INVALID_RETENTION_PERIOD",
			"The retention period must be less than 69 million minutes (about 130 years)")
		return
	}

	satPrice, err := h.Calculator.GetPrice(fileSize, retentionPeriod)
	if err != nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL", "An internal error has occurred.")
		return
	}

	responses.WriteJSON(w, http.StatusOK, quoteResponse{Quote: satPrice})
}
