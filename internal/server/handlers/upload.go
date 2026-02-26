package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bsv-blockchain/go-bsv-middleware/pkg/middleware"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/pricing"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
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

func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	identityKey, err := middleware.ShouldGetIdentity(r.Context())
	if err != nil || isUnknown(identityKey) {
		writeError(w, http.StatusBadRequest, "ERR_MISSING_IDENTITY_KEY", "Missing authfetch identityKey.")
		return
	}

	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_BODY", "Invalid request body.")
		return
	}

	if req.FileSize <= 0 {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_SIZE", "The file size must be a positive integer.")
		return
	}
	if req.RetentionPeriod <= 0 {
		writeError(w, http.StatusBadRequest, "ERR_NO_RETENTION_PERIOD", "You must specify the number of minutes to host the file.")
		return
	}
	if req.RetentionPeriod < int64(h.MinHostingMinutes) {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_RETENTION_PERIOD",
			fmt.Sprintf("The retention period must be >= %d minutes", h.MinHostingMinutes))
		return
	}
	if req.FileSize > 11_000_000_000 {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_SIZE", "Max supported file size is 11000000000 bytes.")
		return
	}

	amount, err := h.Calculator.GetPrice(req.FileSize, req.RetentionPeriod)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL_UPLOAD", "An internal error occurred while handling upload.")
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
		writeError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized.")
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
		writeError(w, http.StatusInternalServerError, "ERR_HMAC", "Failed to create HMAC.")
		return
	}
	hmac = hex.EncodeToString(hmacResult.HMAC[:])

	scheme := "https://"
	if strings.HasPrefix(h.HostingDomain, "localhost") {
		scheme = "http://"
	}
	uploadURL := fmt.Sprintf("%s%s/put?%s&hmac=%s", scheme, h.HostingDomain, queryStr, hmac)

	writeJSON(w, http.StatusOK, uploadResponse{
		Status:          "success",
		UploadURL:       uploadURL,
		RequiredHeaders: map[string]string{},
		Amount:          amount,
		Description:     "File can now be uploaded.",
	})
}

func isUnknown(key *ec.PublicKey) bool {
	return key == nil || middleware.IsUnknownIdentity(key)
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
