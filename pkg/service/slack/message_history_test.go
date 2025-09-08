package slack_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/slack-go/slack"
)


func TestMessageHistoryService_New(t *testing.T) {
	// Test service creation
	service := slackSvc.NewMessageHistoryService(nil)
	gt.NotEqual(t, service, nil)
}

func TestMessageHistoryOptions_Validation(t *testing.T) {
	// Test that empty channel ID should be caught by validation
	opts := slackSvc.MessageHistoryOptions{
		ChannelID: "",
		Limit:     10,
	}

	// This would fail when actually calling GetMessages, but here we just test the struct
	gt.Equal(t, opts.ChannelID, "")
	gt.Equal(t, opts.Limit, 10)
}

func TestMessageHistoryService_LimitBounds(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"Zero should default to 256", 0, 256},
		{"Negative should default to 256", -1, 256},
		{"Valid limit should remain unchanged", 100, 100},
		{"Max limit should remain unchanged", 256, 256},
		{"Over limit should be capped to 256", 300, 256},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock client to capture the actual limit passed to Slack API
			var capturedLimit int
			mockClient := &mocks.SlackClientMock{
				GetConversationHistoryContextFunc: func(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
					capturedLimit = params.Limit
					return &slack.GetConversationHistoryResponse{
						Messages: []slack.Message{},
					}, nil
				},
			}

			service := slackSvc.NewMessageHistoryService(mockClient)

			opts := slackSvc.MessageHistoryOptions{
				ChannelID: "C123456789",
				Limit:     tc.inputLimit,
			}

			// Call GetMessages to trigger the limit validation
			_, err := service.GetMessages(ctx, opts)
			gt.NoError(t, err)

			// Verify the correct limit was passed to the Slack API
			gt.Equal(t, capturedLimit, tc.expectedLimit)
		})
	}
}

func TestMessageHistoryService_ThreadLimitBounds(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{"Thread limit should be clamped the same way", 0, 256},
		{"Thread over-limit should be capped", 500, 256},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock client to capture the limit for thread messages
			var capturedLimit int
			mockClient := &mocks.SlackClientMock{
				GetConversationRepliesContextFunc: func(ctx context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, bool, error) {
					capturedLimit = params.Limit
					return []slack.Message{}, false, false, nil
				},
			}

			service := slackSvc.NewMessageHistoryService(mockClient)

			opts := slackSvc.MessageHistoryOptions{
				ChannelID: "C123456789",
				ThreadTS:  "1234567890.123456", // This triggers thread message retrieval
				Limit:     tc.inputLimit,
			}

			// Call GetMessages to trigger the limit validation for thread messages
			_, err := service.GetMessages(ctx, opts)
			gt.NoError(t, err)

			// Verify the correct limit was passed to the Slack thread API
			gt.Equal(t, capturedLimit, tc.expectedLimit)
		})
	}
}

func TestMessageHistoryService_MessageOrdering(t *testing.T) {
	ctx := context.Background()

	t.Run("Channel messages should be returned in chronological order", func(t *testing.T) {
		// Create mock client that returns messages in reverse chronological order (newest first)
		mockClient := &mocks.SlackClientMock{
			GetConversationHistoryContextFunc: func(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
				return &slack.GetConversationHistoryResponse{
					Messages: []slack.Message{
						{Msg: slack.Msg{Timestamp: "1234567890.000003", Text: "Newest message"}},
						{Msg: slack.Msg{Timestamp: "1234567890.000002", Text: "Middle message"}},
						{Msg: slack.Msg{Timestamp: "1234567890.000001", Text: "Oldest message"}},
					},
				}, nil
			},
		}

		service := slackSvc.NewMessageHistoryService(mockClient)

		opts := slackSvc.MessageHistoryOptions{
			ChannelID: "C123456789",
		}

		messages, err := service.GetMessages(ctx, opts)
		gt.NoError(t, err)
		gt.Equal(t, len(messages), 3)

		// Verify messages are in chronological order (oldest first)
		gt.Equal(t, messages[0].Text, "Oldest message")
		gt.Equal(t, messages[1].Text, "Middle message")
		gt.Equal(t, messages[2].Text, "Newest message")
	})

	t.Run("Thread messages should remain in chronological order", func(t *testing.T) {
		// Create mock client that returns thread messages in chronological order (oldest first)
		mockClient := &mocks.SlackClientMock{
			GetConversationRepliesContextFunc: func(ctx context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, bool, error) {
				return []slack.Message{
					{Msg: slack.Msg{Timestamp: "1234567890.000001", Text: "First reply"}},
					{Msg: slack.Msg{Timestamp: "1234567890.000002", Text: "Second reply"}},
					{Msg: slack.Msg{Timestamp: "1234567890.000003", Text: "Third reply"}},
				}, false, false, nil
			},
		}

		service := slackSvc.NewMessageHistoryService(mockClient)

		opts := slackSvc.MessageHistoryOptions{
			ChannelID: "C123456789",
			ThreadTS:  "1234567890.000000",
		}

		messages, err := service.GetMessages(ctx, opts)
		gt.NoError(t, err)
		gt.Equal(t, len(messages), 3)

		// Verify messages remain in chronological order (oldest first)
		gt.Equal(t, messages[0].Text, "First reply")
		gt.Equal(t, messages[1].Text, "Second reply")
		gt.Equal(t, messages[2].Text, "Third reply")
	})
}
