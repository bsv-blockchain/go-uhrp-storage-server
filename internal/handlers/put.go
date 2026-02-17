package handlers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/bsv-blockchain/go-sdk/script"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/uhrp"
	walletpkg "github.com/bsv-blockchain/go-uhrp-storage-server/internal/wallet"
)

// PutHandler handles PUT /put requests for file upload.
type PutHandler struct {
	Store          *storage.FileStore
	WalletProvider *walletpkg.Provider
	HostingDomain  string
}

func (h *PutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	uploader := q.Get("uploader")
	objectID := q.Get("objectID")
	fileSizeStr := q.Get("fileSize")
	expiry := q.Get("expiry")
	hmac := q.Get("hmac")

	if objectID == "" || fileSizeStr == "" || expiry == "" || hmac == "" {
		writeError(w, http.StatusBadRequest, "", "Missing required query parameters")
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "", "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Verify size
	fileSize, _ := strconv.ParseInt(fileSizeStr, 10, 64)
	if fileSize != int64(len(body)) {
		writeError(w, http.StatusBadRequest, "", "Size mismatch")
		return
	}

	// Verify HMAC using wallet
	str := fmt.Sprintf("fileSize=%s&objectID=%s&expiry=%s&uploader=%s", fileSizeStr, objectID, expiry, uploader)
	wallet := h.WalletProvider.GetWallet()
	if wallet == nil {
		writeError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized")
		return
	}

	hmacBytes, err := hex.DecodeString(hmac)
	if err != nil || len(hmacBytes) != 32 {
		writeError(w, http.StatusBadRequest, "ERR_INVALID_HMAC", "Invalid HMAC format")
		return
	}
	var hmacArr [32]byte
	copy(hmacArr[:], hmacBytes)

	verifyResult, err := wallet.VerifyHMAC(r.Context(), sdkWallet.VerifyHMACArgs{
		EncryptionArgs: sdkWallet.EncryptionArgs{
			ProtocolID: sdkWallet.Protocol{
				SecurityLevel: sdkWallet.SecurityLevelEveryAppAndCounterparty,
				Protocol:      "uhrp file hosting",
			},
			KeyID:        objectID,
			Counterparty: sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeSelf},
		},
		Data: []byte(str),
		HMAC: hmacArr,
	}, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_HMAC_VERIFY", "HMAC verification failed")
		return
	}
	if !verifyResult.Valid {
		writeError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Invalid HMAC")
		return
	}

	// Check if file already exists
	if h.Store.Exists(objectID) {
		writeError(w, http.StatusBadRequest, "", "File exists")
		return
	}

	// Write file
	if err := h.Store.Write(objectID, body); err != nil {
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "Failed to write file")
		return
	}

	// Create UHRP advertisement
	if strings.HasPrefix(h.HostingDomain, "localhost") {
		log.Println("Not advertising, localhost")
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL",
			"An internal error occurred while processing the request.")
		return
	}

	hash := uhrp.HashData(body)
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Build a PushDrop-style locking script for the UHRP advertisement
	// Fields: [uhrpURL, hostingDomain, expiryTime, contentType, contentLength, objectID, uploaderKey]
	uhrpURL := uhrp.GetURLForHash(hash)
	customInstructions := buildCustomInstructions(uhrpURL, h.HostingDomain, expiry, contentType, fileSizeStr, objectID, uploader)

	lockingScript, err := buildPushDropScript(uhrpURL, h.HostingDomain, expiry, contentType, fileSizeStr, objectID, uploader)
	if err != nil {
		log.Printf("Failed to build advertisement script: %v", err)
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "Failed to create advertisement")
		return
	}

	// Create the advertisement transaction via wallet
	_, err = wallet.CreateAction(r.Context(), sdkWallet.CreateActionArgs{
		Description: fmt.Sprintf("UHRP advertisement for %s", uhrpURL),
		Outputs: []sdkWallet.CreateActionOutput{
			{
				LockingScript:      lockingScript,
				Satoshis:           1,
				OutputDescription:  "UHRP advertisement token",
				Basket:             "uhrp advertisements",
				CustomInstructions: customInstructions,
				Tags:               []string{"uhrp-ad", fmt.Sprintf("uploader-%s", uploader)},
			},
		},
		Labels: []string{"uhrp-advertisement"},
	}, "")
	if err != nil {
		log.Printf("Failed to create UHRP advertisement: %v", err)
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "Failed to broadcast advertisement")
		return
	}

	log.Printf("File uploaded: objectID=%s, size=%d, uploader=%s", objectID, len(body), uploader)

	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// buildCustomInstructions creates a JSON string of the advertisement metadata
// stored as customInstructions on the output, for later retrieval by ListOutputs.
func buildCustomInstructions(uhrpURL, hostingDomain, expiry, contentType, fileSize, objectID, uploader string) string {
	data := map[string]string{
		"uhrpURL":       uhrpURL,
		"hostingDomain": hostingDomain,
		"expiryTime":    expiry,
		"contentType":   contentType,
		"fileSize":      fileSize,
		"objectID":      objectID,
		"uploader":      uploader,
	}
	b, _ := json.Marshal(data)
	return string(b)
}

// buildPushDropScript builds an OP_RETURN-based script with push data fields.
func buildPushDropScript(uhrpURL, hostingDomain, expiry, contentType, fileSize, objectID, uploader string) ([]byte, error) {
	s := &script.Script{}
	if err := s.AppendOpcodes(script.OpFALSE, script.OpRETURN); err != nil {
		return nil, err
	}
	for _, field := range []string{uhrpURL, hostingDomain, expiry, contentType, fileSize, objectID, uploader} {
		if err := s.AppendPushDataString(field); err != nil {
			return nil, err
		}
	}
	return *s, nil
}
