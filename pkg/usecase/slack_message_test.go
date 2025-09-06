package usecase_test

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack/slackevents"
)

// MockLLMClient implements interfaces.LLMClient for testing
type MockLLMClient struct {
	GenerateResponseFunc func(ctx context.Context, prompt string) (string, error)
	AnalyzeMessageFunc   func(ctx context.Context, message string) (string, error)
}

func (m *MockLLMClient) GenerateResponse(ctx context.Context, prompt string) (string, error) {
	if m.GenerateResponseFunc != nil {
		return m.GenerateResponseFunc(ctx, prompt)
	}
	return "Mock response", nil
}

func (m *MockLLMClient) AnalyzeMessage(ctx context.Context, message string) (string, error) {
	if m.AnalyzeMessageFunc != nil {
		return m.AnalyzeMessageFunc(ctx, message)
	}
	return "Mock analysis", nil
}

func TestSlackMessageProcessMessage(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	llm := &MockLLMClient{}

	uc := usecase.NewSlackMessage(ctx, repo, llm, nil)

	event := &slackevents.MessageEvent{
		ClientMsgID:     "msg-test-001",
		User:            "U12345",
		Username:        "testuser",
		Channel:         "C12345",
		Text:            "Test message",
		TimeStamp:       "1234567890.123456",
		ThreadTimeStamp: "",
	}

	err := uc.ProcessMessage(ctx, event)
	gt.NoError(t, err)

	// Verify message was saved
	saved, err := repo.GetMessage(ctx, "msg-test-001")
	gt.NoError(t, err)
	gt.Equal(t, "msg-test-001", saved.ID)
	gt.Equal(t, "Test message", saved.Text)
	gt.Equal(t, "U12345", saved.UserID)
	gt.Equal(t, "C12345", saved.ChannelID)
}

func TestSlackMessageGenerateResponse(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()

	t.Run("With LLM client", func(t *testing.T) {
		llm := &MockLLMClient{
			GenerateResponseFunc: func(ctx context.Context, prompt string) (string, error) {
				return "Generated response", nil
			},
		}

		uc := usecase.NewSlackMessage(ctx, repo, llm, nil)

		message := &model.Message{
			ID:   "msg-001",
			Text: "Help me with incident",
		}

		response, err := uc.GenerateResponse(ctx, message)
		gt.NoError(t, err)
		gt.Equal(t, "Generated response", response)
	})

	t.Run("Without LLM client", func(t *testing.T) {
		uc := usecase.NewSlackMessage(ctx, repo, nil, nil)

		message := &model.Message{
			ID:   "msg-002",
			Text: "Help me with incident",
		}

		response, err := uc.GenerateResponse(ctx, message)
		gt.NoError(t, err)
		gt.Equal(t, "Thank you for your message. I'm currently processing it.", response)
	})
}

func TestSlackMessageSaveAndRespond(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	llm := &MockLLMClient{
		GenerateResponseFunc: func(ctx context.Context, prompt string) (string, error) {
			return "Here's help with your incident", nil
		},
	}

	uc := usecase.NewSlackMessage(ctx, repo, llm, nil)

	event := &slackevents.MessageEvent{
		ClientMsgID:     "msg-test-002",
		User:            "U67890",
		Username:        "helpuser",
		Channel:         "C67890",
		Text:            "I need help with an incident",
		TimeStamp:       "1234567890.654321",
		ThreadTimeStamp: "",
	}

	response, err := uc.SaveAndRespond(ctx, event)
	gt.NoError(t, err)
	gt.Equal(t, "Here's help with your incident", response)

	// Verify message was saved
	saved, err := repo.GetMessage(ctx, "msg-test-002")
	gt.NoError(t, err)
	gt.Equal(t, "I need help with an incident", saved.Text)
}
