package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/usecase"
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

	t.Run("Resolve @username to user ID", func(t *testing.T) {
		// Create mock Slack client that returns users when GetUsersContext is called
		mockSlack := &mocks.SlackClientMock{
			GetUsersContextFunc: func(ctx context.Context) ([]slack.User, error) {
				return []slack.User{
					{
						ID:       "U111111",
						Name:     "alice",
						RealName: "Alice Smith",
						IsBot:    false,
					},
					{
						ID:       "U222222",
						Name:     "bob",
						RealName: "Bob Jones",
						IsBot:    false,
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Should receive resolved user IDs
				gt.Equal(t, 2, len(users))
				gt.True(t, contains(users, "U111111"))
				gt.True(t, contains(users, "U222222"))
				return &slack.Channel{}, nil
			},
		}

		// Create use case
		uc := usecase.NewInvite(mockSlack)

		// Test with @usernames that need resolution
		result, err := uc.InviteUsersByList(
			ctx,
			[]string{"@alice", "@bob"},
			[]string{},
			types.ChannelID("C-TEST-CHANNEL"),
		)

		// Verify success
		gt.NoError(t, err).Required()
		gt.V(t, result).NotNil()
		gt.Equal(t, 2, len(result.Details))

		// Verify resolution worked
		for _, detail := range result.Details {
			gt.Equal(t, "success", detail.Status)
			if detail.SourceConfig == "@alice" {
				gt.Equal(t, "U111111", detail.UserID)
			} else if detail.SourceConfig == "@bob" {
				gt.Equal(t, "U222222", detail.UserID)
			}
		}

		// Verify GetUsersContext was called for resolution
		gt.Equal(t, 2, len(mockSlack.GetUsersContextCalls()))
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
				// Only U123456 should be invited since @unknown-user cannot be resolved
				gt.Equal(t, 1, len(users))
				gt.True(t, contains(users, "U123456"))
				return &slack.Channel{}, nil
			},
			GetUsersContextFunc: func(ctx context.Context) ([]slack.User, error) {
				// Return empty list to simulate user not found
				return []slack.User{}, nil
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
		
		// Check status of each user
		for _, detail := range result.Details {
			if detail.SourceConfig == "U123456" {
				gt.Equal(t, "success", detail.Status)
			} else if detail.SourceConfig == "@unknown-user" {
				gt.Equal(t, "failed", detail.Status)
				gt.True(t, detail.Error != "")
			}
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

	t.Run("Resolve Bot @username to user ID", func(t *testing.T) {
		// Test that bot names are properly resolved to user IDs
		mockSlack := &mocks.SlackClientMock{
			GetUsersContextFunc: func(ctx context.Context) ([]slack.User, error) {
				return []slack.User{
					{
						ID:       "UBOT123",
						Name:     "tamamo",
						RealName: "Tamamo Bot",
						IsBot:    true,
						Profile: slack.UserProfile{
							BotID:       "B09E8M5JSPK",
							DisplayName: "tamamo",
						},
					},
					{
						ID:       "UBOT456",
						Name:     "alertbot",
						RealName: "Alert Bot",
						IsBot:    true,
						Profile: slack.UserProfile{
							BotID:       "B987654",
							DisplayName: "Alert Bot",
						},
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Should receive user IDs of bots
				gt.Equal(t, 2, len(users))
				gt.True(t, contains(users, "UBOT123"))
				gt.True(t, contains(users, "UBOT456"))
				return &slack.Channel{}, nil
			},
		}

		uc := usecase.NewInvite(mockSlack)

		result, err := uc.InviteUsersByList(
			ctx,
			[]string{"@tamamo", "@alertbot"},
			[]string{},
			types.ChannelID("C-TEST-CHANNEL"),
		)

		gt.NoError(t, err).Required()
		gt.V(t, result).NotNil()
		gt.Equal(t, 2, len(result.Details))

		for _, detail := range result.Details {
			gt.Equal(t, "success", detail.Status)
			if detail.SourceConfig == "@tamamo" {
				gt.Equal(t, "UBOT123", detail.UserID)
			} else if detail.SourceConfig == "@alertbot" {
				gt.Equal(t, "UBOT456", detail.UserID)
			}
		}
	})

	t.Run("Resolve Bot ID to User ID", func(t *testing.T) {
		// Test that Bot IDs (B-prefix) are resolved to User IDs (U-prefix)
		mockSlack := &mocks.SlackClientMock{
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Should receive user IDs, not bot IDs
				gt.Equal(t, 3, len(users))
				gt.True(t, contains(users, "U123456"))  // Regular user
				gt.True(t, contains(users, "UBOT1"))    // Resolved from B09E8M5JSPK
				gt.True(t, contains(users, "UBOT2"))    // Resolved from B987654
				return &slack.Channel{}, nil
			},
			GetUsersContextFunc: func(ctx context.Context) ([]slack.User, error) {
				// Return bot users for Bot ID resolution
				return []slack.User{
					{
						ID:    "UBOT1",
						Name:  "bot1",
						IsBot: true,
						Profile: slack.UserProfile{
							BotID: "B09E8M5JSPK",
						},
					},
					{
						ID:    "UBOT2",
						Name:  "bot2",
						IsBot: true,
						Profile: slack.UserProfile{
							BotID: "B987654",
						},
					},
				}, nil
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

		// Verify all are marked as success and Bot IDs are resolved
		for _, detail := range result.Details {
			gt.Equal(t, "success", detail.Status)
			gt.Equal(t, "", detail.Error)
			if detail.SourceConfig == "B09E8M5JSPK" {
				gt.Equal(t, "UBOT1", detail.UserID)
			} else if detail.SourceConfig == "B987654" {
				gt.Equal(t, "UBOT2", detail.UserID)
			}
		}

		// Verify mock was called
		gt.Equal(t, 1, len(mockSlack.InviteUsersToConversationCalls()))
	})

	t.Run("Resolve user groups", func(t *testing.T) {
		// Test group resolution
		mockSlack := &mocks.SlackClientMock{
			GetUserGroupsContextFunc: func(ctx context.Context) ([]slack.UserGroup, error) {
				return []slack.UserGroup{
					{
						ID:     "S111111",
						Handle: "engineers",
						Name:   "Engineering Team",
					},
					{
						ID:     "S222222",
						Handle: "oncall",
						Name:   "On-Call Team",
					},
				}, nil
			},
			GetUserGroupMembersContextFunc: func(ctx context.Context, groupID string) ([]string, error) {
				if groupID == "S111111" {
					return []string{"U100", "U101", "U102"}, nil
				} else if groupID == "S222222" {
					return []string{"U200", "U201"}, nil
				}
				return nil, goerr.New("group not found")
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Should receive all group members
				gt.Equal(t, 5, len(users))
				gt.True(t, contains(users, "U100"))
				gt.True(t, contains(users, "U101"))
				gt.True(t, contains(users, "U102"))
				gt.True(t, contains(users, "U200"))
				gt.True(t, contains(users, "U201"))
				return &slack.Channel{}, nil
			},
		}

		uc := usecase.NewInvite(mockSlack)

		result, err := uc.InviteUsersByList(
			ctx,
			[]string{},
			[]string{"@engineers", "@oncall"},
			types.ChannelID("C-TEST-CHANNEL"),
		)

		gt.NoError(t, err).Required()
		gt.V(t, result).NotNil()
		gt.Equal(t, 5, len(result.Details))

		// Verify all members are resolved
		for _, detail := range result.Details {
			gt.Equal(t, "success", detail.Status)
			gt.True(t, strings.HasPrefix(detail.UserID, "U"))
			gt.True(t, detail.SourceConfig == "@engineers" || detail.SourceConfig == "@oncall")
		}
	})

	t.Run("Mixed users, bots, and groups with some failures", func(t *testing.T) {
		// Create mock Slack client
		mockSlack := &mocks.SlackClientMock{
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Only U123456 will be invited (others failed to resolve)
				gt.Equal(t, 1, len(users))
				gt.Equal(t, "U123456", users[0])
				return nil, errors.New("channel_not_found")
			},
			GetUsersContextFunc: func(ctx context.Context) ([]slack.User, error) {
				// Return empty list to simulate users not found
				return []slack.User{}, nil
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
			if detail.SourceConfig == "U123456" {
				gt.Equal(t, "failed", detail.Status)
				gt.Equal(t, "channel_not_found", detail.Error)
			} else if detail.SourceConfig == "B09E8M5JSPK" {
				gt.Equal(t, "failed", detail.Status)
				gt.Equal(t, "bot not found", detail.Error)
			} else if detail.SourceConfig == "@tamamo" || detail.SourceConfig == "@unknown" {
				gt.Equal(t, "failed", detail.Status)
				gt.Equal(t, "user not found", detail.Error)
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