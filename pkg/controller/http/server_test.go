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
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	controller "github.com/secmon-lab/lycaon/pkg/controller/http"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	slackgo "github.com/slack-go/slack"
)

// Helper to create mock clients for HTTP tests
func createMockClients() (gollem.LLMClient, *mocks.SlackClientMock) {
	return &mock.LLMClientMock{}, &mocks.SlackClientMock{
		AuthTestContextFunc: func(ctx context.Context) (*slackgo.AuthTestResponse, error) {
			return &slackgo.AuthTestResponse{UserID: "U_TEST_BOT", User: "test-bot"}, nil
		},
	}
}

func TestServerHealthCheck(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	authUC := usecase.NewAuth(ctx, repo, slackConfig)
	mockLLM, mockSlack := createMockClients()
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, model.GetDefaultCategories())
	gt.NoError(t, err).Required()
	categories := model.GetDefaultCategories()
	incidentUC := usecase.NewIncident(repo, nil, categories)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)
	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, mockSlack)

	server, err := controller.NewServer(
		ctx,
		":8080",
		slackConfig,
		repo,
		authUC,
		messageUC,
		incidentUC,
		taskUC,
		slackInteractionUC,
		mockSlack,
		"",
	)
	gt.NoError(t, err).Required()

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
	mockLLM, mockSlack := createMockClients()
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, model.GetDefaultCategories())
	gt.NoError(t, err).Required()
	categories := model.GetDefaultCategories()
	incidentUC := usecase.NewIncident(repo, nil, categories)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)
	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, mockSlack)

	server, err := controller.NewServer(
		ctx,
		":8080",
		slackConfig,
		repo,
		authUC,
		messageUC,
		incidentUC,
		taskUC,
		slackInteractionUC,
		mockSlack,
		"",
	)
	gt.NoError(t, err).Required()

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
