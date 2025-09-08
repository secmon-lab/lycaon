package usecase_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func TestSlackMessageProcessMessage(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	// Create mock gollem client using gollem's built-in mock
	mockGollem := &mock.LLMClientMock{}

	uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, nil)
	gt.NoError(t, err)

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
	gt.NoError(t, err)

	// Verify message was saved
	saved, err := repo.GetMessage(ctx, types.MessageID(msgID))
	gt.NoError(t, err)
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

		uc, err := usecase.NewSlackMessage(ctx, repo, mockLLM, nil)
		gt.NoError(t, err)

		message := &model.Message{
			ID:   types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
			Text: "Help me with incident",
		}

		response, err := uc.GenerateResponse(ctx, message)
		gt.NoError(t, err)
		gt.Equal(t, "Generated response", response)
	})

	t.Run("Without LLM client", func(t *testing.T) {
		uc, err := usecase.NewSlackMessage(ctx, repo, nil, nil)
		gt.NoError(t, err)

		message := &model.Message{
			ID:   types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
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

	uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, nil)
	gt.NoError(t, err)

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
		gt.NoError(t, err)
		gt.Equal(t, "", response) // No response for general mentions

		// Verify message was saved with ClientMsgID
		saved, err := repo.GetMessage(ctx, types.MessageID(msgID))
		gt.NoError(t, err)
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
		gt.NoError(t, err)
		gt.Equal(t, "", response) // No response for general mentions

		// Verify message was saved with TimeStamp as ID
		saved, err := repo.GetMessage(ctx, types.MessageID(timestamp))
		gt.NoError(t, err)
		gt.Equal(t, fmt.Sprintf("<@%s> inc server is down", botID), saved.Text)
	})
}

func TestSlackMessageParseIncidentCommand(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()

	// Create mock Slack client that returns a specific bot user ID
	mockSlack := &MockSlackClient{
		AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
			return &slack.AuthTestResponse{
				UserID: "U123BOT",
				User:   "lycaon-bot",
			}, nil
		},
	}

	uc, err := usecase.NewSlackMessage(ctx, repo, nil, mockSlack)
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
		mockSlack := &MockSlackClient{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
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

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack)
		gt.NoError(t, err)

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
		mockSlack := &MockSlackClient{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
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

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack)
		gt.NoError(t, err)

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
		mockSlack := &MockSlackClient{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
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

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack)
		gt.NoError(t, err)

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

		mockSlack := &MockSlackClient{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
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

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack)
		gt.NoError(t, err)

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

	t.Run("Manual title specified (no LLM enhancement)", func(t *testing.T) {
		// Mock that should not be called since title is provided
		mockGollem := &mock.LLMClientMock{}
		mockSlack := &MockSlackClient{
			AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
				return &slack.AuthTestResponse{
					UserID: botUserID,
					User:   fmt.Sprintf("bot-%d", time.Now().UnixNano()%1000000),
				}, nil
			},
		}

		uc, err := usecase.NewSlackMessage(ctx, repo, mockGollem, mockSlack)
		gt.NoError(t, err)

		// With manual title - should not trigger LLM
		message := &model.Message{
			ID:        types.MessageID(fmt.Sprintf("msg-%d", time.Now().UnixNano()%1000000)),
			Text:      fmt.Sprintf("<@%s> inc Manual Incident Title", botUserID),
			ChannelID: types.ChannelID(fmt.Sprintf("C%d", time.Now().UnixNano()%1000000)),
			UserID:    types.SlackUserID(fmt.Sprintf("U%d", time.Now().UnixNano()%1000000)),
			Timestamp: time.Now(),
		}

		result := uc.ParseIncidentCommand(ctx, message)
		gt.Equal(t, true, result.IsIncidentTrigger)
		gt.Equal(t, "Manual Incident Title", result.Title)
		gt.Equal(t, "", result.Description) // No description when manually specified
	})
}
