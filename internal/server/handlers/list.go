package handlers

import (
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
	Uploads []listUpload `json:"uploads"`
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

	metadatas, err := walletpkg.ListAdvertisementsByUploader(r.Context(), wallet, identityKey.ToDERHex())
	if err != nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_LIST", "Failed to list outputs.")
		return
	}

	now := time.Now().Unix()
	uploads := make([]listUpload, 0)
	for _, meta := range metadatas {
		if meta.ExpiryTime > 0 && meta.ExpiryTime < now {
			continue // expired
		}
		uploads = append(uploads, listUpload{
			UhrpURL:    meta.URL,
			ExpiryTime: meta.ExpiryTime,
		})
	}

	responses.WriteJSON(w, http.StatusOK, listResponse{
		Status:  "success",
		Uploads: uploads,
	})
}
