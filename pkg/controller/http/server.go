package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/frontend"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	"github.com/secmon-lab/lycaon/pkg/controller/graphql"
	slackCtrl "github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
)

// Server represents the HTTP server
type Server struct {
	*http.Server
	router         chi.Router
	slackConfig    *config.SlackConfig
	authUC         interfaces.Auth
	messageUC      interfaces.SlackMessage
	authMiddleware *Middleware
	slackHandler   *slackCtrl.Handler
	authHandler    *AuthHandler
}

// NewServer creates a new HTTP server
func NewServer(
	ctx context.Context,
	addr string,
	slackConfig *config.SlackConfig,
	categories *model.CategoriesConfig,
	repo interfaces.Repository,
	authUC interfaces.Auth,
	messageUC interfaces.SlackMessage,
	incidentUC interfaces.Incident,
	taskUC interfaces.Task,
	slackInteractionUC interfaces.SlackInteraction,
	slackClient interfaces.SlackClient,
	frontendURL string,
) (*Server, error) {
	router := chi.NewRouter()
	authMiddleware := NewMiddleware(ctx, authUC)

	// Apply global middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(LoggingMiddleware(ctx))
	router.Use(AuthContextMiddleware())
	router.Use(middleware.Recoverer)

	slackHandler := slackCtrl.NewHandler(ctx, slackConfig, repo, messageUC, incidentUC, taskUC, slackInteractionUC, slackClient)
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

	// GraphQL endpoint
	if repo != nil && incidentUC != nil && taskUC != nil {
		graphqlHandler := createGraphQLHandler(repo, slackClient, incidentUC, taskUC, categories)

		router.Route("/graphql", func(r chi.Router) {
			// Apply authentication middleware to GraphQL
			// Note: This ensures GraphQL is protected by authentication
			r.Use(authMiddleware.RequireAuth)
			r.Handle("/", graphqlHandler)
		})

		// GraphQL Playground (development only)
		// Enable playground in development mode
		router.Handle("/playground", playground.Handler("GraphQL playground", "/graphql"))
	}

	// Slack webhook routes
	router.Route("/hooks/slack", func(r chi.Router) {
		r.Post("/event", slackHandler.HandleEvent)
		r.Post("/interaction", slackHandler.HandleInteraction)
	})

	// Frontend routes (serve embedded or filesystem)
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

	server := &Server{
		Server: &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: 15 * time.Second,
		},
		router:         router,
		slackConfig:    slackConfig,
		authUC:         authUC,
		messageUC:      messageUC,
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
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "lycaon",
	}); err != nil {
		ctxlog.From(r.Context()).Error("Failed to encode health response", "error", err)
	}
}

// handleFallbackHome handles the root path when frontend is not available
func handleFallbackHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`<!DOCTYPE html>
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
</html>`)); err != nil {
		ctxlog.From(r.Context()).Error("Failed to write fallback home page", "error", err)
	}
}

// createGraphQLHandler creates a GraphQL handler with dependencies
func createGraphQLHandler(repo interfaces.Repository, slackClient interfaces.SlackClient, incidentUC interfaces.Incident, taskUC interfaces.Task, categories *model.CategoriesConfig) http.Handler {
	useCases := &graphql.UseCases{
		IncidentUC: incidentUC,
		TaskUC:     taskUC,
	}

	resolver := graphql.NewResolver(repo, slackClient, useCases, categories)
	srv := handler.NewDefaultServer(
		graphql.NewExecutableSchema(graphql.Config{Resolvers: resolver}),
	)

	// TODO: Add DataLoader middleware here when implemented
	return srv
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

	if err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	}); err != nil {
		// Can't get context here, so use background context
		ctxlog.From(context.Background()).Error("Failed to encode error response", "error", err)
	}
}
