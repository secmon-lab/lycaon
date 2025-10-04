package http

import (
	"context"
	_ "embed"
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

//go:embed static/fallback.html
var fallbackHTML []byte

// Config holds configuration for the HTTP server
type Config struct {
	slackConfig *config.SlackConfig
	modelConfig *model.Config
	addr        string
	frontendURL string
}

// NewConfig creates a new Config instance
func NewConfig(
	addr string,
	slackConfig *config.SlackConfig,
	modelConfig *model.Config,
	frontendURL string,
) *Config {
	return &Config{
		slackConfig: slackConfig,
		modelConfig: modelConfig,
		addr:        addr,
		frontendURL: frontendURL,
	}
}

// UseCases holds use case dependencies for the HTTP server
type UseCases struct {
	auth             interfaces.Auth
	slackMessage     interfaces.SlackMessage
	incident         interfaces.Incident
	task             interfaces.Task
	slackInteraction interfaces.SlackInteraction
}

// NewUseCases creates a new UseCases instance
func NewUseCases(
	authUC interfaces.Auth,
	messageUC interfaces.SlackMessage,
	incidentUC interfaces.Incident,
	taskUC interfaces.Task,
	slackInteractionUC interfaces.SlackInteraction,
) *UseCases {
	return &UseCases{
		auth:             authUC,
		slackMessage:     messageUC,
		incident:         incidentUC,
		task:             taskUC,
		slackInteraction: slackInteractionUC,
	}
}

// Controllers holds controller dependencies for the HTTP server
type Controllers struct {
	slackHandler   *slackCtrl.Handler
	authHandler    *AuthHandler
	graphqlHandler http.Handler
}

// NewController creates a new Controllers instance with pre-created handlers
func NewController(
	slackHandler *slackCtrl.Handler,
	authHandler *AuthHandler,
	graphqlHandler http.Handler,
) *Controllers {
	return &Controllers{
		slackHandler:   slackHandler,
		authHandler:    authHandler,
		graphqlHandler: graphqlHandler,
	}
}

// Server represents the HTTP server
type Server struct {
	*http.Server
}

// NewServer creates a new HTTP server
func NewServer(
	ctx context.Context,
	config *Config,
	useCases *UseCases,
	controllers *Controllers,
	repo interfaces.Repository,
) (*Server, error) {
	router := chi.NewRouter()
	authMiddleware := NewMiddleware(ctx, useCases.auth)

	// Apply global middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(LoggingMiddleware(ctx))
	router.Use(AuthContextMiddleware())
	router.Use(middleware.Recoverer)

	// Health check
	router.Get("/health", handleHealth)

	// API routes
	router.Route("/api", func(r chi.Router) {
		// Auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Get("/login", controllers.authHandler.HandleLogin)
			r.Get("/callback", controllers.authHandler.HandleCallback)
			r.Post("/logout", controllers.authHandler.HandleLogout)
		})

		// User routes (protected)
		r.Route("/user", func(r chi.Router) {
			r.Use(authMiddleware.RequireAuth)
			r.Get("/me", controllers.authHandler.HandleUserMe)
		})
	})

	// GraphQL endpoint
	if controllers.graphqlHandler != nil {
		router.Route("/graphql", func(r chi.Router) {
			// Apply authentication middleware to GraphQL
			// Note: This ensures GraphQL is protected by authentication
			r.Use(authMiddleware.RequireAuth)
			r.Handle("/", controllers.graphqlHandler)
		})

		// GraphQL Playground (development only)
		// Enable playground in development mode
		router.Handle("/playground", playground.Handler("GraphQL playground", "/graphql"))
	}

	// Slack webhook routes
	router.Route("/hooks/slack", func(r chi.Router) {
		r.Post("/event", controllers.slackHandler.HandleEvent)
		r.Post("/interaction", controllers.slackHandler.HandleInteraction)
	})

	// Frontend routes (serve embedded or filesystem)
	// In production, serve embedded files with SPA support
	fs, err := frontend.GetHTTPFS()
	if err != nil {
		ctxlog.From(ctx).Warn("Failed to get embedded frontend, using fallback",
			"error", err,
		)
		// Fallback to a simple handler
		router.Get("/*", handleFallbackHome)
	} else {
		ctxlog.From(ctx).Info("Serving frontend from embedded files with SPA support")
		spaHandler, err := NewSPAHandler(fs)
		if err != nil {
			ctxlog.From(ctx).Error("Failed to create SPA handler, using simple file server",
				"error", err,
			)
			fileServer := http.FileServer(fs)
			router.Handle("/*", fileServer)
		} else {
			router.Handle("/*", spaHandler)
		}
	}

	server := &Server{
		Server: &http.Server{
			Addr:              config.addr,
			Handler:           router,
			ReadHeaderTimeout: 15 * time.Second,
		},
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
	if _, err := w.Write(fallbackHTML); err != nil {
		ctxlog.From(r.Context()).Error("Failed to write fallback home page", "error", err)
	}
}

// CreateGraphQLHandler creates a GraphQL handler with dependencies
// This is a helper function that can be used externally to create the GraphQL handler
func CreateGraphQLHandler(repo interfaces.Repository, slackClient interfaces.SlackClient, useCases *UseCases, modelConfig *model.Config) http.Handler {
	gqlUseCases := &graphql.UseCases{
		IncidentUC: useCases.incident,
		TaskUC:     useCases.task,
		AuthUC:     useCases.auth,
	}

	resolver := graphql.NewResolver(repo, slackClient, gqlUseCases, modelConfig)
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
