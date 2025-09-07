package http

import (
	"fmt"
	"net/http"
	"strings"
)

// GetFrontendURL returns the frontend URL based on configuration and request
// configuredURL: URL configured via environment variable or configuration
// If configuredURL is empty, dynamically constructs URL from request headers
func GetFrontendURL(r *http.Request, configuredURL string) string {
	// If explicitly configured, use that URL
	if configuredURL != "" {
		return configuredURL
	}

	// Dynamically construct URL from request
	// Always use HTTPS as we assume TLS termination at reverse proxy
	scheme := "https"

	// Determine host from headers
	// Priority: Alt-Used (Cloud Run) > X-Forwarded-Host > Host
	host := r.Host
	
	// Check for Cloud Run's Alt-Used header first
	if altUsed := r.Header.Get("Alt-Used"); altUsed != "" {
		host = altUsed
	} else if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		// X-Forwarded-Host may contain multiple hosts separated by comma
		// Use the first one (original client request)
		if parts := strings.Split(forwardedHost, ","); len(parts) > 0 {
			host = strings.TrimSpace(parts[0])
		}
	}

	// Fallback to localhost if no host header
	if host == "" {
		host = "localhost"
	}

	return fmt.Sprintf("%s://%s", scheme, host)
}
