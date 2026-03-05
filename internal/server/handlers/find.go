package handlers

import (
	"net/http"
	"time"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
)

// FindHandler handles GET /find requests.
type FindHandler struct {
	WalletProvider *walletpkg.Provider
}

type findData struct {
	Name       string `json:"name"`
	Size       string `json:"size"`
	MimeType   string `json:"mimeType"`
	ExpiryTime int64  `json:"expiryTime"`
}

type findResponse struct {
	Status      string    `json:"status"`
	Data        *findData `json:"data,omitempty"`
	Code        string    `json:"code,omitempty"`
	Description string    `json:"description,omitempty"`
}

// ServeHTTP handles the /find endpoint request.
// @Summary Find file metadata
// @Description Find specific UHRP file advertisement.
// @Accept json
// @Produce json
// @Param uhrpUrl query string true "UHRP URL of the file to find"
// @Success 200 {object} findResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 404 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /find [get]
func (h *FindHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey := middlewares.GetIdentityKey(r.Context())
	if identityKey == nil {
		responses.WriteError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Missing or invalid identityKey.")
		return
	}

	uhrpURL := r.URL.Query().Get("uhrpUrl")
	if uhrpURL == "" {
		responses.WriteError(w, http.StatusBadRequest, "ERR_NO_UHRP_URL", "You must provide a uhrpUrl query parameter")
		return
	}

	wallet := h.WalletProvider.GetWallet()
	if wallet == nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized.")
		return
	}

	_, meta, _, err := walletpkg.FindAdvertisementByUhrpURL(r.Context(), wallet, uhrpURL, identityKey.ToDERHex())
	if err != nil {
		responses.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "No active advertisement found for the given uhrpUrl.")
		return
	}

	now := time.Now().Unix()
	expiryTime := meta.ExpiryTime
	if expiryTime > 0 && expiryTime < now {
		responses.WriteError(w, http.StatusNotFound, "ERR_NOT_FOUND", "No active advertisement found for the given uhrpUrl.")
		return
	}

	responses.WriteJSON(w, http.StatusOK, findResponse{
		Status: "success",
		Data: &findData{
			Name:       meta.ObjectIdentifier,
			Size:       meta.Size,
			MimeType:   meta.ContentType,
			ExpiryTime: expiryTime,
		},
	})
}
