package http

import (
	"io"
	"net/http"
	"path"
	"strings"

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
	// Clean the path
	cleanPath := path.Clean(r.URL.Path)

	// Try to open the requested file
	file, err := h.fileSystem.Open(cleanPath)
	if err == nil {
		defer file.Close()

		// Check if it's a directory
		if stat, err := file.Stat(); err == nil && !stat.IsDir() {
			// File exists and is not a directory, serve it
			h.serveFile(w, r, file, cleanPath)
			return
		}
	}

	// If file doesn't exist or is a directory, check for index.html in that directory
	if strings.HasSuffix(cleanPath, "/") || cleanPath == "" {
		indexPath := path.Join(cleanPath, "index.html")
		if indexFile, err := h.fileSystem.Open(indexPath); err == nil {
			defer indexFile.Close()
			if stat, err := indexFile.Stat(); err == nil && !stat.IsDir() {
				h.serveFile(w, r, indexFile, indexPath)
				return
			}
		}
	}

	// Fallback to root index.html for SPA routing
	h.serveSPAFallback(w, r)
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

// getContentType returns the content type for common file extensions
func getContentType(filePath string) string {
	ext := path.Ext(filePath)
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return ""
	}
}
