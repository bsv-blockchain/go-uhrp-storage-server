package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// renewSearchLimit is how many of the caller's matching outputs are searched for the advertisement to renew.
const renewSearchLimit = 200

// RenewHandler handles POST /renew requests.
type RenewHandler struct {
	Calculator     *pricing.Calculator
	WalletProvider *walletpkg.Provider
	Logger         *slog.Logger
}

// RenewRequest is the body of POST /renew.
type RenewRequest struct {
	UhrpURL           string `json:"uhrpUrl"`
	AdditionalMinutes int64  `json:"additionalMinutes"`
	Limit             *int   `json:"limit,omitempty"`
	Offset            *int   `json:"offset,omitempty"`
}

// RenewResponse is the body returned by POST /renew.
type RenewResponse struct {
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
// @Param request body RenewRequest true "UHRP URL and additional minutes to host"
// @Success 200 {object} RenewResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security AuthfetchIdentity
// @Router /renew [post]
func (h *RenewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey := middlewares.GetIdentityKey(r.Context())
	if identityKey == nil {
		responses.WriteError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Missing or invalid identityKey.")
		return
	}

	var req RenewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.WriteError(w, http.StatusBadRequest, "ERR_MISSING_FIELDS", "Invalid request body.")
		return
	}

	if req.UhrpURL == "" || req.AdditionalMinutes <= 0 {
		responses.WriteError(w, http.StatusBadRequest, "ERR_MISSING_FIELDS", "Missing objectIdentifier or additionalMinutes.")
		return
	}

	uploader := identityKey.ToDERHex()

	output, meta, beef, err := h.WalletProvider.FindAdvertisementByUhrpURL(r.Context(), req.UhrpURL, uploader, renewSearchLimit, 0)
	if err != nil {
		if errors.Is(err, walletpkg.ErrAdvertisementNotFound) {
			responses.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "No advertisement found for the given uhrpUrl.")
			return
		}

		h.Logger.ErrorContext(r.Context(), "Failed to find advertisement to renew", "uhrpUrl", req.UhrpURL, "error", err)
		responses.WriteError(w, http.StatusInternalServerError, "ERR_FIND", "Failed to retrieve the existing advertisement.")

		return
	}

	fileSize, err := meta.SizeBytes()
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "Advertisement has no valid size", "uhrpUrl", req.UhrpURL, "error", err)
		responses.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW", "Failed to calculate price.")

		return
	}

	amount, err := h.Calculator.GetPrice(r.Context(), fileSize, req.AdditionalMinutes)
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "Failed to price renewal", "error", err)
		responses.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW", "Failed to calculate price.")

		return
	}

	prevExpiry := meta.ExpiryTime
	newExpiry := prevExpiry + (req.AdditionalMinutes * 60)

	err = h.WalletProvider.RenewAdvertisement(r.Context(), h.WalletProvider.OverlayNetwork(), output, beef, walletpkg.CreateAdParams{
		URL:           req.UhrpURL,
		ExpirySecs:    newExpiry,
		ContentType:   meta.ContentType,
		ContentLength: fileSize,
		ObjectID:      meta.ObjectIdentifier,
		Uploader:      uploader,
	})
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "Failed to renew advertisement", "uhrpUrl", req.UhrpURL, "error", err)
		responses.WriteError(w, http.StatusInternalServerError, "ERR_RENEW", "Failed to renew advertisement on chain.")

		return
	}

	responses.WriteJSON(w, http.StatusOK, RenewResponse{
		Status:         "success",
		PrevExpiryTime: prevExpiry,
		NewExpiryTime:  newExpiry,
		Amount:         amount,
	})
}
