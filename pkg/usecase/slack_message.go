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

const (
	// channelHistoryDuration defines how far back to look for channel messages when analyzing incident context
	channelHistoryDuration = 2 * time.Hour

	// maxMessageHistoryLimit is the maximum number of messages to retrieve for incident analysis
	maxMessageHistoryLimit = 256
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
	categories     *model.CategoriesConfig
}

// NewSlackMessage creates a new SlackMessage use case
// Both gollemClient and slackClient are required parameters
func NewSlackMessage(
	ctx context.Context,
	repo interfaces.Repository,
	gollemClient gollem.LLMClient,
	slackClient interfaces.SlackClient,
	categories *model.CategoriesConfig,
) (*SlackMessage, error) {
	// Validate required parameters
	if repo == nil {
		return nil, goerr.New("repository is required")
	}
	if gollemClient == nil {
		return nil, goerr.New("LLM client is required")
	}
	if slackClient == nil {
		return nil, goerr.New("Slack client is required")
	}
	if categories == nil {
		return nil, goerr.New("categories configuration is required")
	}

	s := &SlackMessage{
		repo:           repo,
		gollemClient:   gollemClient,
		slackClient:    slackClient,
		blockBuilder:   slackSvc.NewBlockBuilder(),
		messageHistory: slackSvc.NewMessageHistoryService(slackClient),
		llmService:     llmSvc.NewLLMService(gollemClient),
		categories:     categories,
	}

	// Get bot user ID from Slack API
	authResp, err := slackClient.AuthTestContext(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to authenticate with Slack")
	}
	s.botUserID = authResp.UserID
	ctxlog.From(ctx).Info("Bot user ID retrieved",
		"botUserID", s.botUserID,
		"botName", authResp.User,
	)

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

	// Always use LLM for analysis - title text becomes additional prompt
	if basicCommand.Title != "" {
		ctxlog.From(ctx).Debug("Incident command with additional prompt provided",
			"prompt", basicCommand.Title)
	} else {
		ctxlog.From(ctx).Info("No additional prompt for incident command, analyzing message history only")
	}

	enhancedCommand := s.enhanceIncidentCommandWithLLM(ctx, message, basicCommand)
	return enhancedCommand
}

// enhanceIncidentCommandWithLLM enhances an incident command by analyzing message history with LLM
func (s *SlackMessage) enhanceIncidentCommandWithLLM(ctx context.Context, message *model.Message, baseCommand interfaces.IncidentCommand) interfaces.IncidentCommand {
	// Extract additional prompt from base command title
	additionalPrompt := ""
	if baseCommand.Title != "" {
		additionalPrompt = baseCommand.Title
		ctxlog.From(ctx).Debug("Using inc command text as additional prompt",
			"prompt", additionalPrompt)
	}

	// Get channel information for context
	channelID := string(message.ChannelID)
	channelInfo, err := s.slackClient.GetConversationInfo(ctx, channelID, false)
	if err != nil {
		ctxlog.From(ctx).Warn("Failed to get channel info",
			"channelID", channelID,
			"error", err)
		channelInfo = nil // Continue without channel info
	} else {
		ctxlog.From(ctx).Debug("Retrieved channel info for incident analysis",
			"channelName", channelInfo.Name,
			"topic", channelInfo.Topic.Value)
	}

	// Determine message retrieval strategy
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
			Limit:     maxMessageHistoryLimit,
		}
	} else {
		// Channel message - get recent channel history
		twoHoursAgo := time.Now().Add(-channelHistoryDuration)
		ctxlog.From(ctx).Debug("Retrieving channel messages for incident analysis",
			"channelID", channelID,
			"oldestTime", twoHoursAgo)
		opts = slackSvc.MessageHistoryOptions{
			ChannelID:  channelID,
			Limit:      maxMessageHistoryLimit,
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
		"messageCount", len(messages),
		"hasAdditionalPrompt", additionalPrompt != "",
		"hasChannelInfo", channelInfo != nil)

	// Perform comprehensive incident analysis with additional context
	summary, err := s.llmService.AnalyzeIncidentWithContext(ctx, messages, s.categories, additionalPrompt, channelInfo)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to analyze incident with LLM",
			"error", err,
			"messageCount", len(messages),
			"additionalPrompt", additionalPrompt)
		// Return basic command on LLM failure
		return baseCommand
	}

	ctxlog.From(ctx).Info("Incident analysis completed with LLM",
		"title", summary.Title,
		"descriptionLength", len(summary.Description),
		"categoryID", summary.CategoryID)

	// Return enhanced command with all information
	return interfaces.IncidentCommand{
		IsIncidentTrigger: true,
		Title:             summary.Title,
		Description:       summary.Description,
		CategoryID:        summary.CategoryID,
	}
}

// IsBasicIncidentTrigger quickly checks if message is an inc command (without LLM analysis)
func (s *SlackMessage) IsBasicIncidentTrigger(ctx context.Context, message *model.Message) bool {
	if s.botUserID == "" || message == nil || message.Text == "" {
		return false
	}

	// Parse basic incident command to check for "inc" trigger
	cmd := parseIncidentCommand(message, s.botUserID)
	return cmd.IsIncidentTrigger
}

// SendProcessingMessage sends an immediate processing context message
func (s *SlackMessage) SendProcessingMessage(ctx context.Context, channelID, messageTS string) error {
	// Send context message using the slack service
	msgTS := s.sendContextMessage(ctx, channelID, messageTS, "🔄 Processing incident command...")
	if msgTS == "" {
		// Don't return error since it's not critical
		ctxlog.From(ctx).Warn("Processing message was not sent, but continuing",
			"channelID", channelID,
			"messageTS", messageTS,
		)
	}
	return nil
}

// SendIncidentMessage sends an incident creation prompt message
func (s *SlackMessage) SendIncidentMessage(ctx context.Context, channelID, messageTS, title, description, categoryID string) error {
	ctxlog.From(ctx).Info("SendIncidentMessage called",
		"channelID", channelID,
		"messageTS", messageTS,
		"title", title,
		"categoryID", categoryID,
	)

	if s.slackClient == nil {
		return goerr.New("slack client is not configured")
	}

	// Get the user ID from the auth test response
	requestedBy := s.botUserID
	if requestedBy == "" {
		requestedBy = "unknown"
	}

	// Create an incident request and save it
	request := model.NewIncidentRequest(types.ChannelID(channelID), types.MessageTS(messageTS), title, description, categoryID, types.SlackUserID(requestedBy))
	if err := s.repo.SaveIncidentRequest(ctx, request); err != nil {
		return goerr.Wrap(err, "failed to save incident request")
	}

	// Build incident prompt blocks with the request ID
	promptBlocks := s.blockBuilder.BuildIncidentPromptBlocks(request.ID.String(), title)

	// Send incident prompt message
	_, botMessageTS, err := s.slackClient.PostMessage(
		ctx,
		channelID,
		slack.MsgOptionBlocks(promptBlocks...),
		slack.MsgOptionTS(messageTS), // Reply in thread
	)
	if err != nil {
		// Clean up the request if we failed to send the message
		_ = s.repo.DeleteIncidentRequest(ctx, request.ID)
		return goerr.Wrap(err, "failed to send incident prompt message")
	}

	// Update the request with the bot message timestamp
	request.BotMessageTS = types.MessageTS(botMessageTS)
	if err := s.repo.SaveIncidentRequest(ctx, request); err != nil {
		ctxlog.From(ctx).Warn("Failed to update incident request with bot message timestamp",
			"error", err,
			"requestID", request.ID,
			"botMessageTS", botMessageTS,
		)
	}

	return nil
}

// sendContextMessage sends a context block message and returns the timestamp
// This method does not return errors - failures are logged but don't affect the main flow
func (s *SlackMessage) sendContextMessage(ctx context.Context, channelID, messageTS, contextText string) string {
	// Use the interface method directly
	return s.slackClient.SendContextMessage(ctx, channelID, messageTS, contextText)
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
