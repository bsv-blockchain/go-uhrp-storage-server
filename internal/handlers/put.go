package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

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
	// In a full implementation, we'd use wallet.VerifyHmac here.
	// The HMAC is over: fileSize=X&objectID=Y&expiry=Z&uploader=W
	str := fmt.Sprintf("fileSize=%s&objectID=%s&expiry=%s&uploader=%s", fileSizeStr, objectID, expiry, uploader)
	_ = str // Used for HMAC verification
	_ = hmac

	// TODO: Implement HMAC verification with wallet when go-sdk supports it
	// wallet := h.WalletProvider.GetWallet()
	// valid := wallet.VerifyHmac(...)

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

	// TODO: Create UHRP advertisement transaction using PushDrop and broadcast via SHIP
	// This requires full wallet integration with go-sdk PushDrop and SHIPBroadcaster
	_ = hash
	_ = contentType

	log.Printf("File uploaded: objectID=%s, size=%d, uploader=%s", objectID, len(body), uploader)

	writeJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
