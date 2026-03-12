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

	// 3. Upload File
	result := storage.UploadFileResult{
		Published: true,
		UhrpURL:   "uhrp://236452WUcLCMgzS5L4Cw6w4krHi4h2xVoRNYQXy61bWpVJVXdj",
	}

	if result.Published {
		fmt.Printf("File published successfully! UHRP URL: %s\n", result.UhrpURL)

		// 4. Find file by URL
		fmt.Println("\n--- Retrieving previous metadata before renewal ---")
		fileMetadata, err := uploader.FindFile(context.Background(), result.UhrpURL)
		if err != nil {
			fmt.Printf("Failed to find file metadata: %v\n", err)
		} else {
			fmt.Printf("File Metadata: %+v\n", fileMetadata)
		}

		// 5. Renew File
		fmt.Println("\n--- Renewing file for additional 120 minutes ---")
		renewResult, err := uploader.RenewFile(context.Background(), result.UhrpURL, 120)
		if err != nil {
			log.Fatalf("Renewal failed: %v", err)
		}

		fmt.Printf("Renew Successful!\n")
		fmt.Printf("Status: %s\n", renewResult.Status)
		fmt.Printf("Previous Expiry: %d\n", renewResult.PrevExpiryTime)
		fmt.Printf("New Expiry: %d\n", renewResult.NewExpiryTime)
		fmt.Printf("Amount Charged: %d satoshis\n", renewResult.Amount)

		// 6. Find file by URL
		fmt.Println("\n--- Retrieving updated metadata ---")
		updatedFileMetadata, err := uploader.FindFile(context.Background(), result.UhrpURL)
		if err != nil {
			fmt.Printf("Failed to find updated file metadata: %v\n", err)
		} else {
			fmt.Printf("Updated File Metadata: %+v\n", updatedFileMetadata)
		}

	}
}
