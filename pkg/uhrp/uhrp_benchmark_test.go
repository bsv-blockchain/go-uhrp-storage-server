package uhrp_test

import (
	"testing"

	"github.com/bsv-blockchain/go-uhrp-storage-server/pkg/uhrp"
)

func BenchmarkHashData(b *testing.B) {
	data := make([]byte, 1<<10)

	b.ReportAllocs()

	for b.Loop() {
		_ = uhrp.HashData(data)
	}
}

func BenchmarkGetURLForHash(b *testing.B) {
	hash := uhrp.HashData([]byte("hello world"))

	b.ReportAllocs()

	for b.Loop() {
		_ = uhrp.GetURLForHash(hash)
	}
}

func BenchmarkGetHashFromURL(b *testing.B) {
	url := "uhrp://" + helloWorldURL

	b.ReportAllocs()

	for b.Loop() {
		_, _ = uhrp.GetHashFromURL(url)
	}
}
