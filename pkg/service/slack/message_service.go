package slack

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/slack-go/slack"
)

// messageService handles Slack message operations (private service)
type messageService struct {
	client  interfaces.SlackClient
	builder *BlockBuilder
	config  *model.Config
}

// newMessageService creates a new messageService instance
func newMessageService(client interfaces.SlackClient, builder *BlockBuilder, config *model.Config) *messageService {
	return &messageService{
		client:  client,
		builder: builder,
		config:  config,
	}
}

// postStatusMessage sends a status message to the incident channel
func (s *messageService) postStatusMessage(ctx context.Context, channelID types.ChannelID, incident *model.Incident, leadName string) error {
	if channelID == "" {
		return goerr.New("channel ID is required")
	}

	// Build status message blocks
	blocks := s.builder.BuildStatusMessageBlocks(incident, leadName, s.config)

	// Post message to Slack
	_, _, err := s.client.PostMessage(ctx, string(channelID), slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return goerr.Wrap(err, "failed to post status message to Slack")
	}

	return nil
}

// updateStatusMessage updates an existing status message
func (s *messageService) updateStatusMessage(ctx context.Context, channelID types.ChannelID, messageTS string, incident *model.Incident, leadName string) error {
	if channelID == "" || messageTS == "" {
		return goerr.New("channelID and messageTS are required",
			goerr.V("channelID", channelID),
			goerr.V("messageTS", messageTS))
	}

	// Build updated status message blocks
	blocks := s.builder.BuildStatusMessageBlocks(incident, leadName, s.config)

	// Update the message
	_, _, _, err := s.client.UpdateMessage(ctx, string(channelID), messageTS, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return goerr.Wrap(err, "failed to update status message",
			goerr.V("channelID", channelID),
			goerr.V("messageTS", messageTS),
			goerr.V("incidentID", incident.ID))
	}

	return nil
}

// postWelcomeMessage sends a welcome message to the incident channel
func (s *messageService) postWelcomeMessage(ctx context.Context, channelID types.ChannelID, incident *model.Incident, originChannelName, leadName string) (string, error) {
	if channelID == "" {
		return "", goerr.New("channel ID is required")
	}

	// Build welcome message blocks
	blocks := s.builder.BuildIncidentChannelWelcomeBlocks(incident, originChannelName, leadName, s.config)

	// Post message to Slack
	_, messageTS, err := s.client.PostMessage(ctx, string(channelID), slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return "", goerr.Wrap(err, "failed to post welcome message to Slack")
	}

	return messageTS, nil
}

// postIncidentCreatedNotification sends an incident creation notification
func (s *messageService) postIncidentCreatedNotification(ctx context.Context, channelID types.ChannelID, messageTS types.MessageTS, originChannelName string, incidentChannelID types.ChannelID, title, categoryID, severityID string) error {
	if channelID == "" {
		return goerr.New("channel ID is required")
	}

	// Build incident created notification blocks
	blocks := s.builder.BuildIncidentCreatedBlocks(originChannelName, string(incidentChannelID), title, categoryID, severityID, s.config)

	// Post message with broadcast option
	_, _, err := s.client.PostMessage(
		ctx,
		string(channelID),
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(string(messageTS)),
		slack.MsgOptionBroadcast(),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to post incident creation notification")
	}

	return nil
}

// updateOriginalPromptToUsed updates the original incident prompt to show it was used
func (s *messageService) updateOriginalPromptToUsed(ctx context.Context, channelID types.ChannelID, messageTS types.MessageTS, title string) error {
	if channelID == "" || messageTS == "" {
		return goerr.New("channelID and messageTS are required",
			goerr.V("channelID", channelID),
			goerr.V("messageTS", messageTS))
	}

	// Build used prompt blocks
	blocks := s.builder.BuildIncidentPromptUsedBlocks(title)

	// Update the message
	_, _, _, err := s.client.UpdateMessage(ctx, string(channelID), string(messageTS), slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return goerr.Wrap(err, "failed to update original prompt message",
			goerr.V("channelID", channelID),
			goerr.V("messageTS", messageTS))
	}

	return nil
}

// postErrorMessage sends an error message
func (s *messageService) postErrorMessage(ctx context.Context, channelID types.ChannelID, errorText string) error {
	if channelID == "" {
		return goerr.New("channel ID is required")
	}

	// Build error blocks
	blocks := s.builder.BuildErrorBlocks(errorText)

	// Post message
	_, _, err := s.client.PostMessage(ctx, string(channelID), slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return goerr.Wrap(err, "failed to post error message")
	}

	return nil
}

// postTaskMessage sends a task message
func (s *messageService) postTaskMessage(ctx context.Context, channelID types.ChannelID, task *model.Task, assigneeUsername string) (string, error) {
	if channelID == "" {
		return "", goerr.New("channel ID is required")
	}

	// Build task message blocks
	blocks := BuildTaskMessage(task, assigneeUsername)

	// Post message
	_, messageTS, err := s.client.PostMessage(ctx, string(channelID), slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return "", goerr.Wrap(err, "failed to post task message")
	}

	return messageTS, nil
}

// updateTaskMessage updates an existing task message
func (s *messageService) updateTaskMessage(ctx context.Context, channelID types.ChannelID, messageTS string, task *model.Task, assigneeUsername string) error {
	if channelID == "" || messageTS == "" {
		return goerr.New("channelID and messageTS are required",
			goerr.V("channelID", channelID),
			goerr.V("messageTS", messageTS))
	}

	// Build task message blocks
	blocks := BuildTaskMessage(task, assigneeUsername)

	// Update the message
	_, _, _, err := s.client.UpdateMessage(ctx, string(channelID), messageTS, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return goerr.Wrap(err, "failed to update task message",
			goerr.V("channelID", channelID),
			goerr.V("messageTS", messageTS),
			goerr.V("taskID", task.ID))
	}

	return nil
}

// postIncidentPromptMessage posts an incident prompt message with LLM-generated details
func (s *messageService) postIncidentPromptMessage(ctx context.Context, channelID string, messageTS, requestID, title, description, categoryID, severityID string) (string, error) {
	if channelID == "" {
		return "", goerr.New("channel ID is required")
	}

	// Build incident prompt blocks
	promptBlocks := s.builder.BuildIncidentPromptBlocks(requestID, title, description, categoryID, severityID, s.config)

	// Post message as a thread reply
	_, botMessageTS, err := s.client.PostMessage(
		ctx,
		channelID,
		slack.MsgOptionBlocks(promptBlocks...),
		slack.MsgOptionTS(messageTS), // Reply in thread
	)
	if err != nil {
		return "", goerr.Wrap(err, "failed to post incident prompt message")
	}

	return botMessageTS, nil
}
