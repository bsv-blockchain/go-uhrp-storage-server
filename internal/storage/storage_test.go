package storage_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/storage"
)

// newStore returns a store rooted in a fresh temporary public directory.
func newStore(t *testing.T) (*storage.FileStore, string) {
	t.Helper()

	dir := t.TempDir()

	store, err := storage.NewFileStore(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	return store, dir
}

func TestFileStoreWriteAndRead(t *testing.T) {
	store, dir := newStore(t)

	assert.Equal(t, filepath.Join(dir, "cdn"), store.CDNPath())
	assert.False(t, store.Exists("testfile"), "file should not exist yet")

	require.NoError(t, store.Write("testfile", []byte("hello world")))
	assert.True(t, store.Exists("testfile"), "file should exist after write")

	read, err := store.Read("testfile")
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(read))

	size, err := store.Size("testfile")
	require.NoError(t, err)
	assert.Equal(t, int64(11), size)

	onDisk, err := fs.ReadFile(os.DirFS(dir), "cdn/testfile")
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(onDisk))
}

func TestFileStoreFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits do not apply on Windows")
	}

	store, dir := newStore(t)
	require.NoError(t, store.Write("private", []byte("x")))

	info, err := os.Stat(filepath.Join(dir, "cdn", "private"))
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0o600), info.Mode().Perm())
}

func TestFileStoreSizeMissingFile(t *testing.T) {
	store, _ := newStore(t)

	_, err := store.Size("missing")
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFileStoreConfinesObjectIDs(t *testing.T) {
	store, dir := newStore(t)

	outside := filepath.Join(dir, "outside.txt")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600))

	for _, objectID := range []string{"../outside.txt", "../../etc/passwd", outside} {
		t.Run(objectID, func(t *testing.T) {
			assert.False(t, store.Exists(objectID))

			_, err := store.Read(objectID)
			require.Error(t, err)

			require.Error(t, store.Write(objectID, []byte("overwrite")))
		})
	}

	data, err := fs.ReadFile(os.DirFS(dir), "outside.txt")
	require.NoError(t, err)
	assert.Equal(t, "secret", string(data), "a file outside the cdn directory must be untouched")
}

func TestFileStoreFS(t *testing.T) {
	store, _ := newStore(t)
	require.NoError(t, store.Write("served", []byte("content")))

	data, err := fs.ReadFile(store.FS(), "served")
	require.NoError(t, err)
	assert.Equal(t, "content", string(data))

	_, err = fs.ReadFile(store.FS(), "../outside")
	require.Error(t, err)
}

func TestNewFileStoreFailsWhenDirectoryCannotBeCreated(t *testing.T) {
	dir := t.TempDir()

	blocker := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o600))

	_, err := storage.NewFileStore(blocker) // <file>/cdn cannot be created under a regular file
	require.Error(t, err)
}
