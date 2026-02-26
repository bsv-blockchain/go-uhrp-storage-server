package handlers

import (
	"net/http"
	"time"

	"github.com/bsv-blockchain/go-bsv-middleware/pkg/middleware"
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

func (h *FindHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey, err := middleware.ShouldGetIdentity(r.Context())
	if err != nil || isUnknown(identityKey) {
		writeError(w, http.StatusBadRequest, "ERR_MISSING_IDENTITY_KEY", "Missing authfetch identityKey.")
		return
	}

	uhrpURL := r.URL.Query().Get("uhrpUrl")
	if uhrpURL == "" {
		writeError(w, http.StatusBadRequest, "ERR_NO_UHRP_URL", "You must provide a uhrpUrl query parameter")
		return
	}

	wallet := h.WalletProvider.GetWallet()
	if wallet == nil {
		writeError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized.")
		return
	}

	_, meta, err := walletpkg.FindAdvertisementByUhrpURL(r.Context(), wallet, uhrpURL)
	if err != nil {
		writeError(w, http.StatusNotFound, "ERR_NOT_FOUND", "No active advertisement found for the given uhrpUrl.")
		return
	}

	now := time.Now().Unix()
	expiryTime := parseExpiryTime(meta["expiryTime"])
	if expiryTime > 0 && expiryTime < now {
		writeError(w, http.StatusNotFound, "ERR_NOT_FOUND", "No active advertisement found for the given uhrpUrl.")
		return
	}

	writeJSON(w, http.StatusOK, findResponse{
		Status: "success",
		Data: &findData{
			Name:       meta["objectID"],
			Size:       meta["fileSize"],
			MimeType:   meta["contentType"],
			ExpiryTime: expiryTime,
		},
	})
}
