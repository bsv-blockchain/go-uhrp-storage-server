package uhrp_test

import (
	"fmt"

	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/uhrp"
)

// ExampleGetURLForHash shows how a file's content maps to its UHRP URL and back.
func ExampleGetURLForHash() {
	hash := uhrp.HashData([]byte("hello world"))
	url := uhrp.GetURLForHash(hash)
	fmt.Println(url)

	decoded, err := uhrp.GetHashFromURL("uhrp://" + url)
	fmt.Printf("%x %v\n", decoded, err)
	// Output:
	// XUUJPXCDnKd2iwUGSFTfTFyhQyxXjvg1ArF7byo58k7cyP5PYT6z
	// b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9 <nil>
}
