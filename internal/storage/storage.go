// Package storage keeps uploaded files on the local filesystem under <public>/cdn.
package storage

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	// cdnDirName is the directory under the public dir that holds uploaded files (served at /cdn/).
	cdnDirName = "cdn"

	dirPerm  = 0o750
	filePerm = 0o600
)

// FileStore manages file storage on the local filesystem.
//
// Every operation goes through an os.Root opened on the cdn directory, so an object ID
// can never resolve to a path outside it (no "..", absolute paths or escaping symlinks).
type FileStore struct {
	cdnPath string
	root    *os.Root
}

// NewFileStore creates the basePath/cdn directory if needed and returns a store rooted there.
// Call Close when the store is no longer needed.
func NewFileStore(basePath string) (*FileStore, error) {
	cdnPath := filepath.Join(basePath, cdnDirName)
	if err := os.MkdirAll(cdnPath, dirPerm); err != nil {
		return nil, fmt.Errorf("create cdn directory: %w", err)
	}

	root, err := os.OpenRoot(cdnPath)
	if err != nil {
		return nil, fmt.Errorf("open cdn directory: %w", err)
	}

	return &FileStore{cdnPath: cdnPath, root: root}, nil
}

// Close releases the store's handle on the cdn directory.
func (s *FileStore) Close() error {
	return s.root.Close()
}

// Exists checks if a file with the given objectID exists.
func (s *FileStore) Exists(objectID string) bool {
	_, err := s.root.Stat(objectID)
	return err == nil
}

// Write writes data to a file named objectID in the cdn directory.
func (s *FileStore) Write(objectID string, data []byte) error {
	return s.root.WriteFile(objectID, data, filePerm)
}

// Read reads a file by objectID.
func (s *FileStore) Read(objectID string) ([]byte, error) {
	return s.root.ReadFile(objectID)
}

// Size returns the file size in bytes.
func (s *FileStore) Size(objectID string) (int64, error) {
	info, err := s.root.Stat(objectID)
	if err != nil {
		return 0, fmt.Errorf("stat: %w", err)
	}

	return info.Size(), nil
}

// FS returns a read-only file system of the cdn directory, confined to it like the store.
func (s *FileStore) FS() fs.FS {
	return s.root.FS()
}

// CDNPath returns the CDN directory path.
func (s *FileStore) CDNPath() string {
	return s.cdnPath
}
