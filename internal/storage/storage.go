package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileStore manages file storage on the local filesystem.
type FileStore struct {
	basePath string
}

// NewFileStore creates a new file store rooted at basePath/cdn/.
func NewFileStore(basePath string) *FileStore {
	cdnPath := filepath.Join(basePath, "cdn")
	_ = os.MkdirAll(cdnPath, 0o755)
	return &FileStore{basePath: basePath}
}

// Exists checks if a file with the given objectID exists.
func (fs *FileStore) Exists(objectID string) bool {
	_, err := os.Stat(filepath.Join(fs.basePath, "cdn", objectID))
	return err == nil
}

// Write writes data to a file named objectID in the cdn directory.
func (fs *FileStore) Write(objectID string, data []byte) error {
	return os.WriteFile(filepath.Join(fs.basePath, "cdn", objectID), data, 0o644)
}

// Read reads a file by objectID.
func (fs *FileStore) Read(objectID string) ([]byte, error) {
	return os.ReadFile(filepath.Join(fs.basePath, "cdn", objectID))
}

// Open opens a file for streaming.
func (fs *FileStore) Open(objectID string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(fs.basePath, "cdn", objectID))
}

// FilePath returns the full path to a CDN file.
func (fs *FileStore) FilePath(objectID string) string {
	return filepath.Join(fs.basePath, "cdn", objectID)
}

// CDNPath returns the CDN directory path.
func (fs *FileStore) CDNPath() string {
	return filepath.Join(fs.basePath, "cdn")
}

// BasePath returns the base public path.
func (fs *FileStore) BasePath() string {
	return fs.basePath
}

// Size returns the file size in bytes.
func (fs *FileStore) Size(objectID string) (int64, error) {
	info, err := os.Stat(filepath.Join(fs.basePath, "cdn", objectID))
	if err != nil {
		return 0, fmt.Errorf("stat: %w", err)
	}
	return info.Size(), nil
}
