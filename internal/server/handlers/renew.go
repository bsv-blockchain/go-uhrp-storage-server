package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// RenewHandler handles POST /renew requests.
type RenewHandler struct {
	Calculator     *pricing.Calculator
	WalletProvider *walletpkg.Provider
	Logger         *slog.Logger
}

type renewRequest struct {
	UhrpURL           string `json:"uhrpUrl"`
	AdditionalMinutes int64  `json:"additionalMinutes"`
	Limit             *int   `json:"limit,omitempty"`
	Offset            *int   `json:"offset,omitempty"`
}

type renewResponse struct {
	Status         string `json:"status"`
	PrevExpiryTime int64  `json:"prevExpiryTime,omitempty"`
	NewExpiryTime  int64  `json:"newExpiryTime,omitempty"`
	Amount         int64  `json:"amount,omitempty"`
	Code           string `json:"code,omitempty"`
	Description    string `json:"description,omitempty"`
}

// ServeHTTP handles the /renew endpoint request.
// @Summary Renew an active file
// @Description Extend the storage time for an existing UHRP file advertisement.
// @Accept json
// @Produce json
// @Param request body renewRequest true "UHRP URL and additional minutes to host"
// @Success 200 {object} renewResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /renew [post]
func (h *RenewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey := middlewares.GetIdentityKey(r.Context())
	if identityKey == nil {
		responses.WriteError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Missing or invalid identityKey.")
		return
	}

	var req renewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.WriteError(w, http.StatusBadRequest, "ERR_MISSING_FIELDS", "Invalid request body.")
		return
	}

	if req.UhrpURL == "" || req.AdditionalMinutes <= 0 {
		responses.WriteError(w, http.StatusBadRequest, "ERR_MISSING_FIELDS", "Missing objectIdentifier or additionalMinutes.")
		return
	}

	output, meta, beef, err := h.WalletProvider.FindAdvertisementByUhrpURL(r.Context(), req.UhrpURL, identityKey.ToDERHex(), 200, 0)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			responses.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "No advertisement found for the given uhrpUrl.")
		} else {
			responses.WriteError(w, http.StatusInternalServerError, "ERR_FIND", "Failed to retrieve the existing advertisement.")
		}
		return
	}

	prevExpiry := meta.ExpiryTime
	var fileSize int64
	fmt.Sscanf(meta.Size, "%d", &fileSize)

	amount, err := h.Calculator.GetPrice(fileSize, req.AdditionalMinutes)
	if err != nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW", "Failed to calculate price.")
		return
	}

	newExpiry := prevExpiry + (req.AdditionalMinutes * 60)

	p := walletpkg.CreateAdParams{
		URL:           req.UhrpURL,
		ExpirySecs:    newExpiry,
		ContentType:   meta.ContentType,
		ContentLength: fileSize,
		ObjectID:      meta.ObjectIdentifier,
		Uploader:      identityKey.ToDERHex(),
	}

	if err := h.WalletProvider.RenewAdvertisement(r.Context(), h.WalletProvider.OverlayNetwork(), output, beef, p); err != nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_RENEW", "Failed to renew advertisement on chain.")
		return
	}

	responses.WriteJSON(w, http.StatusOK, renewResponse{
		Status:         "success",
		PrevExpiryTime: prevExpiry,
		NewExpiryTime:  newExpiry,
		Amount:         amount,
	})
}
