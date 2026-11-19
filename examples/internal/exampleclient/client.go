// Package exampleclient builds the authenticated go-sdk storage client shared by the examples.
package exampleclient

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/storage"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/defs"
	wdkStorage "github.com/bsv-blockchain/go-wallet-toolbox/pkg/storage"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/wallet"
)

// ServerURL returns the UHRP storage server to talk to (UHRP_SERVER_URL, default http://localhost:8080).
func ServerURL() string {
	return envOr("UHRP_SERVER_URL", "http://localhost:8080")
}

// walletStorageURL returns the client wallet's storage server (WALLET_STORAGE_URL, default http://localhost:8100).
func walletStorageURL() string {
	return envOr("WALLET_STORAGE_URL", "http://localhost:8100")
}

// NewUploader creates a go-sdk storage uploader for ServerURL, paying and authenticating with a
// go-wallet-toolbox wallet. The example key is deterministic and for local testing only.
// Call the returned cleanup function when done.
func NewUploader() (*storage.Uploader, func(), error) {
	serverURL := ServerURL()
	log.Printf("Starting Go repo test script against: %s", serverURL)

	privateKey, _ := ec.PrivateKeyFromBytes([]byte("example_xpriv_key"))

	pw, err := sdkWallet.NewCompletedProtoWallet(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("create proto wallet: %w", err)
	}

	wspc, cleanup, err := wdkStorage.NewClient(walletStorageURL(), pw)
	if err != nil {
		return nil, nil, fmt.Errorf("create storage provider client: %w", err)
	}

	userWallet, err := wallet.New(defs.NetworkMainnet, privateKey, wspc)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("create wallet: %w", err)
	}

	log.Printf("User Identity Key (Client): %s", privateKey.PubKey().ToDERHex())

	uploader, err := storage.NewUploader(storage.UploaderConfig{
		StorageURL: serverURL,
		Wallet:     userWallet,
	})
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("create uploader: %w", err)
	}

	return uploader, cleanup, nil
}

// ParseUploads converts the generic value returned by Uploader.ListUploads into typed metadata.
// It reports false when the value does not have the expected shape.
func ParseUploads(uploads any) ([]storage.UploadMetadata, bool) {
	b, err := json.Marshal(uploads)
	if err != nil {
		return nil, false
	}

	var list []storage.UploadMetadata
	if err = json.Unmarshal(b, &list); err != nil {
		return nil, false
	}

	return list, true
}

// envOr returns the environment variable key, or def when it is unset or empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}
