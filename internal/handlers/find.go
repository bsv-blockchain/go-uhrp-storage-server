package handlers

import (
	"net/http"

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

	// In a full implementation, we'd query wallet.ListOutputs for matching
	// UHRP advertisement tokens and extract metadata from tags.

	// TODO: Implement with wallet.ListOutputs when go-sdk wallet toolbox is available

	writeError(w, http.StatusInternalServerError, "ERR_FIND",
		"An error occurred while retrieving the file from uhrpUrl.")
}
