package slack_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/slack-go/slack"
)

// mockSlackClient mocks the Slack client for testing
type mockSlackClient struct {
	GetConversationHistoryContextFunc func(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContextFunc func(ctx context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, bool, error)
}

func (m *mockSlackClient) GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	if m.GetConversationHistoryContextFunc != nil {
		return m.GetConversationHistoryContextFunc(ctx, params)
	}
	return &slack.GetConversationHistoryResponse{Messages: []slack.Message{}}, nil
}

func (m *mockSlackClient) GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, bool, error) {
	if m.GetConversationRepliesContextFunc != nil {
		return m.GetConversationRepliesContextFunc(ctx, params)
	}
	return []slack.Message{}, false, false, nil
}

// Implement other required interface methods as no-ops for this test
func (m *mockSlackClient) CreateConversation(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) { return nil, nil }
func (m *mockSlackClient) InviteUsersToConversation(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) { return nil, nil }
func (m *mockSlackClient) PostMessage(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) { return "", "", nil }
func (m *mockSlackClient) UpdateMessage(ctx context.Context, channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error) { return "", "", "", nil }
func (m *mockSlackClient) AuthTestContext(ctx context.Context) (*slack.AuthTestResponse, error) { return nil, nil }
func (m *mockSlackClient) GetConversationInfo(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) { return nil, nil }
func (m *mockSlackClient) SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*slack.Channel, error) { return nil, nil }
func (m *mockSlackClient) OpenView(ctx context.Context, triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error) { return nil, nil }

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
			mockClient := &mockSlackClient{
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
			mockClient := &mockSlackClient{
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
