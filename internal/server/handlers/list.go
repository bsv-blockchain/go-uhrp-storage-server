package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
)

// ListHandler handles GET /list requests.
type ListHandler struct {
	WalletProvider *walletpkg.Provider
	Logger         *slog.Logger
}

type listReqBody struct {
	Limit  *uint32 `json:"limit"`
	Offset *uint32 `json:"offset"`
}

type listUpload struct {
	UhrpURL    string `json:"uhrpUrl"`
	ExpiryTime int64  `json:"expiryTime"`
}

type ListResponse struct {
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
// @Success 200 {object} ListResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /list [get]
func (h *ListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey := middlewares.GetIdentityKey(r.Context())
	if identityKey == nil {
		responses.WriteError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Missing or invalid identityKey.")
		return
	}

	var limit, offset uint32
	if r.Body != nil {
		var req listReqBody
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.Limit != nil {
				limit = *req.Limit
			}
			if req.Offset != nil {
				offset = *req.Offset
			}
		}
	}

	metadatas, err := h.WalletProvider.ListAdvertisementsByUploader(r.Context(), identityKey.ToDERHex(), limit, offset)
	if err != nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_LIST", "Failed to list outputs.")
		return
	}

	uploads := make([]listUpload, 0)
	for _, meta := range metadatas {
		uploads = append(uploads, listUpload{
			UhrpURL:    meta.URL,
			ExpiryTime: meta.ExpiryTime,
		})
	}

	responses.WriteJSON(w, http.StatusOK, ListResponse{
		Status:  "success",
		Uploads: uploads,
	})
}
