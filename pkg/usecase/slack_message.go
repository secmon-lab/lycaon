package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// SlackMessage implements SlackMessage interface
type SlackMessage struct {
	repo         interfaces.Repository
	llmClient    interfaces.LLMClient
	slackClient  interfaces.SlackClient
	blockBuilder *slackSvc.BlockBuilder
	botUserID    string // Bot's user ID for mention detection
}

// NewSlackMessage creates a new SlackMessage use case
func NewSlackMessage(
	ctx context.Context,
	repo interfaces.Repository,
	llmClient interfaces.LLMClient,
	slackClient interfaces.SlackClient,
	botUserID string, // Optional: if empty, will try to retrieve from Slack API
) (*SlackMessage, error) {
	s := &SlackMessage{
		repo:         repo,
		llmClient:    llmClient,
		slackClient:  slackClient,
		blockBuilder: slackSvc.NewBlockBuilder(),
		botUserID:    botUserID,
	}

	// Get bot user ID if not provided and client is available
	if botUserID == "" && slackClient != nil {
		authResp, err := slackClient.AuthTestContext(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to authenticate with Slack")
		}
		s.botUserID = authResp.UserID
		ctxlog.From(ctx).Info("Bot user ID retrieved",
			"botUserID", s.botUserID,
			"botName", authResp.User,
		)
	}

	return s, nil
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
			ctx,
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

// parseIncidentCommand parses a Slack message to check if it's an incident trigger and extract title
func parseIncidentCommand(message *model.Message, botUserID string) interfaces.IncidentCommand {
	result := interfaces.IncidentCommand{
		IsIncidentTrigger: false,
		Title:             "",
	}

	if message == nil || message.Text == "" || botUserID == "" {
		return result
	}

	originalText := strings.TrimSpace(message.Text)

	// Build the bot mention pattern
	botMention := fmt.Sprintf("<@%s>", botUserID)

	// Find the bot mention in the text
	index := strings.Index(originalText, botMention)
	if index == -1 {
		// Bot is not mentioned
		return result
	}

	// Get the text after the bot mention
	afterMention := originalText[index+len(botMention):]
	afterMention = strings.TrimSpace(afterMention)

	// Check if the text after mention starts with "inc" as a separate word
	// Accept: "inc", "inc something", "INC issue"
	// Reject: "incorrect", "income", "incognito"
	lowerAfterMention := strings.ToLower(afterMention)

	if lowerAfterMention == "inc" {
		// Just "inc" command with no title
		result.IsIncidentTrigger = true
		result.Title = ""
		return result
	}

	if strings.HasPrefix(lowerAfterMention, "inc ") {
		// "inc " followed by title
		result.IsIncidentTrigger = true
		// Extract the title after "inc "
		title := afterMention[4:] // Skip "inc " (4 characters)
		result.Title = strings.TrimSpace(title)
		return result
	}

	// Not an inc command
	return result
}

// ParseIncidentCommand parses a Slack message to check if it's an incident trigger and extract title
// Only messages mentioning this bot specifically are accepted:
// - Bot mention followed by inc: "<@BOT_ID> inc something happened"
// - Multiple mentions with bot: "<@USER1> <@BOT_ID> inc something happened"
// The "inc" command must come immediately after the bot's mention.
func (s *SlackMessage) ParseIncidentCommand(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
	if s.botUserID == "" {
		ctxlog.From(ctx).Debug("Bot user ID not set, cannot parse incident command")
		return interfaces.IncidentCommand{IsIncidentTrigger: false, Title: ""}
	}

	return parseIncidentCommand(message, s.botUserID)
}

// SendIncidentMessage sends an incident creation prompt message
func (s *SlackMessage) SendIncidentMessage(ctx context.Context, channelID, messageTS, title string) error {
	if s.slackClient == nil {
		return goerr.New("slack client is not configured")
	}

	// Get the user ID from the auth test response (bot user ID is the one creating the request)
	// In a real scenario, you might want to pass the actual user ID who triggered this
	requestedBy := s.botUserID
	if requestedBy == "" {
		// If we don't have bot user ID, we can't properly track who requested
		// This is just a fallback
		requestedBy = "unknown"
	}

	// Create an incident request and save it
	request := model.NewIncidentRequest(channelID, messageTS, title, requestedBy)
	if err := s.repo.SaveIncidentRequest(ctx, request); err != nil {
		return goerr.Wrap(err, "failed to save incident request")
	}

	// Build incident prompt blocks with the request ID
	blocks := s.blockBuilder.BuildIncidentPromptBlocks(request.ID, title)

	// Send message with blocks
	_, _, err := s.slackClient.PostMessage(
		ctx,
		channelID,
		slack.MsgOptionBlocks(blocks...),
		slack.MsgOptionTS(messageTS), // Reply in thread
	)
	if err != nil {
		// Clean up the request if we failed to send the message
		_ = s.repo.DeleteIncidentRequest(ctx, request.ID)
		return goerr.Wrap(err, "failed to send incident prompt message")
	}

	return nil
}

// eventToMessage converts a Slack event to our domain model
func (s *SlackMessage) eventToMessage(event *slackevents.MessageEvent) *model.Message {
	// Use ClientMsgID if available, otherwise use TimeStamp as the ID
	// ClientMsgID can be empty for certain event types (e.g., app mentions)
	messageID := event.ClientMsgID
	if messageID == "" {
		messageID = event.TimeStamp
	}

	return &model.Message{
		ID:        messageID,
		UserID:    event.User,
		UserName:  event.Username,
		ChannelID: event.Channel,
		Text:      event.Text,
		ThreadTS:  event.ThreadTimeStamp,
		EventTS:   event.TimeStamp,
	}
}
