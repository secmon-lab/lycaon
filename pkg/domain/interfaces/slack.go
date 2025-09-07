package interfaces

import (
	"context"

	"github.com/slack-go/slack"
)

// SlackClient defines the interface for Slack operations
type SlackClient interface {
	CreateConversation(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error)
	InviteUsersToConversation(ctx context.Context, channelID string, users ...string) (*slack.Channel, error)
	PostMessage(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error)
	AuthTestContext(ctx context.Context) (*slack.AuthTestResponse, error)
	GetConversationInfo(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error)
	SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*slack.Channel, error)
}
