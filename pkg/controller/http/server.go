package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/frontend"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	slackCtrl "github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
)

// Server represents the HTTP server
type Server struct {
	*http.Server
	router         chi.Router
	slackConfig    *config.SlackConfig
	authUC         interfaces.Auth
	messageUC      interfaces.SlackMessage
	devMode        bool
	authMiddleware *Middleware
	slackHandler   *slackCtrl.Handler
	authHandler    *AuthHandler
}

// NewServer creates a new HTTP server
func NewServer(
	ctx context.Context,
	addr string,
	slackConfig *config.SlackConfig,
	repo interfaces.Repository,
	authUC interfaces.Auth,
	messageUC interfaces.SlackMessage,
	incidentUC interfaces.Incident,
	devMode bool,
	frontendURL string,
) (*Server, error) {
	router := chi.NewRouter()
	authMiddleware := NewMiddleware(ctx, authUC)

	// Apply global middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(LoggingMiddleware(ctx))
	router.Use(middleware.Recoverer)

	slackHandler := slackCtrl.NewHandler(ctx, slackConfig, repo, messageUC, incidentUC)
	authHandler := NewAuthHandler(ctx, slackConfig, authUC, frontendURL)

	// Health check
	router.Get("/health", handleHealth)

	// API routes
	router.Route("/api", func(r chi.Router) {
		// Auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Get("/login", authHandler.HandleLogin)
			r.Get("/callback", authHandler.HandleCallback)
			r.Post("/logout", authHandler.HandleLogout)
		})

		// User routes (protected)
		r.Route("/user", func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			r.Get("/me", authHandler.HandleUserMe)
		})
	})

	// Slack webhook routes
	router.Route("/hooks/slack", func(r chi.Router) {
		r.Post("/event", slackHandler.HandleEvent)
		r.Post("/interaction", slackHandler.HandleInteraction)
	})

	// Frontend routes (serve embedded or filesystem)
	if devMode {
		// In dev mode, serve from filesystem
		ctxlog.From(ctx).Info("Serving frontend from filesystem (dev mode)")
		fs := http.FileServer(http.Dir("frontend/dist"))
		router.Handle("/*", fs)
	} else {
		// In production, serve embedded files
		fs, err := frontend.GetHTTPFS()
		if err != nil {
			ctxlog.From(ctx).Warn("Failed to get embedded frontend, using fallback",
				"error", err,
			)
			// Fallback to a simple handler
			router.Get("/*", handleFallbackHome)
		} else {
			ctxlog.From(ctx).Info("Serving frontend from embedded files")
			fileServer := http.FileServer(fs)
			router.Handle("/*", fileServer)
		}
	}

	server := &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: router,
		},
		router:         router,
		slackConfig:    slackConfig,
		authUC:         authUC,
		messageUC:      messageUC,
		devMode:        devMode,
		authMiddleware: authMiddleware,
		slackHandler:   slackHandler,
		authHandler:    authHandler,
	}

	return server, nil
}

// handleHealth handles health check requests
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "lycaon",
	})
}

// handleFallbackHome handles the root path when frontend is not available
func handleFallbackHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
    <title>Lycaon</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        .container {
            text-align: center;
            padding: 2rem;
            background: rgba(255, 255, 255, 0.1);
            border-radius: 10px;
            backdrop-filter: blur(10px);
        }
        h1 {
            margin: 0 0 1rem 0;
            font-size: 3rem;
        }
        p {
            margin: 0.5rem 0;
            font-size: 1.2rem;
        }
        a {
            color: white;
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎭 Lycaon</h1>
        <p>Slack-based Incident Management Service</p>
        <p><a href="/api/auth/login">Sign in with Slack</a></p>
    </div>
</body>
</html>`))
}

// writeError writes an error response
func writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	var message string
	if goErr := goerr.Unwrap(err); goErr != nil {
		message = goErr.Error()
	} else {
		message = err.Error()
	}

	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}
