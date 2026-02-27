package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/responses"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
)

// UploadHandler handles POST /upload requests.
type UploadHandler struct {
	Calculator        *pricing.Calculator
	WalletProvider    *walletpkg.Provider
	HostingDomain     string
	MinHostingMinutes int
}

type uploadRequest struct {
	FileSize        int64 `json:"fileSize"`
	RetentionPeriod int64 `json:"retentionPeriod"`
}

type uploadResponse struct {
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
// @Param request body uploadRequest true "File size and desired retention period"
// @Success 200 {object} uploadResponse
// @Failure 400 {object} responses.ErrorResponse
// @Failure 401 {object} responses.ErrorResponse
// @Failure 500 {object} responses.ErrorResponse
// @Router /upload [post]
func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey := middlewares.GetIdentityKey(r.Context())
	if identityKey == nil {
		responses.WriteError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Missing or invalid identityKey.")
		return
	}

	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.WriteError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "Invalid request body.")
		return
	}

	if req.FileSize <= 0 {
		responses.WriteError(w, http.StatusBadRequest, "ERR_INVALID_SIZE", "The file size must be a positive integer.")
		return
	}
	if req.RetentionPeriod <= 0 {
		responses.WriteError(w, http.StatusBadRequest, "ERR_NO_RETENTION_PERIOD", "You must specify the number of minutes to host the file.")
		return
	}
	if req.RetentionPeriod < int64(h.MinHostingMinutes) {
		responses.WriteError(w, http.StatusBadRequest, "ERR_INVALID_RETENTION_PERIOD",
			fmt.Sprintf("The retention period must be >= %d minutes", h.MinHostingMinutes))
		return
	}
	if req.FileSize > 11_000_000_000 {
		responses.WriteError(w, http.StatusBadRequest, "ERR_INVALID_SIZE", "Max supported file size is 11000000000 bytes.")
		return
	}

	amount, err := h.Calculator.GetPrice(req.FileSize, req.RetentionPeriod)
	if err != nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_INTERNAL_UPLOAD", "An internal error occurred while handling upload.")
		return
	}

	objectIdentifier := toBase58(randomBytes(16))
	expiryTime := (req.RetentionPeriod * 60) + time.Now().Unix()
	customTime := time.Unix(expiryTime+300, 0).UTC().Format(time.RFC3339)

	uploaderKey := identityKey.ToDERHex()
	queryStr := fmt.Sprintf("fileSize=%d&objectID=%s&expiry=%s&uploader=%s",
		req.FileSize, objectIdentifier, customTime, uploaderKey)

	// Create HMAC using the wallet to secure the upload URL
	hmac := ""
	wallet := h.WalletProvider.GetWallet()
	if wallet == nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized.")
		return
	}
	hmacResult, hmacErr := wallet.CreateHMAC(r.Context(), sdkWallet.CreateHMACArgs{
		EncryptionArgs: sdkWallet.EncryptionArgs{
			ProtocolID: sdkWallet.Protocol{
				SecurityLevel: sdkWallet.SecurityLevelEveryAppAndCounterparty,
				Protocol:      "uhrp file hosting",
			},
			KeyID:        objectIdentifier,
			Counterparty: sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeSelf},
		},
		Data: []byte(queryStr),
	}, "")
	if hmacErr != nil {
		responses.WriteError(w, http.StatusInternalServerError, "ERR_HMAC", "Failed to create HMAC.")
		return
	}
	hmac = hex.EncodeToString(hmacResult.HMAC[:])

	scheme := "https://"
	if strings.HasPrefix(h.HostingDomain, "localhost") {
		scheme = "http://"
	}
	uploadURL := fmt.Sprintf("%s%s/put?%s&hmac=%s", scheme, h.HostingDomain, queryStr, hmac)

	responses.WriteJSON(w, http.StatusOK, uploadResponse{
		Status:          "success",
		UploadURL:       uploadURL,
		RequiredHeaders: map[string]string{},
		Amount:          amount,
		Description:     "File can now be uploaded.",
	})
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

// toBase58 encodes bytes to base58 (Bitcoin alphabet).
func toBase58(data []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	if len(data) == 0 {
		return ""
	}

	// Count leading zeros
	zeros := 0
	for _, b := range data {
		if b != 0 {
			break
		}
		zeros++
	}

	// Convert to big integer and encode
	size := len(data)*138/100 + 1
	buf := make([]byte, size)
	for _, b := range data {
		carry := int(b)
		for i := size - 1; i >= 0; i-- {
			carry += 256 * int(buf[i])
			buf[i] = byte(carry % 58)
			carry /= 58
		}
	}

	// Skip leading zeros in buf
	i := 0
	for i < size && buf[i] == 0 {
		i++
	}

	// Build result
	result := make([]byte, zeros+size-i)
	for j := 0; j < zeros; j++ {
		result[j] = '1'
	}
	for j := zeros; i < size; i, j = i+1, j+1 {
		result[j] = alphabet[buf[i]]
	}
	return string(result)
}
