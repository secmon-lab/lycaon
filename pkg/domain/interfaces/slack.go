package interfaces

//go:generate moq -out mocks/slack_mock.go -pkg mocks . SlackClient

import (
	"context"

	"github.com/slack-go/slack"
)

// SlackClient defines the interface for Slack operations
type SlackClient interface {
	CreateConversation(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error)
	InviteUsersToConversation(ctx context.Context, channelID string, users ...string) (*slack.Channel, error)
	PostMessage(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error)
	UpdateMessage(ctx context.Context, channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error)
	AuthTestContext(ctx context.Context) (*slack.AuthTestResponse, error)
	GetConversationInfo(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error)
	SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*slack.Channel, error)
	OpenView(ctx context.Context, triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error)
	GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, bool, error)
	SendContextMessage(ctx context.Context, channelID, messageTS, contextText string) string

	// User and group resolution methods
	GetUsersContext(ctx context.Context) ([]slack.User, error)
	GetUserInfoContext(ctx context.Context, userID string) (*slack.User, error)
	GetUserGroupsContext(ctx context.Context) ([]slack.UserGroup, error)
	GetUserGroupMembersContext(ctx context.Context, groupID string) ([]string, error)
	GetUsersInConversationContext(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error)

	// Bookmark management
	AddBookmark(ctx context.Context, channelID, title, link string) error
}
