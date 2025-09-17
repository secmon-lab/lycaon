package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

// FS embeds the frontend build artifacts
//
//go:embed all:dist
var FS embed.FS

// GetHTTPFS returns the embedded frontend filesystem for HTTP serving
func GetHTTPFS() (http.FileSystem, error) {
	sub, err := fs.Sub(FS, "dist")
	if err != nil {
		return nil, err
	}

	// Check if the dist directory actually contains built frontend files
	if !isFrontendBuilt(sub) {
		return nil, &fs.PathError{Op: "stat", Path: "index.html", Err: fs.ErrNotExist}
	}

	return http.FS(sub), nil
}

// isFrontendBuilt checks if the frontend has been properly built
func isFrontendBuilt(fsys fs.FS) bool {
	// Check for index.html as a marker that frontend is built
	if _, err := fs.Stat(fsys, "index.html"); err != nil {
		return false
	}
	return true
}
