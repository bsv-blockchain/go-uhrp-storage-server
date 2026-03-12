package main

import (
	"context"
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

	// 3. Find file by URL
	fileURL := "uhrp://xtDGjhQW8HYVjT3n1RFNMzojq1uDVdGYdaotUiLrYrhKzmVij"

	fileMetadata, err := uploader.FindFile(context.Background(), fileURL)
	if err != nil {
		fmt.Printf("Failed to find file metadata: %v\n", err)
	} else {
		fmt.Printf("File Metadata: %+v\n", fileMetadata)
		fmt.Println("\n--- Downloading file content ---")
		fmt.Printf("You can view it at: %s/cdn/%s\n", serverURL, fileMetadata.Name)
	}
}
