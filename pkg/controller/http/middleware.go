package http

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/m-mizutani/ctxlog"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
)

// Middleware provides common HTTP middleware
type Middleware struct {
	authUC interfaces.Auth
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(ctx context.Context, authUC interfaces.Auth) *Middleware {
	return &Middleware{
		authUC: authUC,
	}
}

// CORS middleware adds CORS headers
func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAuth middleware checks session authentication (chi compatible)
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get session ID and secret from cookies
		sessionIDCookie, err := r.Cookie("session_id")
		if err != nil {
			http.Error(w, "Unauthorized: missing session_id", http.StatusUnauthorized)
			return
		}

		sessionSecretCookie, err := r.Cookie("session_secret")
		if err != nil {
			http.Error(w, "Unauthorized: missing session_secret", http.StatusUnauthorized)
			return
		}

		// Validate session
		session, err := m.authUC.ValidateSession(r.Context(), sessionIDCookie.Value, sessionSecretCookie.Value)
		if err != nil {
			logger := ctxlog.From(r.Context())
			logger.Debug("Session validation failed",
				"error", err,
				"sessionID", sessionIDCookie.Value,
			)
			http.Error(w, "Unauthorized: invalid session", http.StatusUnauthorized)
			return
		}

		// Add user info to request context (could be enhanced with context values)
		logger := ctxlog.From(r.Context())
		logger.Debug("Authenticated request",
			"userID", session.UserID,
			"sessionID", session.ID,
		)

		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware creates a chi-compatible logging middleware
func LoggingMiddleware(ctx context.Context) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Embed logger from the initial context into request context
			r = r.WithContext(ctxlog.With(r.Context(), ctxlog.From(ctx)))

			logger := ctxlog.From(r.Context())
			start := time.Now()

			// Wrap response writer to capture status
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			// Process request
			next.ServeHTTP(ww, r)

			// Log request
			logger.Info("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.Query(),
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration", time.Since(start),
				"remote", r.RemoteAddr,
			)
		})
	}
}
