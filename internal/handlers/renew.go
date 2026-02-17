package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/bsv-blockchain/go-bsv-middleware/pkg/middleware"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/pricing"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
)

// RenewHandler handles POST /renew requests.
type RenewHandler struct {
	Calculator     *pricing.Calculator
	WalletProvider *walletpkg.Provider
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

func (h *RenewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey, err := middleware.ShouldGetIdentity(r.Context())
	if err != nil || isUnknown(identityKey) {
		writeError(w, http.StatusBadRequest, "ERR_MISSING_IDENTITY_KEY", "Missing authfetch identityKey.")
		return
	}

	var req renewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_MISSING_FIELDS", "Invalid request body.")
		return
	}

	if req.UhrpURL == "" || req.AdditionalMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "ERR_MISSING_FIELDS", "Missing objectIdentifier or additionalMinutes.")
		return
	}

	if req.AdditionalMinutes <= 0 {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_TIME", "Additional Minutes must be a positive integer")
		return
	}

	// In a full implementation:
	// 1. Get metadata (objectIdentifier, size, expiryTime) from wallet.ListOutputs
	// 2. Calculate new expiry = prevExpiry + (additionalMinutes * 60)
	// 3. Calculate price based on file size and additionalMinutes
	// 4. Redeem old PushDrop token and create new one with updated expiry
	// 5. Broadcast via SHIP

	// TODO: Implement with wallet.ListOutputs, PushDrop, and SHIPBroadcaster

	writeError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW",
		"An error occurred while handling the renewal.")
}
