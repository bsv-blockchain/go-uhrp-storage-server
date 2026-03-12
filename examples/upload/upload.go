package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

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

	// 3. Example File Data
	fileContent := []byte("TEST123 at: " + time.Now().String())
	retentionMinutes := 60

	fmt.Printf("\n--- Quoting file (size: %d bytes) ---\n", len(fileContent))
	quoteReqBody, _ := json.Marshal(map[string]interface{}{
		"fileSize":        len(fileContent),
		"retentionPeriod": retentionMinutes,
	})

	resp, err := http.Post(serverURL+"/quote", "application/json", bytes.NewBuffer(quoteReqBody))
	if err != nil {
		log.Printf("Failed to fetch quote: %v", err)
	} else {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Quote: %s\n", string(body))
	}

	fmt.Println("\n--- Uploading file ---")
	file := storage.UploadableFile{
		Data: fileContent,
		Type: "text/plain",
	}

	result, err := uploader.PublishFile(context.Background(), file, retentionMinutes)
	if err != nil {
		fmt.Printf("Publish failed (likely due to missing wallet payment methods): %v\n", err)
	} else if result.Published {
		fmt.Printf("\nFile published successfully! UHRP URL: %s\n", result.UhrpURL)

		fmt.Println("\n--- Finding file metadata ---")
		fileMetadata, err := uploader.FindFile(context.Background(), result.UhrpURL)
		if err != nil {
			fmt.Printf("Failed to find file metadata: %v\n", err)
		} else {
			fmt.Printf("File Metadata: %+v\n", fileMetadata)
			fmt.Println("\n--- Downloading file content ---")
			fmt.Printf("You can view it at: %s/cdn/%s\n", serverURL, fileMetadata.Name)
		}
	}

	fmt.Println("\n--- Listing user uploads ---")
	uploads, err := uploader.ListUploads(context.Background())
	if err != nil {
		fmt.Printf("List uploads failed: %v\n", err)
	} else {
		b, err := json.Marshal(uploads)
		if err == nil {
			var uploadsList []storage.UploadMetadata
			err = json.Unmarshal(b, &uploadsList)
			if err == nil {
				fmt.Printf("Found %d uploads for this user identity:\n", len(uploadsList))
				for _, u := range uploadsList {
					fmt.Printf("%+v\n", u)
				}
				return // Success
			}
		}

		fmt.Printf("Found uploads, but could not parse list: %v\n", uploads)
	}
}
