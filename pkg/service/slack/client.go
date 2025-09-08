package slack

import (
	"context"

	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/slack-go/slack"
)

// ClientAdapter adapts the Service to the interfaces.SlackClient interface
type ClientAdapter struct {
	service *Service
}

// NewClientAdapter creates a new ClientAdapter
func NewClientAdapter(token string) interfaces.SlackClient {
	return &ClientAdapter{
		service: New(token),
	}
}

// CreateConversation implements interfaces.SlackClient
func (a *ClientAdapter) CreateConversation(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
	return a.service.CreateConversation(ctx, params)
}

// InviteUsersToConversation implements interfaces.SlackClient
func (a *ClientAdapter) InviteUsersToConversation(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
	return a.service.InviteUsersToConversation(ctx, channelID, users...)
}

// PostMessage implements interfaces.SlackClient
func (a *ClientAdapter) PostMessage(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
	return a.service.PostMessage(ctx, channelID, options...)
}

// AuthTestContext implements interfaces.SlackClient
func (a *ClientAdapter) AuthTestContext(ctx context.Context) (*slack.AuthTestResponse, error) {
	return a.service.AuthTestContext(ctx)
}

// GetConversationInfo implements interfaces.SlackClient
func (a *ClientAdapter) GetConversationInfo(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
	return a.service.GetConversationInfo(ctx, channelID, includeLocale)
}

// SetPurposeOfConversationContext implements interfaces.SlackClient
func (a *ClientAdapter) SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
	return a.service.SetPurposeOfConversationContext(ctx, channelID, purpose)
}

// OpenView implements interfaces.SlackClient
func (a *ClientAdapter) OpenView(ctx context.Context, triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error) {
	return a.service.OpenView(ctx, triggerID, view)
}
