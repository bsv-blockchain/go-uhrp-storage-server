package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	base58 "github.com/bsv-blockchain/go-sdk/compat/base58"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

const (
	// objectIDBytes is the number of random bytes in a new object identifier (Base58-encoded).
	objectIDBytes = 16

	// expiryGraceSeconds is added to the paid retention when stamping a presigned URL's expiry.
	expiryGraceSeconds = 300
)

// UploadHandler handles POST /upload requests.
type UploadHandler struct {
	Calculator        *pricing.Calculator
	WalletProvider    *walletpkg.Provider
	HostingDomain     string
	MinHostingMinutes int
	Logger            *slog.Logger
}

// UploadRequest is the body of POST /upload.
type UploadRequest struct {
	FileSize        int64 `json:"fileSize"`
	RetentionPeriod int64 `json:"retentionPeriod"`
}

// UploadResponse is the body returned by POST /upload.
type UploadResponse struct {
	Status          string            `json:"status"`
	UploadURL       string            `json:"uploadURL,omitempty"`
	RequiredHeaders map[string]string `json:"requiredHeaders,omitempty"`
	Amount          int64             `json:"amount,omitempty"`
	Description     string            `json:"description,omitempty"`
	Code            string            `json:"code,omitempty"`
}

// ServeHTTP handles the /upload endpoint request.
// @Summary Request an upload URL
// @Description Get a pre-signed URL and payment details to upload a file to the UHRP storage server.
// @Accept json
// @Produce json
// @Param request body UploadRequest true "File size and desired retention period"
// @Success 200 {object} UploadResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Security AuthfetchIdentity
// @Router /upload [post]
func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey := middlewares.GetIdentityKey(r.Context())
	if identityKey == nil {
		responses.WriteError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Missing or invalid identityKey.")
		return
	}

	var req UploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "Invalid request body.")
		return
	}

	if code, description := h.validate(req); code != "" {
		responses.WriteError(w, http.StatusBadRequest, code, description)
		return
	}

	amount, err := h.Calculator.GetPrice(r.Context(), req.FileSize, req.RetentionPeriod)
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "Failed to price upload", "error", err)
		responses.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL_UPLOAD", "An internal error occurred while handling upload.")

		return
	}

	objectIdentifier := base58.Encode(randomBytes(objectIDBytes))
	expiryTime := (req.RetentionPeriod * 60) + time.Now().Unix()
	customTime := time.Unix(expiryTime+expiryGraceSeconds, 0).UTC().Format(time.RFC3339)

	queryStr := fmt.Sprintf("fileSize=%d&objectID=%s&expiry=%s&uploader=%s",
		req.FileSize, objectIdentifier, customTime, identityKey.ToDERHex())

	hmac, err := h.WalletProvider.CreateUploaderHMAC(r.Context(), queryStr)
	if err != nil {
		h.Logger.ErrorContext(r.Context(), "Failed to create upload HMAC", "error", err)
		responses.WriteError(w, http.StatusInternalServerError, "ERR_HMAC", "Failed to create HMAC.")

		return
	}

	responses.WriteJSON(w, http.StatusOK, UploadResponse{
		Status:          "success",
		UploadURL:       h.uploadURL(queryStr, hmac),
		RequiredHeaders: map[string]string{},
		Amount:          amount,
		Description:     "File can now be uploaded.",
	})
}

// validate checks an upload request, returning the error code and description of the first problem.
func (h *UploadHandler) validate(req UploadRequest) (code, description string) {
	switch {
	case req.FileSize <= 0:
		return "ERR_INVALID_SIZE", "The file size must be a positive integer."
	case req.RetentionPeriod <= 0:
		return "ERR_NO_RETENTION_PERIOD", "You must specify the number of minutes to host the file."
	case req.RetentionPeriod < int64(h.MinHostingMinutes):
		return "ERR_INVALID_RETENTION_PERIOD", fmt.Sprintf("The retention period must be >= %d minutes", h.MinHostingMinutes)
	case req.FileSize > maxFileSizeBytes:
		return "ERR_INVALID_SIZE", fmt.Sprintf("Max supported file size is %d bytes.", maxFileSizeBytes)
	default:
		return "", ""
	}
}

// uploadURL builds the presigned PUT URL for the hosting domain (http for localhost, https otherwise).
func (h *UploadHandler) uploadURL(queryStr, hmac string) string {
	scheme := "https://"
	if strings.HasPrefix(h.HostingDomain, "localhost") {
		scheme = "http://"
	}

	return fmt.Sprintf("%s%s/put?%s&hmac=%s", scheme, h.HostingDomain, queryStr, hmac)
}

// randomBytes returns n cryptographically random bytes.
func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error (Go 1.24+)

	return b
}
