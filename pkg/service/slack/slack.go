package slack

import (
	"context"
	"fmt"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/slack-go/slack"
)

// Service provides Slack messaging capabilities
type Service struct {
	client *slack.Client
}

// New creates a new Slack service that implements interfaces.SlackClient
func New(token string) interfaces.SlackClient {
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


// OpenView opens a modal view in Slack
func (s *Service) OpenView(ctx context.Context, triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error) {
	resp, err := s.client.OpenView(triggerID, view)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to open modal view")
	}
	return resp, nil
}

// GetConversationHistoryContext retrieves conversation history
func (s *Service) GetConversationHistoryContext(ctx context.Context, params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error) {
	resp, err := s.client.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get conversation history")
	}
	return resp, nil
}

// GetConversationRepliesContext retrieves conversation replies (thread messages)
func (s *Service) GetConversationRepliesContext(ctx context.Context, params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, bool, error) {
	messages, hasMore, nextCursor, err := s.client.GetConversationRepliesContext(ctx, params)
	if err != nil {
		return nil, false, false, goerr.Wrap(err, "failed to get conversation replies")
	}
	return messages, hasMore, nextCursor != "", nil
}

// SendContextMessage sends a context block message to a channel
// This method does not return errors - failures are logged but don't affect the main flow
// Returns the timestamp of the sent message, or empty string if failed
func (s *Service) SendContextMessage(ctx context.Context, channelID, messageTS, contextText string) string {
	ctxlog.From(ctx).Info("Sending context message",
		"channelID", channelID,
		"messageTS", messageTS,
		"contextText", contextText,
	)

	// Build context blocks
	contextBlocks := []slack.Block{
		slack.NewContextBlock(
			"",
			slack.NewTextBlockObject(
				slack.MarkdownType,
				contextText,
				false,
				false,
			),
		),
	}

	// Debug log the actual block structure
	ctxlog.From(ctx).Debug("Context block structure",
		"block", fmt.Sprintf("%+v", contextBlocks[0]),
	)

	ctxlog.From(ctx).Debug("Built context blocks",
		"blockCount", len(contextBlocks),
		"channelID", channelID,
	)

	channelResp, contextMsgTS, err := s.client.PostMessageContext(
		ctx,
		channelID,
		slack.MsgOptionBlocks(contextBlocks...),
		slack.MsgOptionTS(messageTS), // Reply in thread
	)
	if err != nil {
		// Log error but don't fail - context message is not critical
		ctxlog.From(ctx).Error("Failed to send context message",
			"error", err,
			"errorType", fmt.Sprintf("%T", err),
			"channelID", channelID,
			"messageTS", messageTS,
			"contextText", contextText,
		)
		return ""
	}

	ctxlog.From(ctx).Info("Context message sent successfully",
		"channelID", channelID,
		"channelResp", channelResp,
		"contextMsgTS", contextMsgTS,
		"contextText", contextText,
	)
	return contextMsgTS
}

// GetUsersContext retrieves the list of users (including bots) from the workspace
func (s *Service) GetUsersContext(ctx context.Context) ([]slack.User, error) {
	users, err := s.client.GetUsersContext(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get users")
	}
	return users, nil
}

// GetUserInfoContext retrieves information about a specific user
func (s *Service) GetUserInfoContext(ctx context.Context, userID string) (*slack.User, error) {
	user, err := s.client.GetUserInfoContext(ctx, userID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get user info", goerr.V("userID", userID))
	}
	return user, nil
}

// GetUserGroupsContext retrieves the list of user groups from the workspace
func (s *Service) GetUserGroupsContext(ctx context.Context) ([]slack.UserGroup, error) {
	groups, err := s.client.GetUserGroupsContext(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get user groups")
	}
	return groups, nil
}

// GetUserGroupMembersContext retrieves the member IDs of a user group
func (s *Service) GetUserGroupMembersContext(ctx context.Context, groupID string) ([]string, error) {
	members, err := s.client.GetUserGroupMembersContext(ctx, groupID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get user group members", goerr.V("groupID", groupID))
	}
	return members, nil
}
