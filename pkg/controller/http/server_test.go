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
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	slackgo "github.com/slack-go/slack"
)

// Simple mock SlackClient for HTTP controller tests
type simpleSlackMock struct{}

func (m *simpleSlackMock) CreateConversation(ctx context.Context, params slackgo.CreateConversationParams) (*slackgo.Channel, error) {
	return &slackgo.Channel{}, nil
}
func (m *simpleSlackMock) InviteUsersToConversation(ctx context.Context, channelID string, users ...string) (*slackgo.Channel, error) {
	return &slackgo.Channel{}, nil
}
func (m *simpleSlackMock) PostMessage(ctx context.Context, channelID string, options ...slackgo.MsgOption) (string, string, error) {
	return "", "", nil
}
func (m *simpleSlackMock) UpdateMessage(ctx context.Context, channelID, timestamp string, options ...slackgo.MsgOption) (string, string, string, error) {
	return "", "", "", nil
}
func (m *simpleSlackMock) AuthTestContext(ctx context.Context) (*slackgo.AuthTestResponse, error) {
	return &slackgo.AuthTestResponse{UserID: "U_TEST_BOT", User: "test-bot"}, nil
}
func (m *simpleSlackMock) GetConversationInfo(ctx context.Context, channelID string, includeLocale bool) (*slackgo.Channel, error) {
	return &slackgo.Channel{}, nil
}
func (m *simpleSlackMock) SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*slackgo.Channel, error) {
	return &slackgo.Channel{}, nil
}
func (m *simpleSlackMock) OpenView(ctx context.Context, triggerID string, view slackgo.ModalViewRequest) (*slackgo.ViewResponse, error) {
	return &slackgo.ViewResponse{}, nil
}
func (m *simpleSlackMock) GetConversationHistoryContext(ctx context.Context, params *slackgo.GetConversationHistoryParameters) (*slackgo.GetConversationHistoryResponse, error) {
	return &slackgo.GetConversationHistoryResponse{}, nil
}
func (m *simpleSlackMock) GetConversationRepliesContext(ctx context.Context, params *slackgo.GetConversationRepliesParameters) ([]slackgo.Message, bool, bool, error) {
	return []slackgo.Message{}, false, false, nil
}

// Helper to create mock clients for HTTP tests
func createMockClients() (gollem.LLMClient, *simpleSlackMock) {
	return &mock.LLMClientMock{}, &simpleSlackMock{}
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
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack)
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
	mockLLM, mockSlack := createMockClients()
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack)
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
