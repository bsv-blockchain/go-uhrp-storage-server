package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bsv-blockchain/go-bsv-middleware/pkg/middleware"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
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

	wallet := h.WalletProvider.GetWallet()
	if wallet == nil {
		writeError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized.")
		return
	}

	// 1. Find the existing advertisement via FindAdvertisementByUhrpURL
	matchedOutputPtr, meta, err := walletpkg.FindAdvertisementByUhrpURL(r.Context(), wallet, req.UhrpURL)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, "ERR_NOT_FOUND", "No advertisement found for the given uhrpUrl.")
		} else {
			writeError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW", "Failed to query wallet outputs.")
		}
		return
	}
	matchedOutput := *matchedOutputPtr

	// 2. Calculate pricing
	prevExpiry := parseExpiryTime(meta["expiryTime"])
	var fileSize int64
	fmt.Sscanf(meta["fileSize"], "%d", &fileSize)

	amount, err := h.Calculator.GetPrice(fileSize, req.AdditionalMinutes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW", "Failed to calculate price.")
		return
	}

	// 3. Compute new expiry
	newExpiry := prevExpiry + (req.AdditionalMinutes * 60)

	// 4. Renew the advertisement via walletpkg
	p := walletpkg.CreateAdParams{
		UhrpURL:       meta["uhrpURL"],
		HostingDomain: meta["hostingDomain"],
		ExpirySecs:    newExpiry,
		ContentType:   meta["contentType"],
		FileSize:      fileSize,
		ObjectID:      meta["objectID"],
		Uploader:      meta["uploader"],
	}

	if err := walletpkg.RenewAdvertisement(r.Context(), wallet, matchedOutput, p); err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL_RENEW", "An error occurred while handling the renewal.")
		return
	}

	writeJSON(w, http.StatusOK, renewResponse{
		Status:         "success",
		PrevExpiryTime: prevExpiry,
		NewExpiryTime:  newExpiry,
		Amount:         amount,
	})
}
