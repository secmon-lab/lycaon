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
	return http.FS(sub), nil
}
