package handlers

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/uhrp"
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

	wallet := h.WalletProvider.GetWallet()
	if wallet == nil {
		writeError(w, http.StatusInternalServerError, "ERR_NO_WALLET", "Wallet not initialized")
		return
	}

	if err := walletpkg.VerifyUploaderHMAC(r.Context(), wallet, fileSizeStr, objectID, expiry, uploader, hmac); err != nil {
		if strings.Contains(err.Error(), "invalid HMAC") {
			writeError(w, http.StatusUnauthorized, "ERR_UNAUTHORIZED", "Invalid HMAC")
		} else {
			writeError(w, http.StatusInternalServerError, "ERR_HMAC_VERIFY", "HMAC verification failed")
		}
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

	uhrpURL := uhrp.GetURLForHash(hash)

	expiryInt, _ := strconv.ParseInt(expiry, 10, 64)
	err = walletpkg.CreateAdvertisement(r.Context(), wallet, walletpkg.CreateAdParams{
		UhrpURL:       uhrpURL,
		HostingDomain: h.HostingDomain,
		ExpirySecs:    expiryInt,
		ContentType:   contentType,
		FileSize:      fileSize,
		ObjectID:      objectID,
		Uploader:      uploader,
	})

	if err != nil {
		log.Printf("Failed to create UHRP advertisement: %v", err)
		writeError(w, http.StatusInternalServerError, "ERR_INTERNAL", "Failed to broadcast advertisement")
		return
	}

	log.Printf("File uploaded: objectID=%s, size=%d, uploader=%s", objectID, len(body), uploader)

	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
