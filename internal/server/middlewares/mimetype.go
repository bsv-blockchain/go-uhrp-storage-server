package middlewares

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MimeTypeMiddleware sets correct Content-Type for CDN files.
type MimeTypeMiddleware struct {
	CDNPath string
	cache   sync.Map // objectID -> cachedMimeType
}

type cachedMimeType struct {
	mimeType  string
	timestamp time.Time
}

const cacheTTL = 5 * time.Minute

// extensionMimeMap maps file extensions to MIME types.
var extensionMimeMap = map[string]string{
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

// ServeHTTP intercepts CDN requests and sets the correct MIME type.
func (m *MimeTypeMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/cdn/") {
			next.ServeHTTP(w, r)
			return
		}

		objectID := strings.TrimPrefix(r.URL.Path, "/cdn/")
		if objectID == "" {
			next.ServeHTTP(w, r)
			return
		}

		filePath := filepath.Join(m.CDNPath, objectID)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			next.ServeHTTP(w, r)
			return
		}

		mimeType := m.detectMimeType(filePath)
		w.Header().Set("Content-Type", mimeType)
		http.ServeFile(w, r, filePath)
	})
}

func (m *MimeTypeMiddleware) detectMimeType(filePath string) string {
	// Try extension first
	ext := strings.ToLower(filepath.Ext(filePath))
	if mt, ok := extensionMimeMap[ext]; ok {
		return mt
	}

	// Try magic bytes
	f, err := os.Open(filePath)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n == 0 {
		return "application/octet-stream"
	}

	return http.DetectContentType(buf[:n])
}
