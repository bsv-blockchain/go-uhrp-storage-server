package handlers

import (
	"net/http"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-bsv-middleware/pkg/middleware"
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

func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey, err := middleware.ShouldGetIdentity(r.Context())
	if err != nil || isUnknownKey(identityKey) {
		writeError(w, http.StatusBadRequest, "ERR_MISSING_IDENTITY_KEY", "Missing authfetch identityKey.")
		return
	}

	// In a full implementation, we'd use wallet.ListOutputs to query the
	// 'uhrp advertisements' basket filtered by the uploader's identity key tag.
	// Then decode tags to extract uhrpUrl and expiryTime, filtering out expired entries.

	// TODO: Implement with wallet.ListOutputs when go-sdk wallet toolbox is available
	// wallet := h.WalletProvider.GetWallet()
	// outputs, _ := wallet.ListOutputs(...)

	// For now, return empty list
	writeJSON(w, http.StatusOK, listResponse{
		Status:  "success",
		Uploads: []listUpload{},
	})
}

func isUnknownKey(key *ec.PublicKey) bool {
	return key == nil || middleware.IsUnknownIdentity(key)
}
