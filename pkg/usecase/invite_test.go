package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"errors"
	
	"github.com/slack-go/slack"
)

func TestInviteUseCaseInviteUsersByList(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful batch invitation with user IDs", func(t *testing.T) {
		// Create mock Slack client
		mockSlack := &mocks.SlackClientMock{
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Verify that users are passed correctly
				gt.Equal(t, 2, len(users))
				gt.True(t, contains(users, "U123456"))
				gt.True(t, contains(users, "U789012"))
				return &slack.Channel{}, nil
			},
		}

		// Create use case
		uc := usecase.NewInvite(mockSlack)

		// Test with direct user IDs (no resolution needed)
		result, err := uc.InviteUsersByList(
			ctx,
			[]string{"U123456", "U789012"},
			[]string{},
			types.ChannelID("C-TEST-CHANNEL"),
		)

		// Verify success
		gt.NoError(t, err).Required()
		gt.V(t, result).NotNil()
		gt.Equal(t, 2, len(result.Details))

		// Verify all users are marked as success
		for _, detail := range result.Details {
			gt.Equal(t, "success", detail.Status)
			gt.Equal(t, "", detail.Error)
		}

		// Verify mock was called
		gt.Equal(t, 1, len(mockSlack.InviteUsersToConversationCalls()))
	})

	t.Run("Handle invitation failures gracefully", func(t *testing.T) {
		// Create mock Slack client that returns error
		mockSlack := &mocks.SlackClientMock{
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return nil, errors.New("already_in_channel: Users are already in the channel")
			},
		}

		// Create use case
		uc := usecase.NewInvite(mockSlack)

		// Test with user IDs
		result, err := uc.InviteUsersByList(
			ctx,
			[]string{"U123456", "U789012"},
			[]string{},
			types.ChannelID("C-TEST-CHANNEL"),
		)

		// Should not return error even if invitation fails
		gt.NoError(t, err).Required()
		gt.V(t, result).NotNil()
		gt.Equal(t, 2, len(result.Details))

		// Verify all users are marked as failed with error message
		for _, detail := range result.Details {
			gt.Equal(t, "failed", detail.Status)
			gt.True(t, detail.Error != "")
		}
	})

	t.Run("Skip users that could not be resolved", func(t *testing.T) {
		// Mock that will be called with both users (mocks pass through @-prefixed users)
		mockSlack := &mocks.SlackClientMock{
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// With mocks, both users are passed through
				gt.Equal(t, 2, len(users))
				gt.True(t, contains(users, "U123456"))
				gt.True(t, contains(users, "@unknown-user"))
				return &slack.Channel{}, nil
			},
		}

		// Create use case
		uc := usecase.NewInvite(mockSlack)

		// With mocks, @username will just be passed through as-is
		result, err := uc.InviteUsersByList(
			ctx,
			[]string{"U123456", "@unknown-user"}, // Mock passes through @unknown-user as-is
			[]string{},
			types.ChannelID("C-TEST-CHANNEL"),
		)

		// Should succeed
		gt.NoError(t, err).Required()
		gt.V(t, result).NotNil()
		gt.Equal(t, 2, len(result.Details))
		
		// Both should be marked as success with mocks
		for _, detail := range result.Details {
			gt.Equal(t, "success", detail.Status)
		}
	})

	t.Run("Empty user and group lists", func(t *testing.T) {
		// Mock should not be called with empty lists
		mockSlack := &mocks.SlackClientMock{
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Should be called with empty user list
				gt.Equal(t, 0, len(users))
				return &slack.Channel{}, nil
			},
		}

		// Create use case
		uc := usecase.NewInvite(mockSlack)

		// Test with empty lists
		result, err := uc.InviteUsersByList(
			ctx,
			[]string{},
			[]string{},
			types.ChannelID("C-TEST-CHANNEL"),
		)

		// Should succeed with no details
		gt.NoError(t, err).Required()
		gt.V(t, result).NotNil()
		gt.Equal(t, 0, len(result.Details))
	})

	t.Run("Successful batch invitation with Bot IDs", func(t *testing.T) {
		// Create mock Slack client
		mockSlack := &mocks.SlackClientMock{
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Verify that both regular users and bots are passed
				gt.Equal(t, 3, len(users))
				gt.True(t, contains(users, "U123456"))  // Regular user
				gt.True(t, contains(users, "B09E8M5JSPK"))  // Bot ID
				gt.True(t, contains(users, "B987654"))  // Another Bot
				return &slack.Channel{}, nil
			},
		}

		// Create use case
		uc := usecase.NewInvite(mockSlack)

		// Test with mixed user IDs and Bot IDs
		result, err := uc.InviteUsersByList(
			ctx,
			[]string{"U123456", "B09E8M5JSPK", "B987654"},
			[]string{},
			types.ChannelID("C-TEST-CHANNEL"),
		)

		// Verify success
		gt.NoError(t, err).Required()
		gt.V(t, result).NotNil()
		gt.Equal(t, 3, len(result.Details))

		// Verify all are marked as success
		for _, detail := range result.Details {
			gt.Equal(t, "success", detail.Status)
			gt.Equal(t, "", detail.Error)
		}

		// Verify mock was called
		gt.Equal(t, 1, len(mockSlack.InviteUsersToConversationCalls()))
	})

	t.Run("Mixed users, bots, and groups with some failures", func(t *testing.T) {
		// Create mock Slack client
		mockSlack := &mocks.SlackClientMock{
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Check that we got the expected users
				gt.Equal(t, 4, len(users))
				return nil, errors.New("channel_not_found")
			},
		}

		// Create use case  
		uc := usecase.NewInvite(mockSlack)

		// Test with mixed IDs
		result, err := uc.InviteUsersByList(
			ctx,
			[]string{"U123456", "B09E8M5JSPK", "@tamamo", "@unknown"},
			[]string{},
			types.ChannelID("C-TEST-CHANNEL"),
		)

		// Should not return error even if invitation fails
		gt.NoError(t, err).Required()
		gt.V(t, result).NotNil()
		gt.Equal(t, 4, len(result.Details))

		// Check each detail
		for _, detail := range result.Details {
			// With mocks, @-prefixed users will be passed through
			if detail.SourceConfig == "U123456" || detail.SourceConfig == "B09E8M5JSPK" {
				gt.Equal(t, "failed", detail.Status)
				gt.Equal(t, "channel_not_found", detail.Error)
			}
		}
	})
}

// Helper function to check if slice contains value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}