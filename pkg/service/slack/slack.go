package slack

import (
	"context"
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/slack-go/slack"
)

// Service provides Slack messaging capabilities
type Service struct {
	client *slack.Client
}

// New creates a new Slack service
func New(token string) *Service {
	return &Service{
		client: slack.New(token),
	}
}

// PostMessage sends a message to a Slack channel
func (s *Service) PostMessage(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
	channel, timestamp, err := s.client.PostMessageContext(ctx, channelID, options...)
	if err != nil {
		return "", "", goerr.Wrap(err, "failed to post message to Slack")
	}
	return channel, timestamp, nil
}

// CreateConversation creates a new Slack channel
func (s *Service) CreateConversation(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
	channel, err := s.client.CreateConversationContext(ctx, params)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Slack conversation")
	}
	return channel, nil
}

// InviteUsersToConversation invites users to a Slack channel
func (s *Service) InviteUsersToConversation(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
	channel, err := s.client.InviteUsersToConversationContext(ctx, channelID, users...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to invite users to conversation")
	}
	return channel, nil
}

// PostEphemeral sends an ephemeral message visible only to the specified user
func (s *Service) PostEphemeral(ctx context.Context, channelID, userID string, options ...slack.MsgOption) (string, error) {
	timestamp, err := s.client.PostEphemeralContext(ctx, channelID, userID, options...)
	if err != nil {
		return "", goerr.Wrap(err, "failed to post ephemeral message")
	}
	return timestamp, nil
}

// UpdateMessage updates an existing Slack message
func (s *Service) UpdateMessage(ctx context.Context, channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error) {
	channel, ts, text, err := s.client.UpdateMessageContext(ctx, channelID, timestamp, options...)
	if err != nil {
		return "", "", "", goerr.Wrap(err, "failed to update message")
	}
	return channel, ts, text, nil
}

// GetConversationInfo retrieves information about a Slack channel
func (s *Service) GetConversationInfo(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
	params := &slack.GetConversationInfoInput{
		ChannelID:     channelID,
		IncludeLocale: includeLocale,
	}
	channel, err := s.client.GetConversationInfoContext(ctx, params)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get conversation info")
	}
	return channel, nil
}

// OpenConversation opens a conversation with a user
func (s *Service) OpenConversation(ctx context.Context, params *slack.OpenConversationParameters) (*slack.Channel, bool, bool, error) {
	channel, wasAlreadyOpen, noOp, err := s.client.OpenConversationContext(ctx, params)
	if err != nil {
		return nil, false, false, goerr.Wrap(err, "failed to open conversation")
	}
	return channel, wasAlreadyOpen, noOp, nil
}

// AuthTestContext tests authentication and returns basic information about the team and bot
func (s *Service) AuthTestContext(ctx context.Context) (*slack.AuthTestResponse, error) {
	resp, err := s.client.AuthTestContext(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to authenticate with Slack")
	}
	return resp, nil
}

// SetPurposeOfConversationContext sets the purpose (description) of a Slack channel
func (s *Service) SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
	channel, err := s.client.SetPurposeOfConversationContext(ctx, channelID, purpose)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to set channel purpose")
	}
	return channel, nil
}

// GetClient returns the underlying Slack client for advanced operations
func (s *Service) GetClient() *slack.Client {
	return s.client
}

// FormatIncidentChannelName formats the incident channel name with proper padding
func FormatIncidentChannelName(incidentNumber int) string {
	return fmt.Sprintf("inc-%03d", incidentNumber)
}
