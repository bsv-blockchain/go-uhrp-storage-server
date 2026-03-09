package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/storage"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/defs"
	wdkStorage "github.com/bsv-blockchain/go-wallet-toolbox/pkg/storage"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/wallet"
)

func main() {
	serverURL := "http://localhost:8080"
	storageURL := "http://localhost:8100"
	fmt.Printf("Starting Go repo test script against: %s\n", serverURL)

	// 1. Setup Auth - create user wallet
	privateKey, _ := ec.PrivateKeyFromBytes([]byte("example_xpriv_key"))

	pw, err := sdkWallet.NewCompletedProtoWallet(privateKey)
	if err != nil {
		log.Fatalf("failed to create proto wallet: %v", err)
	}
	wspc, cleanup, err := wdkStorage.NewClient(storageURL, pw)
	if err != nil {
		log.Fatalf("failed to create storage provider client: %v", err)
	}

	defer cleanup()

	aliceWallet, err := wallet.New(defs.NetworkMainnet, privateKey, wspc)
	if err != nil {
		log.Fatalf("failed: %v", err)
	}

	fmt.Printf("User Identity Key (Client): %s\n", privateKey.PubKey().ToDERHex())

	// 2. Initialize the StorageUploader
	uploader, err := storage.NewUploader(storage.UploaderConfig{
		StorageURL: serverURL,
		Wallet:     aliceWallet,
	})
	if err != nil {
		log.Fatalf("Failed to create uploader: %v", err)
	}

	fmt.Println("\n--- Listing user uploads (via uploader.ListUploads) ---")
	uploads, err := uploader.ListUploads(context.Background())
	if err != nil {
		log.Fatalf("List uploads failed: %v", err)
	}

	// 3. Marshall and correctly deserialize the generic interface response from the go-sdk
	b, err := json.Marshal(uploads)
	if err == nil {
		var uploadsList []storage.UploadMetadata
		err = json.Unmarshal(b, &uploadsList)
		if err == nil {
			fmt.Printf("Found %d non-expired uploads for this user identity:\n", len(uploadsList))
			for i, u := range uploadsList {
				fmt.Printf("[%d] URL: %s | Expiry: %d\n", i+1, u.UhrpURL, u.ExpiryTime)
			}
			return
		}
	}

	fmt.Printf("Parsed generic fallback output: %v\n", uploads)
}
