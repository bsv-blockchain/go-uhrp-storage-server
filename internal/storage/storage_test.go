package storage

import (
	"os"
	"testing"
)

func TestFileStore_WriteAndRead(t *testing.T) {
	dir, err := os.MkdirTemp("", "storage-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	store := NewFileStore(dir)

	if store.Exists("testfile") {
		t.Error("file should not exist yet")
	}

	data := []byte("hello world")
	if err := store.Write("testfile", data); err != nil {
		t.Fatalf("write error: %v", err)
	}

	if !store.Exists("testfile") {
		t.Error("file should exist after write")
	}

	read, err := store.Read("testfile")
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(read) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(read))
	}

	size, err := store.Size("testfile")
	if err != nil {
		t.Fatalf("size error: %v", err)
	}
	if size != 11 {
		t.Errorf("expected size 11, got %d", size)
	}
}
