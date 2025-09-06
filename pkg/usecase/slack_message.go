package usecase

import (
	"context"
	"fmt"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// SlackMessage implements SlackMessageUseCase
type SlackMessage struct {
	repo        interfaces.Repository
	llmClient   interfaces.LLMClient
	slackClient *slack.Client
}

// NewSlackMessage creates a new SlackMessage use case
func NewSlackMessage(
	ctx context.Context,
	repo interfaces.Repository,
	llmClient interfaces.LLMClient,
	slackClient *slack.Client,
) SlackMessageUseCase {
	return &SlackMessage{
		repo:        repo,
		llmClient:   llmClient,
		slackClient: slackClient,
	}
}

// ProcessMessage processes an incoming Slack message
func (s *SlackMessage) ProcessMessage(ctx context.Context, event *slackevents.MessageEvent) error {
	if event == nil {
		return goerr.New("message event is nil")
	}

	// Convert Slack event to our domain model
	message := s.eventToMessage(event)

	// Save the message to repository
	if err := s.repo.SaveMessage(ctx, message); err != nil {
		return goerr.Wrap(err, "failed to save message")
	}

	ctxlog.From(ctx).Info("Message processed",
		"messageID", message.ID,
		"channelID", message.ChannelID,
		"userID", message.UserID,
	)

	return nil
}

// GenerateResponse generates an LLM response for a message
func (s *SlackMessage) GenerateResponse(ctx context.Context, message *model.Message) (string, error) {
	if message == nil {
		return "", goerr.New("message is nil")
	}

	if s.llmClient == nil {
		// If LLM client is not configured, return a default response
		return "Thank you for your message. I'm currently processing it.", nil
	}

	// Create a prompt for the LLM
	prompt := fmt.Sprintf(
		"You are a helpful incident management assistant. "+
			"A user sent the following message in Slack: '%s'. "+
			"Please provide a helpful and concise response.",
		message.Text,
	)

	// Generate response using LLM
	response, err := s.llmClient.GenerateResponse(ctx, prompt)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to generate LLM response",
			"error", err,
			"messageID", message.ID,
		)
		// Return a fallback response on error
		return "I understand your message. Let me help you with that.", nil
	}

	return response, nil
}

// SaveAndRespond saves a message and generates a response
func (s *SlackMessage) SaveAndRespond(ctx context.Context, event *slackevents.MessageEvent) (string, error) {
	if event == nil {
		return "", goerr.New("message event is nil")
	}

	// Process and save the message
	if err := s.ProcessMessage(ctx, event); err != nil {
		return "", goerr.Wrap(err, "failed to process message")
	}

	// Convert to domain model for response generation
	message := s.eventToMessage(event)

	// Generate a response
	response, err := s.GenerateResponse(ctx, message)
	if err != nil {
		return "", goerr.Wrap(err, "failed to generate response")
	}

	// Send the response back to Slack if client is configured
	if s.slackClient != nil {
		_, _, err = s.slackClient.PostMessage(
			event.Channel,
			slack.MsgOptionText(response, false),
			slack.MsgOptionTS(event.TimeStamp), // Thread response
		)
		if err != nil {
			ctxlog.From(ctx).Error("Failed to send Slack response",
				"error", err,
				"channel", event.Channel,
			)
			// Don't fail the whole operation if sending fails
		}
	}

	return response, nil
}

// eventToMessage converts a Slack event to our domain model
func (s *SlackMessage) eventToMessage(event *slackevents.MessageEvent) *model.Message {
	return &model.Message{
		ID:        event.ClientMsgID,
		UserID:    event.User,
		UserName:  event.Username,
		ChannelID: event.Channel,
		Text:      event.Text,
		ThreadTS:  event.ThreadTimeStamp,
		EventTS:   event.TimeStamp,
	}
}
