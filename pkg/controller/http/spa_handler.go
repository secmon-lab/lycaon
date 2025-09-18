package http

import (
	"io"
	"net/http"
	"os"
	"path"

	"github.com/m-mizutani/goerr/v2"
)

// SPAHandler handles Single Page Application routing with fallback to index.html
type SPAHandler struct {
	fileSystem http.FileSystem
	indexFile  []byte
}

// NewSPAHandler creates a new SPA handler
func NewSPAHandler(filesystem http.FileSystem) (*SPAHandler, error) {
	// Read index.html content for fallback
	indexFile, err := filesystem.Open("/index.html")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to open index.html for SPA handler")
	}
	defer indexFile.Close()

	indexContent, err := io.ReadAll(indexFile)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to read index.html content")
	}

	return &SPAHandler{
		fileSystem: filesystem,
		indexFile:  indexContent,
	}, nil
}

// ServeHTTP implements the http.Handler interface for SPA routing
func (h *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path to prevent directory traversal attacks.
	cleanPath := path.Clean(r.URL.Path)

	// Try to open the requested file.
	file, err := h.fileSystem.Open(cleanPath)
	if err != nil {
		// If the file doesn't exist, it's likely a SPA route.
		// Fallback to serving index.html.
		if os.IsNotExist(err) {
			h.serveSPAFallback(w, r)
			return
		}
		// For other errors (e.g., permission denied), return an internal server error.
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Get file stats.
	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// If the path is a directory, it's not a static asset. Fallback to index.html.
	if stat.IsDir() {
		h.serveSPAFallback(w, r)
		return
	}

	// The path corresponds to an existing file, so serve it.
	h.serveFile(w, r, file, cleanPath)
}

// serveFile serves a specific file with appropriate headers
func (h *SPAHandler) serveFile(w http.ResponseWriter, r *http.Request, file http.File, filePath string) {
	// Set content type based on file extension
	contentType := getContentType(filePath)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	// Copy file content to response
	if _, err := io.Copy(w, file); err != nil {
		http.Error(w, "Failed to serve file", http.StatusInternalServerError)
		return
	}
}

// serveSPAFallback serves the index.html for SPA routing
func (h *SPAHandler) serveSPAFallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write(h.indexFile); err != nil {
		http.Error(w, "Failed to serve SPA fallback", http.StatusInternalServerError)
		return
	}
}

var mimeTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "application/javascript; charset=utf-8",
	".json":  "application/json; charset=utf-8",
	".png":   "image/png",
	".jpg":   "image/jpeg",
	".jpeg":  "image/jpeg",
	".gif":   "image/gif",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".ttf":   "font/ttf",
	".eot":   "application/vnd.ms-fontobject",
}

// getContentType returns the content type for common file extensions
func getContentType(filePath string) string {
	ext := path.Ext(filePath)
	return mimeTypes[ext]
}
