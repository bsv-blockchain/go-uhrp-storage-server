// Package main lists the caller's active uploads on a running UHRP storage server.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/bsv-blockchain/go-uhrp-storage-server/examples/internal/exampleclient"
)

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

	log.Println("--- Listing user uploads (via uploader.ListUploads) ---")

	uploads, err := uploader.ListUploads(ctx)
	if err != nil {
		return fmt.Errorf("list uploads: %w", err)
	}

	uploadsList, ok := exampleclient.ParseUploads(uploads)
	if !ok {
		log.Printf("Parsed generic fallback output: %v", uploads)
		return nil
	}

	log.Printf("Found %d non-expired uploads for this user identity:", len(uploadsList))

	for i, u := range uploadsList {
		log.Printf("[%d] URL: %s | Expiry: %d", i+1, u.UhrpURL, u.ExpiryTime)
	}

	return nil
}
