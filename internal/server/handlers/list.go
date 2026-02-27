package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
)

// ListHandler handles GET /list requests.
type ListHandler struct {
	WalletProvider *walletpkg.Provider
}

type listUpload struct {
	UhrpURL    string `json:"uhrpUrl"`
	ExpiryTime int64  `json:"expiryTime"`
}

type listResponse struct {
	Status  string       `json:"status"`
	Uploads []listUpload `json:"uploads,omitempty"`
	Code    string       `json:"code,omitempty"`
	Desc    string       `json:"description,omitempty"`
}

// ServeHTTP handles the /list endpoint request.
// @Summary List user's active file uploads
// @Description Retrieve a list of all currently active (non-expired) UHRP advertisements created by the authenticated user.
// @Accept json
// @Produce json
// @Success 200 {object} listResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /list [get]
func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey := middlewares.GetIdentityKey(r.Context())
	if identityKey == nil {
		responses.WriteError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Missing or invalid identityKey.")
		return
	}

	wallet := h.WalletProvider.GetWallet()
	if wallet == nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized.")
		return
	}

	outputs, err := walletpkg.ListAdvertisementsByUploader(r.Context(), wallet, identityKey.ToDERHex())
	if err != nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_LIST", "Failed to list outputs.")
		return
	}

	now := time.Now().Unix()
	uploads := make([]listUpload, 0)
	for _, out := range outputs {
		meta := walletpkg.ParseCustomInstructions(out.CustomInstructions)
		if meta == nil {
			continue
		}
		expiryTime := parseExpiryTime(meta["expiryTime"])
		if expiryTime > 0 && expiryTime < now {
			continue // expired
		}
		uploads = append(uploads, listUpload{
			UhrpURL:    meta["uhrpURL"],
			ExpiryTime: expiryTime,
		})
	}

	responses.WriteJSON(w, http.StatusOK, listResponse{
		Status:  "success",
		Uploads: uploads,
	})
}

// parseExpiryTime parses an RFC3339 or unix timestamp string.
func parseExpiryTime(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, s)
	if err == nil {
		return t.Unix()
	}
	// Try as raw unix
	var unix int64
	fmt.Sscanf(s, "%d", &unix)
	return unix
}
