package usecase_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// getTestCategories returns categories for testing purposes
func getTestCategories() *model.CategoriesConfig {
	return &model.CategoriesConfig{
		Categories: []model.Category{
			{
				ID:           "security_incident",
				Name:         "Security Incident",
				Description:  "Security-related incidents including unauthorized access and malware infections",
				InviteUsers:  []string{"@security-lead"},
				InviteGroups: []string{"@security-team"},
			},
			{
				ID:           "system_failure",
				Name:         "System Failure",
				Description:  "System or service failures and outages",
				InviteUsers:  []string{"@sre-lead"},
				InviteGroups: []string{"@sre-oncall"},
			},
			{
				ID:          "performance_issue",
				Name:        "Performance Issue",
				Description: "System performance degradation or response time issues",
			},
			{
				ID:          "unknown",
				Name:        "Unknown",
				Description: "Incidents that cannot be categorized",
			},
		},
	}
}

// Helper function to create default mock clients for testing
func createMockClients() (gollem.LLMClient, *mocks.SlackClientMock) {
	// Create default mock LLM client
	mockLLM := &mock.LLMClientMock{}

	// Create default mock Slack client with a test bot user
	mockSlack := &mocks.SlackClientMock{
		AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
			return &slack.AuthTestResponse{
				UserID: "U_TEST_BOT",
				User:   "test-bot",
			}, nil
		},
		GetConversationInfoFunc: func(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
			// Return basic channel info for testing
			channel := &slack.Channel{}
			channel.Name = "test-channel"
			channel.Topic = slack.Topic{Value: "Test channel"}
			channel.Purpose = slack.Purpose{Value: "Testing"}
			return channel, nil
		},
	}

	return mockLLM, mockSlack
}

// Test for the enhanced ParseIncidentCommand with LLM always being used
func TestParseIncidentCommand_AlwaysUsesLLM(t *testing.T) {
	tests := []struct {
		name                   string
		messageText            string
		channelID              string
		channelName            string
		channelTopic           string
		messageHistory         []slack.Message
		expectedPromptContains []string
		expectedLLMResponse    string
		expectChannelAPICall   bool
		expectLLMCall          bool
	}{
		{
			name:        "with title text - should use LLM with additional prompt",
			messageText: "<@U_TEST_BOT> inc database error",
			channelID:   "C123456",
			channelName: "production-alerts",
			channelTopic: "Production system monitoring",
			messageHistory: []slack.Message{
				{Msg: slack.Msg{Text: "システムが重い", User: "user1", Timestamp: "1234567890.123"}},
			},
			expectedPromptContains: []string{
				"**Channel Name**: production-alerts",
				"**Topic**: Production system monitoring",
				"database error", // additional prompt
				"システムが重い", // message history
			},
			expectedLLMResponse: `{"title":"データベース接続障害","description":"本番環境でデータベース接続エラーが発生","category_id":"system_failure"}`,
			expectChannelAPICall: true,
			expectLLMCall:        true,
		},
		{
			name:        "without title text - should use LLM with message history only",
			messageText: "<@U_TEST_BOT> inc",
			channelID:   "C123456",
			channelName: "api-team",
			channelTopic: "API development discussions",
			messageHistory: []slack.Message{
				{Msg: slack.Msg{Text: "APIが遅い", User: "user1", Timestamp: "1234567890.123"}},
				{Msg: slack.Msg{Text: "レスポンス時間が長い", User: "user2", Timestamp: "1234567891.456"}},
			},
			expectedPromptContains: []string{
				"**Channel Name**: api-team",
				"**Topic**: API development discussions",
				"APIが遅い",
				"レスポンス時間が長い",
			},
			expectedLLMResponse: `{"title":"API パフォーマンス問題","description":"APIレスポンス時間の遅延が発生","category_id":"performance_issue"}`,
			expectChannelAPICall: true,
			expectLLMCall:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx := context.Background()
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx = ctxlog.With(ctx, logger)
			repo := repository.NewMemory()

			// Mock LLM client with detailed verification
			var actualPrompt string
			mockLLM := &mock.LLMClientMock{
				NewSessionFunc: func(ctx context.Context, opts ...gollem.SessionOption) (gollem.Session, error) {
					// Verify session options
					gt.A(t, opts).Longer(0) // Should have content type option

					return &mock.SessionMock{
						GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
							// Verify prompt content
							gt.Equal(t, len(input), 1)
							textInput, ok := input[0].(gollem.Text)
							gt.True(t, ok)

							actualPrompt = string(textInput)
							for _, expectedContent := range tt.expectedPromptContains {
								if !strings.Contains(actualPrompt, expectedContent) {
									t.Logf("Prompt should contain: %s\nActual prompt: %s", expectedContent, actualPrompt)
									t.Fail()
								}
							}

							// Return mock response
							return &gollem.Response{
								Texts: []string{tt.expectedLLMResponse},
							}, nil
						},
					}, nil
				},
			}

			// Mock Slack client with channel info verification
			var channelAPICallCount int
			mockSlack := &mocks.SlackClientMock{
				AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
					return &slack.AuthTestResponse{UserID: "U_TEST_BOT", User: "test-bot"}, nil
				},
				GetConversationInfoFunc: func(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
					channelAPICallCount++
					gt.Equal(t, tt.channelID, channelID)
					gt.False(t, includeLocale) // Should be false

					channel := &slack.Channel{}
					channel.Name = tt.channelName
					channel.Topic = slack.Topic{Value: tt.channelTopic}
					channel.Purpose = slack.Purpose{Value: "Test channel purpose"}
					channel.IsPrivate = false
					channel.NumMembers = 10
					return channel, nil
				},
				GetConversationHistoryContextFunc: func(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
					return &slack.GetConversationHistoryResponse{
						Messages: tt.messageHistory,
					}, nil
				},
			}

			// Create UseCase with mocks
			categories := getTestCategories()
			slackMessage, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, categories)
			gt.NoError(t, err)

			// Create test message
			message := &model.Message{
				ID:        types.MessageID("test-msg-001"),
				UserID:    types.SlackUserID("user1"),
				ChannelID: types.ChannelID(tt.channelID),
				Text:      tt.messageText,
				EventTS:   types.EventTS("1234567890.123"),
			}

			// Execute
			result := slackMessage.ParseIncidentCommand(ctx, message)

			// Verify LLM call expectations
			if tt.expectLLMCall {
				gt.True(t, result.IsIncidentTrigger)
				gt.NotEqual(t, "", result.Title)
				gt.NotEqual(t, "", result.Description)
				gt.NotEqual(t, "", result.CategoryID)

				// Verify LLM client was called
				gt.Equal(t, 1, len(mockLLM.NewSessionCalls()))
			} else {
				gt.False(t, result.IsIncidentTrigger)
			}

			// Verify channel API call expectations
			if tt.expectChannelAPICall {
				gt.Equal(t, 1, channelAPICallCount)
			} else {
				gt.Equal(t, 0, channelAPICallCount)
			}
		})
	}
}

func TestSlackMessageProcessMessage(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	// Create mock clients
	mockGollem, mockSlack := createMockClients()

	uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack, getTestCategories())
	gt.NoError(t, err).Required()

	// Use random IDs as per CLAUDE.md
	msgID := fmt.Sprintf("msg-test-%d", time.Now().UnixNano()%1000000)
	userID := fmt.Sprintf("U%d", time.Now().UnixNano()%1000000)
	channelID := fmt.Sprintf("C%d", time.Now().UnixNano()%1000000)

	event := &slackevents.MessageEvent{
		ClientMsgID:     msgID,
		User:            userID,
		Username:        "testuser",
		Channel:         channelID,
		Text:            "Test message",
		TimeStamp:       "1234567890.123456",
		ThreadTimeStamp: "",
	}

	err = uc.ProcessMessage(ctx, event)
	gt.NoError(t, err).Required()

	// Verify message was saved
	saved, err := repo.GetMessage(ctx, types.MessageID(msgID))
	gt.NoError(t, err).Required()
	gt.Equal(t, msgID, saved.ID.String())
	gt.Equal(t, "Test message", saved.Text)
	gt.Equal(t, userID, saved.UserID.String())
	gt.Equal(t, channelID, saved.ChannelID.String())
}

func TestSlackMessageGenerateResponse(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()

	t.Run("With LLM client", func(t *testing.T) {
		// Create mock LLM client using gollem's built-in mock
		mockLLM := &mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				mockSession := &mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{"Generated response"},
						}, nil
					},
				}
				return mockSession, nil
			},
		}

		// Create mock clients
		_, mockSlack := createMockClients()

		uc, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, getTestCategories())
		gt.NoError(t, err).Required()

		message := &model.Message{
			ID:   types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
			Text: "Help me with incident",
		}

		response, err := uc.GenerateResponse(ctx, message)
		gt.NoError(t, err).Required()
		gt.Equal(t, "Generated response", response)
	})

	t.Run("With LLM error fallback", func(t *testing.T) {
		// Test what happens when LLM generation fails - should get a fallback response
		mockLLM := &mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				return nil, fmt.Errorf("LLM service unavailable")
			},
		}
		_, mockSlack := createMockClients()

		uc, err := usecase.NewSlackMessage(ctx, repo, mockLLM, mockSlack, getTestCategories())
		gt.NoError(t, err).Required()

		message := &model.Message{
			ID:   types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
			Text: "Help me with incident",
		}

		response, err := uc.GenerateResponse(ctx, message)
		gt.NoError(t, err).Required()
		gt.Equal(t, "I understand your message. Let me help you with that.", response)
	})
}

func TestSlackMessageSaveAndRespond(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	// Create mock gollem client using gollem's built-in mock
	mockGollem := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"Here's help with your incident"},
					}, nil
				},
			}
			return mockSession, nil
		},
	}

	// Create mock slack client
	_, mockSlack := createMockClients()

	uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack, getTestCategories())
	gt.NoError(t, err).Required()

	t.Run("With ClientMsgID", func(t *testing.T) {
		msgID := fmt.Sprintf("msg-test-%d", time.Now().UnixNano()%1000000)
		userID := fmt.Sprintf("U%d", time.Now().UnixNano()%1000000)
		channelID := fmt.Sprintf("C%d", time.Now().UnixNano()%1000000)

		event := &slackevents.MessageEvent{
			ClientMsgID:     msgID,
			User:            userID,
			Username:        "helpuser",
			Channel:         channelID,
			Text:            "I need help with an incident",
			TimeStamp:       "1234567890.654321",
			ThreadTimeStamp: "",
		}

		response, err := uc.SaveAndRespond(ctx, event)
		gt.NoError(t, err).Required()
		gt.Equal(t, "", response) // No response for general mentions

		// Verify message was saved with ClientMsgID
		saved, err := repo.GetMessage(ctx, types.MessageID(msgID))
		gt.NoError(t, err).Required()
		gt.Equal(t, "I need help with an incident", saved.Text)
	})

	t.Run("Without ClientMsgID (app mention)", func(t *testing.T) {
		userID := fmt.Sprintf("U%d", time.Now().UnixNano()%1000000)
		channelID := fmt.Sprintf("C%d", time.Now().UnixNano()%1000000)
		botID := fmt.Sprintf("U%dBOT", time.Now().UnixNano()%1000000)
		timestamp := fmt.Sprintf("%d.123456", time.Now().Unix())

		event := &slackevents.MessageEvent{
			ClientMsgID:     "", // Empty for app mentions
			User:            userID,
			Username:        "mentionuser",
			Channel:         channelID,
			Text:            fmt.Sprintf("<@%s> inc server is down", botID),
			TimeStamp:       timestamp,
			ThreadTimeStamp: "",
		}

		response, err := uc.SaveAndRespond(ctx, event)
		gt.NoError(t, err).Required()
		gt.Equal(t, "", response) // No response for general mentions

		// Verify message was saved with TimeStamp as ID
		saved, err := repo.GetMessage(ctx, types.MessageID(timestamp))
		gt.NoError(t, err).Required()
		gt.Equal(t, fmt.Sprintf("<@%s> inc server is down", botID), saved.Text)
	})
}

func TestSlackMessageParseIncidentCommand(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()

	// Create mock Slack client that returns a specific bot user ID
	mockSlack := &mocks.SlackClientMock{
		AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
			return &slack.AuthTestResponse{
				UserID: "U123BOT",
				User:   "lycaon-bot",
			}, nil
		},
		GetConversationInfoFunc: func(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
			// Return basic channel info for testing
			channel := &slack.Channel{}
			channel.Name = "test-channel"
			channel.Topic = slack.Topic{Value: "Test channel"}
			channel.Purpose = slack.Purpose{Value: "Testing"}
			return channel, nil
		},
	}

	// Create mock LLM client
	mockGollem, _ := createMockClients()

	uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack, getTestCategories())
	gt.NoError(t, err).Required()

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
		// Additional edge cases for token-based parsing
		{
			name: "Multiple bot mentions, inc after second",
			message: &model.Message{
				ID:   "msg-026",
				Text: "Hey <@U123BOT> can you help? Later <@U123BOT> inc database issue",
			},
			expected: true, // Should find inc after second mention
		},
		{
			name: "Bot mention with punctuation then inc",
			message: &model.Message{
				ID:   "msg-027",
				Text: "<@U123BOT>, inc production issue",
			},
			expected: false, // Comma creates separate token
		},
		{
			name: "Bot mention with newline before inc",
			message: &model.Message{
				ID:   "msg-028",
				Text: "<@U123BOT>\ninc server down",
			},
			expected: true, // Newline is whitespace, tokens are adjacent
		},
		{
			name: "Bot substring in user ID (edge case)",
			message: &model.Message{
				ID:   "msg-029",
				Text: "<@U123BOTXXX> inc something",
			},
			expected: false, // Different user ID, not exact match
		},
		{
			name: "Bot mention followed by inc in quotes",
			message: &model.Message{
				ID:   "msg-030",
				Text: `<@U123BOT> "inc" the system`,
			},
			expected: false, // Quotes create separate token
		},
		// Title extraction test cases
		{
			name: "Bot mention inc with simple title",
			message: &model.Message{
				ID:   "msg-031",
				Text: "<@U123BOT> inc database is down",
			},
			expected:      true,
			expectedTitle: "database is down",
		},
		{
			name: "Bot mention inc with multi-word title",
			message: &model.Message{
				ID:   "msg-032",
				Text: "<@U123BOT> inc urgent production database connection timeout issue",
			},
			expected:      true,
			expectedTitle: "urgent production database connection timeout issue",
		},
		{
			name: "Bot mention inc with no title",
			message: &model.Message{
				ID:   "msg-033",
				Text: "<@U123BOT> inc",
			},
			expected:      true,
			expectedTitle: "",
		},
		{
			name: "Multiple bot mentions, title after second inc",
			message: &model.Message{
				ID:   "msg-034",
				Text: "First <@U123BOT> hello, then <@U123BOT> inc server crashed",
			},
			expected:      true,
			expectedTitle: "server crashed",
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

// TestSlackMessageLLMIntegration tests the LLM-enhanced incident creation functionality
func TestSlackMessageLLMIntegration(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()

	// Use random IDs as per CLAUDE.md
	botUserID := fmt.Sprintf("U%dBOT", time.Now().UnixNano()%1000000)

	t.Run("LLM enhancement with successful generation", func(t *testing.T) {
		// Create mock gollem client that returns structured JSON
		mockGollem := &mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				mockSession := &mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{`{
								"title": "Database Connection Timeout",
								"description": "Multiple users reporting database connection timeouts affecting user login and data retrieval operations."
							}`},
						}, nil
					},
				}
				return mockSession, nil
			},
		}

		// Create mock slack client for message history
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
			},
			GetConversationInfoFunc: func(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
				// Return basic channel info for testing
				channel := &slack.Channel{}
				channel.Name = "test-channel"
				channel.Topic = slack.Topic{Value: "Test channel"}
				channel.Purpose = slack.Purpose{Value: "Testing"}
				return channel, nil
			},
			GetConversationHistoryContextFunc: func(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
				return &slack.GetConversationHistoryResponse{
					Messages: []slack.Message{
						{
							Msg: slack.Msg{
								Timestamp: fmt.Sprintf("%d.000001", time.Now().Unix()),
								User:      fmt.Sprintf("U%d", time.Now().UnixNano()%1000000),
								Text:      "Database is really slow",
							},
						},
						{
							Msg: slack.Msg{
								Timestamp: fmt.Sprintf("%d.000002", time.Now().Unix()),
								User:      fmt.Sprintf("U%d", time.Now().UnixNano()%1000000),
								Text:      "I'm getting timeout errors on login",
							},
						},
					},
				}, nil
			},
		}

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack, getTestCategories())
		gt.NoError(t, err).Required()

		// Test parsing incident command without title (should trigger LLM enhancement)
		message := &model.Message{
			ID:        types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
			Text:      fmt.Sprintf("<@%s> inc", botUserID),
			ChannelID: types.ChannelID(fmt.Sprintf("C%d", time.Now().UnixNano()%1000000)),
			UserID:    types.SlackUserID(fmt.Sprintf("U%d", time.Now().UnixNano()%1000000)),
			Timestamp: time.Now(),
		}

		result := uc.ParseIncidentCommand(ctx, message)
		gt.Equal(t, true, result.IsIncidentTrigger)
		gt.Equal(t, "Database Connection Timeout", result.Title)
		gt.Equal(t, "Multiple users reporting database connection timeouts affecting user login and data retrieval operations.", result.Description)
	})

	t.Run("LLM enhancement with thread messages", func(t *testing.T) {
		// Mock LLM client that returns incident summary based on thread
		mockGollem := &mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				mockSession := &mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{`{
								"title": "API Service Outage",
								"description": "Complete API service outage causing 500 errors across all endpoints, affecting user authentication and data access."
							}`},
						}, nil
					},
				}
				return mockSession, nil
			},
		}

		// Mock slack client for thread replies
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
			},
			GetConversationInfoFunc: func(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
				// Return basic channel info for testing
				channel := &slack.Channel{}
				channel.Name = "test-channel"
				channel.Topic = slack.Topic{Value: "Test channel"}
				channel.Purpose = slack.Purpose{Value: "Testing"}
				return channel, nil
			},
			GetConversationRepliesContextFunc: func(ctx context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, bool, error) {
				return []slack.Message{
					{
						Msg: slack.Msg{
							Timestamp: fmt.Sprintf("%d.000001", time.Now().Unix()),
							User:      fmt.Sprintf("U%d", time.Now().UnixNano()%1000000),
							Text:      "API is returning 500 errors",
						},
					},
					{
						Msg: slack.Msg{
							Timestamp: fmt.Sprintf("%d.000002", time.Now().Unix()),
							User:      fmt.Sprintf("U%d", time.Now().UnixNano()%1000000),
							Text:      "All authentication is failing",
						},
					},
				}, false, false, nil
			},
		}

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack, getTestCategories())
		gt.NoError(t, err).Required()

		// Test with thread timestamp (should use thread messages)
		threadTS := fmt.Sprintf("%d.000000", time.Now().Unix())
		message := &model.Message{
			ID:        types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
			Text:      fmt.Sprintf("<@%s> inc", botUserID),
			ChannelID: types.ChannelID(fmt.Sprintf("C%d", time.Now().UnixNano()%1000000)),
			UserID:    types.SlackUserID(fmt.Sprintf("U%d", time.Now().UnixNano()%1000000)),
			ThreadTS:  types.ThreadTS(threadTS),
			Timestamp: time.Now(),
		}

		result := uc.ParseIncidentCommand(ctx, message)
		gt.Equal(t, true, result.IsIncidentTrigger)
		gt.Equal(t, "API Service Outage", result.Title)
		gt.Equal(t, "Complete API service outage causing 500 errors across all endpoints, affecting user authentication and data access.", result.Description)
	})

	t.Run("LLM enhancement failure fallback", func(t *testing.T) {
		// Mock LLM client that returns an error
		mockGollem := &mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				mockSession := &mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return nil, fmt.Errorf("LLM service unavailable")
					},
				}
				return mockSession, nil
			},
		}

		// Mock slack client that returns some messages
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
			},
			GetConversationInfoFunc: func(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
				// Return basic channel info for testing
				channel := &slack.Channel{}
				channel.Name = "test-channel"
				channel.Topic = slack.Topic{Value: "Test channel"}
				channel.Purpose = slack.Purpose{Value: "Testing"}
				return channel, nil
			},
			GetConversationHistoryContextFunc: func(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
				return &slack.GetConversationHistoryResponse{
					Messages: []slack.Message{
						{
							Msg: slack.Msg{
								Timestamp: fmt.Sprintf("%d.000001", time.Now().Unix()),
								User:      fmt.Sprintf("U%d", time.Now().UnixNano()%1000000),
								Text:      "Something is broken",
							},
						},
					},
				}, nil
			},
		}

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack, getTestCategories())
		gt.NoError(t, err).Required()

		message := &model.Message{
			ID:        types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
			Text:      fmt.Sprintf("<@%s> inc", botUserID),
			ChannelID: types.ChannelID(fmt.Sprintf("C%d", time.Now().UnixNano()%1000000)),
			UserID:    types.SlackUserID(fmt.Sprintf("U%d", time.Now().UnixNano()%1000000)),
			Timestamp: time.Now(),
		}

		// Should still work but without LLM enhancement (fallback to manual input)
		result := uc.ParseIncidentCommand(ctx, message)
		gt.Equal(t, true, result.IsIncidentTrigger)
		gt.Equal(t, "", result.Title)       // Should be empty when LLM fails
		gt.Equal(t, "", result.Description) // Should be empty when LLM fails
	})

	t.Run("LLM enhancement with invalid JSON response", func(t *testing.T) {
		// Mock LLM client that returns invalid JSON
		mockGollem := &mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				mockSession := &mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{"This is not valid JSON"},
						}, nil
					},
				}
				return mockSession, nil
			},
		}

		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
			},
			GetConversationInfoFunc: func(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
				// Return basic channel info for testing
				channel := &slack.Channel{}
				channel.Name = "test-channel"
				channel.Topic = slack.Topic{Value: "Test channel"}
				channel.Purpose = slack.Purpose{Value: "Testing"}
				return channel, nil
			},
			GetConversationHistoryContextFunc: func(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
				return &slack.GetConversationHistoryResponse{
					Messages: []slack.Message{
						{
							Msg: slack.Msg{
								Timestamp: fmt.Sprintf("%d.000001", time.Now().Unix()),
								User:      fmt.Sprintf("U%d", time.Now().UnixNano()%1000000),
								Text:      "Server issue",
							},
						},
					},
				}, nil
			},
		}

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack, getTestCategories())
		gt.NoError(t, err).Required()

		message := &model.Message{
			ID:        types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
			Text:      fmt.Sprintf("<@%s> inc", botUserID),
			ChannelID: types.ChannelID(fmt.Sprintf("C%d", time.Now().UnixNano()%1000000)),
			UserID:    types.SlackUserID(fmt.Sprintf("U%d", time.Now().UnixNano()%1000000)),
			Timestamp: time.Now(),
		}

		// Should fallback gracefully when JSON parsing fails
		result := uc.ParseIncidentCommand(ctx, message)
		gt.Equal(t, true, result.IsIncidentTrigger)
		gt.Equal(t, "", result.Title)       // Should fallback to empty
		gt.Equal(t, "", result.Description) // Should fallback to empty
	})

	t.Run("Manual title specified (LLM enhancement with additional prompt)", func(t *testing.T) {
		// Mock LLM client that will be called with additional prompt
		mockGollem := &mock.LLMClientMock{
			NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
				mockSession := &mock.SessionMock{
					GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
						return &gollem.Response{
							Texts: []string{`{
								"title": "Enhanced Manual Incident Title",
								"description": "LLM-enhanced description based on manual title and message history",
								"category_id": "system_failure"
							}`},
						}, nil
					},
				}
				return mockSession, nil
			},
		}
		mockSlack := &mocks.SlackClientMock{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
			},
			GetConversationInfoFunc: func(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
				// Return basic channel info for testing
				channel := &slack.Channel{}
				channel.Name = "test-channel"
				channel.Topic = slack.Topic{Value: "Test channel"}
				channel.Purpose = slack.Purpose{Value: "Testing"}
				return channel, nil
			},
			GetConversationHistoryContextFunc: func(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
				return &slack.GetConversationHistoryResponse{
					Messages: []slack.Message{
						{
							Msg: slack.Msg{
								Timestamp: fmt.Sprintf("%d.000001", time.Now().Unix()),
								User:      fmt.Sprintf("U%d", time.Now().UnixNano()%1000000),
								Text:      "Manual incident title test",
							},
						},
					},
				}, nil
			},
		}

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack, getTestCategories())
		gt.NoError(t, err).Required()

		// With manual title - should trigger LLM with title as additional prompt
		message := &model.Message{
			ID:        types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
			Text:      fmt.Sprintf("<@%s> inc Manual Incident Title", botUserID),
			ChannelID: types.ChannelID(fmt.Sprintf("C%d", time.Now().UnixNano()%1000000)),
			UserID:    types.SlackUserID(fmt.Sprintf("U%d", time.Now().UnixNano()%1000000)),
			Timestamp: time.Now(),
		}

		result := uc.ParseIncidentCommand(ctx, message)
		gt.Equal(t, true, result.IsIncidentTrigger)
		gt.Equal(t, "Enhanced Manual Incident Title", result.Title) // LLM enhanced title
		gt.Equal(t, "LLM-enhanced description based on manual title and message history", result.Description) // LLM generated description
		gt.Equal(t, "system_failure", result.CategoryID) // LLM selected category
	})
}
