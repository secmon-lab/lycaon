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

	uc, err := usecase.NewSlackMessage(ctx, repo, llm, nil, "")
	gt.NoError(t, err)

	event := &slackevents.MessageEvent{
		ClientMsgID:     "msg-test-001",
		User:            "U12345",
		Username:        "testuser",
		Channel:         "C12345",
		Text:            "Test message",
		TimeStamp:       "1234567890.123456",
		ThreadTimeStamp: "",
	}

	err = uc.ProcessMessage(ctx, event)
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

		uc, err := usecase.NewSlackMessage(ctx, repo, llm, nil, "")
		gt.NoError(t, err)

		message := &model.Message{
			ID:   "msg-001",
			Text: "Help me with incident",
		}

		response, err := uc.GenerateResponse(ctx, message)
		gt.NoError(t, err)
		gt.Equal(t, "Generated response", response)
	})

	t.Run("Without LLM client", func(t *testing.T) {
		uc, err := usecase.NewSlackMessage(ctx, repo, nil, nil, "")
		gt.NoError(t, err)

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

	uc, err := usecase.NewSlackMessage(ctx, repo, llm, nil, "")
	gt.NoError(t, err)

	t.Run("With ClientMsgID", func(t *testing.T) {
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

		// Verify message was saved with ClientMsgID
		saved, err := repo.GetMessage(ctx, "msg-test-002")
		gt.NoError(t, err)
		gt.Equal(t, "I need help with an incident", saved.Text)
	})

	t.Run("Without ClientMsgID (app mention)", func(t *testing.T) {
		event := &slackevents.MessageEvent{
			ClientMsgID:     "", // Empty for app mentions
			User:            "U99999",
			Username:        "mentionuser",
			Channel:         "C99999",
			Text:            "<@U123BOT> inc server is down",
			TimeStamp:       "9876543210.123456",
			ThreadTimeStamp: "",
		}

		response, err := uc.SaveAndRespond(ctx, event)
		gt.NoError(t, err)
		gt.Equal(t, "Here's help with your incident", response)

		// Verify message was saved with TimeStamp as ID
		saved, err := repo.GetMessage(ctx, "9876543210.123456")
		gt.NoError(t, err)
		gt.Equal(t, "<@U123BOT> inc server is down", saved.Text)
	})
}

func TestSlackMessageParseIncidentCommand(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()

	// Use a specific bot ID for testing
	botUserID := "U123BOT"
	uc, err := usecase.NewSlackMessage(ctx, repo, nil, nil, botUserID)
	gt.NoError(t, err)

	testCases := []struct {
		name          string
		message       *model.Message
		expected      bool
		expectedTitle string // Added for new test cases
	}{
		{
			name: "Direct message starting with inc (should be ignored)",
			message: &model.Message{
				ID:   "msg-001",
				Text: "inc something happened",
			},
			expected: false, // Changed: Direct messages without mentions are not accepted
		},
		{
			name: "Direct message starting with INC (should be ignored)",
			message: &model.Message{
				ID:   "msg-002",
				Text: "INC alert",
			},
			expected: false, // Changed: Direct messages without mentions are not accepted
		},
		{
			name: "Direct message starting with Inc (should be ignored)",
			message: &model.Message{
				ID:   "msg-003",
				Text: "Inc issue detected",
			},
			expected: false, // Changed: Direct messages without mentions are not accepted
		},
		{
			name: "Message with inc in middle",
			message: &model.Message{
				ID:   "msg-004",
				Text: "there is an incident",
			},
			expected: false,
		},
		{
			name: "Message without inc",
			message: &model.Message{
				ID:   "msg-005",
				Text: "normal message",
			},
			expected: false,
		},
		{
			name: "Direct message with inc after spaces (should be ignored)",
			message: &model.Message{
				ID:   "msg-006",
				Text: "  inc with spaces",
			},
			expected: false, // Changed: Direct messages without mentions are not accepted
		},
		{
			name: "Empty message",
			message: &model.Message{
				ID:   "msg-007",
				Text: "",
			},
			expected: false,
		},
		{
			name:     "Nil message",
			message:  nil,
			expected: false,
		},
		// New test cases for mention handling
		{
			name: "Bot mention followed by inc",
			message: &model.Message{
				ID:   "msg-008",
				Text: "<@U123BOT> inc production issue",
			},
			expected:      true,
			expectedTitle: "production issue",
		},
		{
			name: "Plain @mention (not Slack format, should be ignored)",
			message: &model.Message{
				ID:   "msg-009",
				Text: "@lycaon inc database is down",
			},
			expected: false, // Changed: Slack always sends mentions as <@USERID> format
		},
		{
			name: "Multiple mentions with bot",
			message: &model.Message{
				ID:   "msg-010",
				Text: "<@U123456> <@U123BOT> inc urgent issue",
			},
			expected: true,
		},
		{
			name: "Bot mention without inc command",
			message: &model.Message{
				ID:   "msg-011",
				Text: "<@U123BOT> help me with this",
			},
			expected: false,
		},
		{
			name: "Bot mention with inc not immediately after",
			message: &model.Message{
				ID:   "msg-012",
				Text: "<@U123BOT> please inc this",
			},
			expected: false,
		},
		{
			name: "Only bot mention without text",
			message: &model.Message{
				ID:   "msg-013",
				Text: "<@U123BOT>",
			},
			expected: false,
		},
		{
			name: "Bot mention with spaces before inc",
			message: &model.Message{
				ID:   "msg-014",
				Text: "<@U123BOT>    inc   with extra spaces",
			},
			expected: true,
		},
		{
			name: "Different user mention with inc (should be ignored)",
			message: &model.Message{
				ID:   "msg-015",
				Text: "<@U999USER> inc server crash",
			},
			expected: false, // Not the bot
		},
		{
			name: "Bot mention with INC uppercase",
			message: &model.Message{
				ID:   "msg-016",
				Text: "<@U123BOT> INC emergency",
			},
			expected: true,
		},
		// Additional test cases for bot ID validation
		{
			name: "Bot mentioned but inc comes after other user",
			message: &model.Message{
				ID:   "msg-017",
				Text: "<@U123BOT> hello <@U999USER> inc problem",
			},
			expected: false, // inc is not immediately after bot mention
		},
		{
			name: "Inc before bot mention",
			message: &model.Message{
				ID:   "msg-018",
				Text: "inc <@U123BOT> help",
			},
			expected: false,
		},
		{
			name: "Multiple bot mentions, inc after first",
			message: &model.Message{
				ID:   "msg-019",
				Text: "<@U123BOT> inc issue <@U123BOT> again",
			},
			expected: true, // First mention triggers
		},
		{
			name: "Bot mention in middle with inc",
			message: &model.Message{
				ID:   "msg-020",
				Text: "Hey <@U999USER> and <@U123BOT> inc database down",
			},
			expected: true,
		},
		// Test cases for word boundary checking
		{
			name: "Bot mention with incorrect (should be rejected)",
			message: &model.Message{
				ID:   "msg-021",
				Text: "<@U123BOT> incorrect assumption",
			},
			expected: false,
		},
		{
			name: "Bot mention with income (should be rejected)",
			message: &model.Message{
				ID:   "msg-022",
				Text: "<@U123BOT> income report",
			},
			expected: false,
		},
		{
			name: "Bot mention with incognito (should be rejected)",
			message: &model.Message{
				ID:   "msg-023",
				Text: "<@U123BOT> incognito mode",
			},
			expected: false,
		},
		{
			name: "Bot mention with just inc (should be accepted)",
			message: &model.Message{
				ID:   "msg-024",
				Text: "<@U123BOT> inc",
			},
			expected: true,
		},
		{
			name: "Bot mention with INC in caps (should be accepted)",
			message: &model.Message{
				ID:   "msg-025",
				Text: "<@U123BOT> INC",
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := uc.ParseIncidentCommand(ctx, tc.message)
			gt.Equal(t, tc.expected, result.IsIncidentTrigger)
			if tc.expectedTitle != "" {
				gt.Equal(t, tc.expectedTitle, result.Title)
			}
		})
	}
}
