package http_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	controller "github.com/secmon-lab/lycaon/pkg/controller/http"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
)

func TestServerHealthCheck(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)
	messageUC, err := usecase.NewSlackMessage(ctx, repo, nil, nil, "")
	gt.NoError(t, err)
	incidentUC := usecase.NewIncident(repo, nil)

	server, err := controller.NewServer(
		ctx,
		":8080",
		slackConfig,
		repo,
		authUC,
		messageUC,
		incidentUC,
		false,
		"",
	)
	gt.NoError(t, err)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Execute
	server.Server.Handler.ServeHTTP(w, req)

	// Assert
	gt.Equal(t, http.StatusOK, w.Code)
	gt.True(t, strings.Contains(w.Body.String(), "healthy"))
	gt.True(t, strings.Contains(w.Body.String(), "lycaon"))
}

func TestServerFallbackHome(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)
	messageUC, err := usecase.NewSlackMessage(ctx, repo, nil, nil, "")
	gt.NoError(t, err)
	incidentUC := usecase.NewIncident(repo, nil)

	server, err := controller.NewServer(
		ctx,
		":8080",
		slackConfig,
		repo,
		authUC,
		messageUC,
		incidentUC,
		false, // Production mode to trigger fallback
		"",
	)
	gt.NoError(t, err)

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Execute
	server.Server.Handler.ServeHTTP(w, req)

	// Assert
	gt.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	t.Logf("Response body: %s", body)
	// Check that we got an HTML response
	gt.True(t, strings.Contains(body, "<!DOCTYPE html>") || strings.Contains(body, "<!doctype html>"))
	gt.True(t, strings.Contains(body, "<html"))
	gt.True(t, strings.Contains(body, "</html>"))
}
