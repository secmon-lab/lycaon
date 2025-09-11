package slack_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	"github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	slackgo "github.com/slack-go/slack"
)

// Helper function to create mock clients for controller tests
func createMockClientsForController() (gollem.LLMClient, *mocks.SlackClientMock) {
	mockLLM := &mock.LLMClientMock{}
	mockSlack := &mocks.SlackClientMock{
		AuthTestContextFunc: func(ctx context.Context) (*slackgo.AuthTestResponse, error) {
			return &slackgo.AuthTestResponse{
				UserID: "U_TEST_BOT",
				User:   "test-bot",
			}, nil
		},
	}
	return mockLLM, mockSlack
}

func TestSlackHandlerChallenge(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{
		SigningSecret: "test-secret",
		OAuthToken:    "test-token",
	}
	repo := repository.NewMemory()
	mockLLM, mockSlack := createMockClientsForController()
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, model.GetDefaultCategories())
	gt.NoError(t, err).Required()
	categories := model.GetDefaultCategories()
	incidentUC := usecase.NewIncident(repo, nil, categories, nil)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)

	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, mockSlack)
	handler := slack.NewHandler(ctx, slackConfig, repo, messageUC, incidentUC, taskUC, slackInteractionUC, mockSlack)

	// Create challenge request with type field
	challenge := map[string]any{
		"type":      "url_verification",
		"challenge": "test-challenge-string",
		"token":     "test-token",
	}
	body, err := json.Marshal(challenge)
	gt.NoError(t, err).Required()

	req := httptest.NewRequest(http.MethodPost, "/hooks/slack/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Add valid signature
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	signature := generateSlackSignature(slackConfig.SigningSecret, timestamp, body)
	req.Header.Set("X-Slack-Signature", signature)

	w := httptest.NewRecorder()

	// Execute
	handler.HandleEvent(w, req)

	// Assert
	gt.Equal(t, http.StatusOK, w.Code)
	gt.Equal(t, "test-challenge-string", w.Body.String())
}

func TestSlackHandlerInvalidSignature(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{
		SigningSecret: "test-secret",
		OAuthToken:    "test-token",
	}
	repo := repository.NewMemory()
	mockLLM, mockSlack := createMockClientsForController()
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, model.GetDefaultCategories())
	gt.NoError(t, err).Required()
	categories := model.GetDefaultCategories()
	incidentUC := usecase.NewIncident(repo, nil, categories, nil)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)

	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, mockSlack)
	handler := slack.NewHandler(ctx, slackConfig, repo, messageUC, incidentUC, taskUC, slackInteractionUC, mockSlack)

	// Create request with invalid signature
	body := []byte(`{"type":"event_callback","event":{"type":"message","text":"test"}}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/slack/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Slack-Request-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("X-Slack-Signature", "v0=invalid")

	w := httptest.NewRecorder()

	// Execute
	handler.HandleEvent(w, req)

	// Assert
	gt.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSlackHandlerNotConfigured(t *testing.T) {
	// Setup with empty config
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)

	slackConfig := &config.SlackConfig{}
	repo := repository.NewMemory()
	mockLLM, mockSlack := createMockClientsForController()
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, model.GetDefaultCategories())
	gt.NoError(t, err).Required()
	categories := model.GetDefaultCategories()
	incidentUC := usecase.NewIncident(repo, nil, categories, nil)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)

	slackInteractionUC := usecase.NewSlackInteraction(incidentUC, taskUC, mockSlack)
	handler := slack.NewHandler(ctx, slackConfig, repo, messageUC, incidentUC, taskUC, slackInteractionUC, mockSlack)

	// Create request with valid JSON body
	body := []byte(`{"type":"event_callback","event":{"type":"message","text":"test"}}`)
	req := httptest.NewRequest(http.MethodPost, "/hooks/slack/events", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Execute
	handler.HandleEvent(w, req)

	// Assert
	gt.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// generateSlackSignature generates a valid Slack signature for testing
func generateSlackSignature(secret, timestamp string, body []byte) string {
	baseString := fmt.Sprintf("v0:%s:%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(baseString))
	return "v0=" + hex.EncodeToString(mac.Sum(nil))
}
