package uhrp

import (
	"encoding/hex"
	"testing"
)

func TestHashData(t *testing.T) {
	data := []byte("hello world")
	hash := HashData(data)
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hex.EncodeToString(hash) != expected {
		t.Errorf("expected %s, got %s", expected, hex.EncodeToString(hash))
	}
}

func TestGetURLForHash(t *testing.T) {
	hash, _ := hex.DecodeString("b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")
	url := GetURLForHash(hash)
	expected := "uhrp://b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if url != expected {
		t.Errorf("expected %s, got %s", expected, url)
	}
}

func TestGetHashFromURL(t *testing.T) {
	url := "uhrp://b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	hash, err := GetHashFromURL(url)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hex.EncodeToString(hash) != expected {
		t.Errorf("expected %s, got %s", expected, hex.EncodeToString(hash))
	}
}

func TestGetHashFromURL_Invalid(t *testing.T) {
	_, err := GetHashFromURL("invalid")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}
