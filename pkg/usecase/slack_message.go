package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	llmSvc "github.com/secmon-lab/lycaon/pkg/service/llm"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

// SlackMessage implements SlackMessage interface
type SlackMessage struct {
	repo           interfaces.Repository
	gollemClient   gollem.LLMClient
	slackClient    interfaces.SlackClient
	blockBuilder   *slackSvc.BlockBuilder
	botUserID      string // Bot's user ID for mention detection
	messageHistory *slackSvc.MessageHistoryService
	llmService     *llmSvc.LLMService
}

// NewSlackMessage creates a new SlackMessage use case
func NewSlackMessage(
	ctx context.Context,
	repo interfaces.Repository,
	gollemClient gollem.LLMClient,
	slackClient interfaces.SlackClient,
) (*SlackMessage, error) {
	s := &SlackMessage{
		repo:           repo,
		gollemClient:   gollemClient,
		slackClient:    slackClient,
		blockBuilder:   slackSvc.NewBlockBuilder(),
		messageHistory: slackSvc.NewMessageHistoryService(slackClient),
		llmService:     llmSvc.NewLLMService(gollemClient),
	}

	// Always get bot user ID from Slack API if client is available
	if slackClient != nil {
		authResp, err := slackClient.AuthTestContext(ctx)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to authenticate with Slack")
		}
		s.botUserID = authResp.UserID
		ctxlog.From(ctx).Info("Bot user ID retrieved",
			"botUserID", s.botUserID,
			"botName", authResp.User,
		)
	} else {
		ctxlog.From(ctx).Debug("Slack client not provided, bot user ID not available")
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

	if s.gollemClient == nil {
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

	// Create session for LLM generation
	session, err := s.gollemClient.NewSession(ctx)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to create LLM session",
			"error", err,
			"messageID", message.ID,
		)
		// Return a fallback response on error
		return "I understand your message. Let me help you with that.", nil
	}

	// Generate response using gollem
	response, err := session.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		ctxlog.From(ctx).Error("Failed to generate LLM response",
			"error", err,
			"messageID", message.ID,
		)
		// Return a fallback response on error
		return "I understand your message. Let me help you with that.", nil
	}

	if len(response.Texts) == 0 || response.Texts[0] == "" {
		// Return fallback response if no content generated
		return "I understand your message. Let me help you with that.", nil
	}

	return response.Texts[0], nil
}

// SaveAndRespond saves a message and optionally generates a response
// Deprecated: This method is kept for backward compatibility but should not be used for new features
// The bot no longer responds to general mentions, only to incident triggers
func (s *SlackMessage) SaveAndRespond(ctx context.Context, event *slackevents.MessageEvent) (string, error) {
	if event == nil {
		return "", goerr.New("message event is nil")
	}

	// Process and save the message
	if err := s.ProcessMessage(ctx, event); err != nil {
		return "", goerr.Wrap(err, "failed to process message")
	}

	// No longer generate responses for general mentions
	// Only incident triggers get responses via SendIncidentMessage
	return "", nil
}

// parseIncidentCommand parses a Slack message to check if it's an incident trigger and extract title
func parseIncidentCommand(message *model.Message, botUserID string) interfaces.IncidentCommand {
	if message == nil || message.Text == "" || botUserID == "" {
		return interfaces.IncidentCommand{IsIncidentTrigger: false}
	}

	// Build the bot mention pattern
	botMention := fmt.Sprintf("<@%s>", botUserID)

	// Split message into tokens (words)
	parts := strings.Fields(message.Text)

	// Look for bot mention followed by "inc" command
	for i, part := range parts {
		if part == botMention {
			// Check if 'inc' is the next token
			if i+1 < len(parts) && strings.ToLower(parts[i+1]) == "inc" {
				// Found valid inc command after bot mention
				title := ""
				if i+2 < len(parts) {
					// Collect all remaining parts as the title
					title = strings.Join(parts[i+2:], " ")
				}
				return interfaces.IncidentCommand{
					IsIncidentTrigger: true,
					Title:             strings.TrimSpace(title),
					Description:       "", // Will be populated by ParseIncidentCommand if needed
				}
			}
		}
	}

	// No valid inc command found
	return interfaces.IncidentCommand{IsIncidentTrigger: false}
}

// ParseIncidentCommand parses a Slack message to check if it's an incident trigger and extract title
// Only messages mentioning this bot specifically are accepted:
// - Bot mention followed by inc: "<@BOT_ID> inc something happened"
// - Multiple mentions with bot: "<@USER1> <@BOT_ID> inc something happened"
// The "inc" command must come immediately after the bot's mention.
// If no title is provided, it will analyze message history and generate title/description using LLM.
func (s *SlackMessage) ParseIncidentCommand(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
	if s.botUserID == "" {
		ctxlog.From(ctx).Debug("Bot user ID not set, cannot parse incident command")
		return interfaces.IncidentCommand{IsIncidentTrigger: false, Title: "", Description: ""}
	}

	// Parse basic incident command
	basicCommand := parseIncidentCommand(message, s.botUserID)
	if !basicCommand.IsIncidentTrigger {
		return basicCommand
	}

	// If title is provided, return as-is
	if basicCommand.Title != "" {
		ctxlog.From(ctx).Debug("Incident command with title provided",
			"title", basicCommand.Title)
		return basicCommand
	}

	// No title provided - analyze message history with LLM if available
	if s.gollemClient != nil {
		ctxlog.From(ctx).Info("No title provided for incident command, analyzing message history")
		enhancedCommand := s.enhanceIncidentCommandWithLLM(ctx, message, basicCommand)
		return enhancedCommand
	}

	// LLM not available, return basic command without enhancement
	ctxlog.From(ctx).Debug("LLM not available for incident enhancement, returning basic command")
	return basicCommand
}

// enhanceIncidentCommandWithLLM enhances an incident command by analyzing message history with LLM
func (s *SlackMessage) enhanceIncidentCommandWithLLM(ctx context.Context, message *model.Message, baseCommand interfaces.IncidentCommand) interfaces.IncidentCommand {
	// Determine message retrieval strategy
	channelID := string(message.ChannelID)
	threadTS := string(message.ThreadTS)

	var opts slackSvc.MessageHistoryOptions
	if threadTS != "" && threadTS != string(message.EventTS) {
		// Thread message - get thread history
		ctxlog.From(ctx).Debug("Retrieving thread messages for incident analysis",
			"channelID", channelID,
			"threadTS", threadTS)
		opts = slackSvc.MessageHistoryOptions{
			ChannelID: channelID,
			ThreadTS:  threadTS,
			Limit:     256,
		}
	} else {
		// Channel message - get recent channel history
		twoHoursAgo := time.Now().Add(-2 * time.Hour)
		ctxlog.From(ctx).Debug("Retrieving channel messages for incident analysis",
			"channelID", channelID,
			"oldestTime", twoHoursAgo)
		opts = slackSvc.MessageHistoryOptions{
			ChannelID:  channelID,
			Limit:      256,
			OldestTime: &twoHoursAgo,
		}
	}

	// Retrieve message history
	messages, err := s.messageHistory.GetMessages(ctx, opts)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to retrieve message history for incident analysis",
			"error", err,
			"channelID", channelID)
		// Return basic command on failure
		return baseCommand
	}

	if len(messages) == 0 {
		ctxlog.From(ctx).Debug("No messages found for incident analysis")
		return baseCommand
	}

	ctxlog.From(ctx).Debug("Analyzing messages with LLM",
		"messageCount", len(messages))

	// Generate summary using LLM
	summary, err := s.llmService.GenerateIncidentSummary(ctx, messages)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to generate incident summary with LLM",
			"error", err,
			"messageCount", len(messages))
		// Return basic command on LLM failure
		return baseCommand
	}

	ctxlog.From(ctx).Info("Generated incident summary with LLM",
		"title", summary.Title,
		"descriptionLength", len(summary.Description))

	// Return enhanced command
	return interfaces.IncidentCommand{
		IsIncidentTrigger: true,
		Title:             summary.Title,
		Description:       summary.Description,
	}
}

// SendIncidentMessage sends an incident creation prompt message
func (s *SlackMessage) SendIncidentMessage(ctx context.Context, channelID, messageTS, title, description string) error {
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
	request := model.NewIncidentRequest(types.ChannelID(channelID), types.MessageTS(messageTS), title, description, types.SlackUserID(requestedBy))
	if err := s.repo.SaveIncidentRequest(ctx, request); err != nil {
		return goerr.Wrap(err, "failed to save incident request")
	}

	// Build incident prompt blocks with the request ID
	blocks := s.blockBuilder.BuildIncidentPromptBlocks(request.ID.String(), title)

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
		ID:        types.MessageID(messageID),
		UserID:    types.SlackUserID(event.User),
		UserName:  event.Username,
		ChannelID: types.ChannelID(event.Channel),
		Text:      event.Text,
		ThreadTS:  types.ThreadTS(event.ThreadTimeStamp),
		EventTS:   types.EventTS(event.TimeStamp),
	}
}
