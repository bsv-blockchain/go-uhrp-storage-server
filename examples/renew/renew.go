// Package main uploads a file and then renews its retention against a running UHRP storage server.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/bsv-blockchain/go-sdk/storage"

	"github.com/bsv-blockchain/go-uhrp-storage-server/examples/internal/exampleclient"
)

const (
	retentionMinutes  = 60
	additionalMinutes = 120
)

// errNotPublished is returned when the server accepted the upload but did not publish it.
var errNotPublished = errors.New("file was not published")

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	uploader, cleanup, err := exampleclient.NewUploader()
	if err != nil {
		return err
	}
	defer cleanup()

	file := storage.UploadableFile{
		Data: []byte("TEST123 at: " + time.Now().String()),
		Type: "text/plain",
	}

	result, err := uploader.PublishFile(ctx, file, retentionMinutes)
	if err != nil {
		return fmt.Errorf("publish file: %w", err)
	}

	if !result.Published {
		return fmt.Errorf("%w: %+v", errNotPublished, result)
	}

	log.Printf("File published successfully! UHRP URL: %s", result.UhrpURL)

	log.Println("--- Retrieving previous metadata before renewal ---")
	printMetadata(ctx, uploader, result.UhrpURL)

	log.Printf("--- Renewing file for additional %d minutes ---", additionalMinutes)

	renewResult, err := uploader.RenewFile(ctx, result.UhrpURL, additionalMinutes)
	if err != nil {
		return fmt.Errorf("renew file: %w", err)
	}

	log.Println("Renew Successful!")
	log.Printf("Status: %s", renewResult.Status)
	log.Printf("Previous Expiry: %d", renewResult.PrevExpiryTime)
	log.Printf("New Expiry: %d", renewResult.NewExpiryTime)
	log.Printf("Amount Charged: %d satoshis", renewResult.Amount)

	log.Println("--- Retrieving updated metadata ---")
	printMetadata(ctx, uploader, result.UhrpURL)

	return nil
}

// printMetadata looks up a published file's metadata.
func printMetadata(ctx context.Context, uploader *storage.Uploader, uhrpURL string) {
	fileMetadata, err := uploader.FindFile(ctx, uhrpURL)
	if err != nil {
		log.Printf("Failed to find file metadata: %v", err)
		return
	}

	log.Printf("File Metadata: %+v", fileMetadata)
}
