package middlewares

import (
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

const (
	// cdnPrefix is the URL path prefix uploaded files are served under.
	cdnPrefix = "/cdn/"

	// sniffLen is how many leading bytes http.DetectContentType inspects.
	sniffLen = 512

	defaultMimeType = "application/octet-stream"
)

// MimeTypeMiddleware serves uploaded files under /cdn/ with a Content-Type detected from
// the file's extension, falling back to its leading bytes. Anything that is not a regular
// file in Files (missing objects, directories, invalid paths) is passed to the next handler.
type MimeTypeMiddleware struct {
	// Files is the cdn directory, e.g. storage.FileStore.FS(); it confines every lookup to that directory.
	Files fs.FS

	extensionTypes map[string]string
}

// NewMimeTypeMiddleware creates a middleware serving files from the given cdn file system.
func NewMimeTypeMiddleware(files fs.FS) *MimeTypeMiddleware {
	return &MimeTypeMiddleware{
		Files:          files,
		extensionTypes: extensionMimeTypes(),
	}
}

// extensionMimeTypes maps lower-case file extensions to MIME types.
func extensionMimeTypes() map[string]string {
	return map[string]string{
		".html": "text/html", ".htm": "text/html", ".css": "text/css",
		".js": "application/javascript", ".json": "application/json",
		".xml": "application/xml", ".txt": "text/plain", ".md": "text/markdown",
		".pdf": "application/pdf", ".zip": "application/zip",
		".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
		".gif": "image/gif", ".svg": "image/svg+xml", ".webp": "image/webp",
		".ico": "image/x-icon", ".mp4": "video/mp4", ".mp3": "audio/mpeg",
		".wav": "audio/wav", ".woff": "font/woff", ".woff2": "font/woff2",
		".ttf": "font/ttf", ".otf": "font/otf",
	}
}

// Handle intercepts CDN requests and serves the file with the correct MIME type.
func (m *MimeTypeMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		objectID, ok := strings.CutPrefix(r.URL.Path, cdnPrefix)
		if !ok || !m.isFile(objectID) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", m.detectMimeType(objectID))
		//nolint:gosec // G703: isFile accepted objectID only as an fs.ValidPath to a regular file, and Files is os.Root-confined
		http.ServeFileFS(w, r, m.Files, objectID)
	})
}

// isFile reports whether objectID names a regular file in Files.
func (m *MimeTypeMiddleware) isFile(objectID string) bool {
	if m.Files == nil || objectID == "" || !fs.ValidPath(objectID) {
		return false
	}

	info, err := fs.Stat(m.Files, objectID)

	return err == nil && info.Mode().IsRegular()
}

func (m *MimeTypeMiddleware) detectMimeType(objectID string) string {
	// Try extension first
	if mt, ok := m.extensionTypes[strings.ToLower(path.Ext(objectID))]; ok {
		return mt
	}

	// Try magic bytes
	f, err := m.Files.Open(objectID)
	if err != nil {
		return defaultMimeType
	}

	defer func() { _ = f.Close() }()

	buf := make([]byte, sniffLen)

	n, _ := io.ReadFull(f, buf) // a short file is fine; only an empty one has nothing to sniff
	if n == 0 {
		return defaultMimeType
	}

	return http.DetectContentType(buf[:n])
}
