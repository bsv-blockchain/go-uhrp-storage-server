// Package main quotes, uploads, finds and lists a file against a running UHRP storage server.
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

	"github.com/bsv-blockchain/go-sdk/storage"

	"github.com/bsv-blockchain/go-uhrp-storage-server/examples/internal/exampleclient"
)

const retentionMinutes = 60

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

	fileContent := []byte("TEST123 at: " + time.Now().String())

	log.Printf("--- Quoting file (size: %d bytes) ---", len(fileContent))

	if err = printQuote(ctx, len(fileContent)); err != nil {
		log.Printf("Failed to fetch quote: %v", err)
	}

	log.Println("--- Uploading file ---")

	result, err := uploader.PublishFile(ctx, storage.UploadableFile{Data: fileContent, Type: "text/plain"}, retentionMinutes)
	if err != nil {
		log.Printf("Publish failed (likely due to missing wallet payment methods): %v", err)
	} else if result.Published {
		log.Printf("File published successfully! UHRP URL: %s", result.UhrpURL)
		printMetadata(ctx, uploader, result.UhrpURL)
	}

	return printUploads(ctx, uploader)
}

// printQuote asks the server's public /quote endpoint for the price of storing size bytes.
func printQuote(ctx context.Context, size int) error {
	body, err := json.Marshal(map[string]int{
		"fileSize":        size,
		"retentionPeriod": retentionMinutes,
	})
	if err != nil {
		return fmt.Errorf("encode quote request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, exampleclient.ServerURL()+"/quote", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build quote request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send quote request: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	quote, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read quote response: %w", err)
	}

	log.Printf("Quote: %s", quote)

	return nil
}

// printMetadata looks up a published file and prints where it can be downloaded.
func printMetadata(ctx context.Context, uploader *storage.Uploader, uhrpURL string) {
	log.Println("--- Finding file metadata ---")

	fileMetadata, err := uploader.FindFile(ctx, uhrpURL)
	if err != nil {
		log.Printf("Failed to find file metadata: %v", err)
		return
	}

	log.Printf("File Metadata: %+v", fileMetadata)
	log.Println("--- Downloading file content ---")
	log.Printf("You can view it at: %s/cdn/%s", exampleclient.ServerURL(), fileMetadata.Name)
}

// printUploads lists the caller's active uploads.
func printUploads(ctx context.Context, uploader *storage.Uploader) error {
	log.Println("--- Listing user uploads ---")

	uploads, err := uploader.ListUploads(ctx)
	if err != nil {
		return fmt.Errorf("list uploads: %w", err)
	}

	uploadsList, ok := exampleclient.ParseUploads(uploads)
	if !ok {
		log.Printf("Found uploads, but could not parse list: %v", uploads)
		return nil
	}

	log.Printf("Found %d uploads for this user identity:", len(uploadsList))

	for _, u := range uploadsList {
		log.Printf("%+v", u)
	}

	return nil
}
