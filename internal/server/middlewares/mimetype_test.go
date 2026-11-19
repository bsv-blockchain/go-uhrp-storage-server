package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/bsv-blockchain/go-uhrp-storage-server/internal/server/middlewares"
)

// passedThrough marks requests the middleware handed to the next handler.
const passedThrough = http.StatusTeapot

func newCDN() http.Handler {
	files := fstest.MapFS{
		"page.HTML":       {Data: []byte("<p>hi</p>")},
		"logo.png":        {Data: []byte("not really a png")},
		"noext-png":       {Data: []byte("\x89PNG\r\n\x1a\n" + "rest-of-image")},
		"noext-text":      {Data: []byte("plain words")},
		"empty":           {Data: []byte{}},
		"nested/file.txt": {Data: []byte("nested")},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(passedThrough)
	})

	return middlewares.NewMimeTypeMiddleware(files).Handle(next)
}

func TestMimeTypeMiddlewareServesFiles(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		contentType string
		body        string
	}{
		{name: "extension is case-insensitive", path: "/cdn/page.HTML", contentType: "text/html", body: "<p>hi</p>"},
		{name: "extension wins over content", path: "/cdn/logo.png", contentType: "image/png", body: "not really a png"},
		{name: "png detected from content", path: "/cdn/noext-png", contentType: "image/png"},
		{name: "text detected from content", path: "/cdn/noext-text", contentType: "text/plain; charset=utf-8", body: "plain words"},
		{name: "empty file is octet-stream", path: "/cdn/empty", contentType: "application/octet-stream"},
		{name: "nested path", path: "/cdn/nested/file.txt", contentType: "text/plain", body: "nested"},
	}

	cdn := newCDN()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			cdn.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, nil))

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tt.contentType, w.Header().Get("Content-Type"))

			if tt.body != "" {
				assert.Equal(t, tt.body, w.Body.String())
			}
		})
	}
}

func TestMimeTypeMiddlewarePassesThrough(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "not a cdn path", path: "/quote"},
		{name: "cdn root", path: "/cdn/"},
		{name: "directory", path: "/cdn/nested"},
		{name: "directory with slash", path: "/cdn/nested/"},
		{name: "missing file", path: "/cdn/missing"},
		{name: "parent traversal", path: "/cdn/../go.mod"},
		{name: "encoded traversal", path: "/cdn/..%2fgo.mod"},
		{name: "absolute path", path: "/cdn//etc/passwd"},
	}

	cdn := newCDN()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			cdn.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, tt.path, nil))

			assert.Equal(t, passedThrough, w.Code)
		})
	}
}

func TestMimeTypeMiddlewareWithoutFiles(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(passedThrough) })
	handler := (&middlewares.MimeTypeMiddleware{}).Handle(next)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/cdn/anything", nil))

	assert.Equal(t, passedThrough, w.Code)
}
