// Package main looks up a file's metadata by UHRP URL on a running UHRP storage server.
//
// Usage: go run ./examples/find_file [uhrp-url]
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bsv-blockchain/go-uhrp-storage-server/examples/internal/exampleclient"
)

// defaultFileURL is looked up when no UHRP URL is given on the command line.
const defaultFileURL = "uhrp://xtDGjhQW8HYVjT3n1RFNMzojq1uDVdGYdaotUiLrYrhKzmVij"

func main() {
	fileURL := defaultFileURL
	if len(os.Args) > 1 {
		fileURL = os.Args[1]
	}

	if err := run(context.Background(), fileURL); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, fileURL string) error {
	uploader, cleanup, err := exampleclient.NewUploader()
	if err != nil {
		return err
	}
	defer cleanup()

	fileMetadata, err := uploader.FindFile(ctx, fileURL)
	if err != nil {
		return fmt.Errorf("find file metadata: %w", err)
	}

	// Server-supplied strings are printed quoted so they cannot forge log lines.
	log.Printf("File Metadata: name=%q size=%q mimeType=%q expiryTime=%d",
		fileMetadata.Name, fileMetadata.Size, fileMetadata.MimeType, fileMetadata.ExpiryTime)
	log.Println("--- Downloading file content ---")
	log.Printf("You can view it at: %q", exampleclient.ServerURL()+"/cdn/"+fileMetadata.Name)

	return nil
}
