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
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	slackgo "github.com/slack-go/slack"
)

// MockSlackClient for controller tests - simplified version of the one in incident tests
type MockSlackClient struct {
	CreateConversationFunc              func(params slackgo.CreateConversationParams) (*slackgo.Channel, error)
	InviteUsersToConversationFunc       func(channelID string, users ...string) (*slackgo.Channel, error)
	PostMessageFunc                     func(channelID string, options ...slackgo.MsgOption) (string, string, error)
	UpdateMessageFunc                   func(channelID, timestamp string, options ...slackgo.MsgOption) (string, string, string, error)
	AuthTestContextFunc                 func(ctx context.Context) (*slackgo.AuthTestResponse, error)
	GetConversationInfoFunc             func(ctx context.Context, channelID string, includeLocale bool) (*slackgo.Channel, error)
	SetPurposeOfConversationContextFunc func(ctx context.Context, channelID, purpose string) (*slackgo.Channel, error)
	OpenViewFunc                        func(ctx context.Context, triggerID string, view slackgo.ModalViewRequest) (*slackgo.ViewResponse, error)
	GetConversationHistoryContextFunc   func(ctx context.Context, params *slackgo.GetConversationHistoryParameters) (*slackgo.GetConversationHistoryResponse, error)
	GetConversationRepliesContextFunc   func(ctx context.Context, params *slackgo.GetConversationRepliesParameters) ([]slackgo.Message, bool, bool, error)
}

func (m *MockSlackClient) CreateConversation(ctx context.Context, params slackgo.CreateConversationParams) (*slackgo.Channel, error) {
	if m.CreateConversationFunc != nil {
		return m.CreateConversationFunc(params)
	}
	return &slackgo.Channel{}, nil
}

func (m *MockSlackClient) InviteUsersToConversation(ctx context.Context, channelID string, users ...string) (*slackgo.Channel, error) {
	if m.InviteUsersToConversationFunc != nil {
		return m.InviteUsersToConversationFunc(channelID, users...)
	}
	return &slackgo.Channel{}, nil
}

func (m *MockSlackClient) PostMessage(ctx context.Context, channelID string, options ...slackgo.MsgOption) (string, string, error) {
	if m.PostMessageFunc != nil {
		return m.PostMessageFunc(channelID, options...)
	}
	return "", "", nil
}

func (m *MockSlackClient) UpdateMessage(ctx context.Context, channelID, timestamp string, options ...slackgo.MsgOption) (string, string, string, error) {
	if m.UpdateMessageFunc != nil {
		return m.UpdateMessageFunc(channelID, timestamp, options...)
	}
	return "", "", "", nil
}

func (m *MockSlackClient) AuthTestContext(ctx context.Context) (*slackgo.AuthTestResponse, error) {
	if m.AuthTestContextFunc != nil {
		return m.AuthTestContextFunc(ctx)
	}
	return &slackgo.AuthTestResponse{
		UserID: "U_TEST_BOT",
		User:   "test-bot",
	}, nil
}

func (m *MockSlackClient) GetConversationInfo(ctx context.Context, channelID string, includeLocale bool) (*slackgo.Channel, error) {
	if m.GetConversationInfoFunc != nil {
		return m.GetConversationInfoFunc(ctx, channelID, includeLocale)
	}
	return &slackgo.Channel{}, nil
}

func (m *MockSlackClient) SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*slackgo.Channel, error) {
	if m.SetPurposeOfConversationContextFunc != nil {
		return m.SetPurposeOfConversationContextFunc(ctx, channelID, purpose)
	}
	return &slackgo.Channel{}, nil
}

func (m *MockSlackClient) OpenView(ctx context.Context, triggerID string, view slackgo.ModalViewRequest) (*slackgo.ViewResponse, error) {
	if m.OpenViewFunc != nil {
		return m.OpenViewFunc(ctx, triggerID, view)
	}
	return &slackgo.ViewResponse{}, nil
}

func (m *MockSlackClient) GetConversationHistoryContext(ctx context.Context, params *slackgo.GetConversationHistoryParameters) (*slackgo.GetConversationHistoryResponse, error) {
	if m.GetConversationHistoryContextFunc != nil {
		return m.GetConversationHistoryContextFunc(ctx, params)
	}
	return &slackgo.GetConversationHistoryResponse{}, nil
}

func (m *MockSlackClient) GetConversationRepliesContext(ctx context.Context, params *slackgo.GetConversationRepliesParameters) ([]slackgo.Message, bool, bool, error) {
	if m.GetConversationRepliesContextFunc != nil {
		return m.GetConversationRepliesContextFunc(ctx, params)
	}
	return []slackgo.Message{}, false, false, nil
}

// Helper function to create mock clients for controller tests
func createMockClientsForController() (gollem.LLMClient, *MockSlackClient) {
	mockLLM := &mock.LLMClientMock{}
	mockSlack := &MockSlackClient{}
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
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack)
	gt.NoError(t, err)
	incidentUC := usecase.NewIncident(repo, nil)

	handler := slack.NewHandler(ctx, slackConfig, repo, messageUC, incidentUC)

	// Create challenge request with type field
	challenge := map[string]any{
		"type":      "url_verification",
		"challenge": "test-challenge-string",
		"token":     "test-token",
	}
	body, err := json.Marshal(challenge)
	gt.NoError(t, err)

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
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack)
	gt.NoError(t, err)
	incidentUC := usecase.NewIncident(repo, nil)

	handler := slack.NewHandler(ctx, slackConfig, repo, messageUC, incidentUC)

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
	messageUC, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack)
	gt.NoError(t, err)
	incidentUC := usecase.NewIncident(repo, nil)

	handler := slack.NewHandler(ctx, slackConfig, repo, messageUC, incidentUC)

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
