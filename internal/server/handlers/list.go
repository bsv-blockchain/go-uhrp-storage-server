package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bsv-blockchain/go-bsv-middleware/pkg/middleware"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
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

	wallet := h.WalletProvider.GetWallet()
	if wallet == nil {
		writeError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized.")
		return
	}

	outputs, err := walletpkg.ListAdvertisementsByUploader(r.Context(), wallet, identityKey.ToDERHex())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_LIST", "Failed to list outputs.")
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

	writeJSON(w, http.StatusOK, listResponse{
		Status:  "success",
		Uploads: uploads,
	})
}

func isUnknownKey(key *ec.PublicKey) bool {
	return key == nil || middleware.IsUnknownIdentity(key)
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
