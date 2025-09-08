package slack

import (
	"context"
	"strconv"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/slack-go/slack"
)

const (
	// maxMessageLimit is the maximum number of messages that can be retrieved from Slack API
	maxMessageLimit = 256
)

// MessageHistoryService handles retrieving message history from Slack
type MessageHistoryService struct {
	slackClient interfaces.SlackClient
}

// MessageHistoryOptions contains options for retrieving message history
type MessageHistoryOptions struct {
	ChannelID  string     // Required: Channel ID to retrieve messages from
	ThreadTS   string     // Optional: Thread timestamp for thread messages
	Limit      int        // Maximum number of messages to retrieve (max maxMessageLimit)
	OldestTime *time.Time // Optional: Oldest time to retrieve messages from
}

// NewMessageHistoryService creates a new MessageHistoryService
func NewMessageHistoryService(slackClient interfaces.SlackClient) *MessageHistoryService {
	return &MessageHistoryService{
		slackClient: slackClient,
	}
}

// GetMessages retrieves messages from Slack based on the provided options
func (s *MessageHistoryService) GetMessages(ctx context.Context, opts MessageHistoryOptions) ([]slack.Message, error) {
	if opts.ChannelID == "" {
		return nil, goerr.New("channel ID is required")
	}

	// Validate and set default limit
	if opts.Limit <= 0 || opts.Limit > maxMessageLimit {
		opts.Limit = maxMessageLimit
	}

	if opts.ThreadTS != "" {
		// Retrieve thread messages
		return s.getThreadMessages(ctx, opts)
	}

	// Retrieve channel messages
	return s.getChannelMessages(ctx, opts)
}

// getThreadMessages retrieves messages from a specific thread
func (s *MessageHistoryService) getThreadMessages(ctx context.Context, opts MessageHistoryOptions) ([]slack.Message, error) {
	params := &slack.GetConversationRepliesParameters{
		ChannelID: opts.ChannelID,
		Timestamp: opts.ThreadTS,
		Limit:     opts.Limit,
	}

	messages, _, _, err := s.slackClient.GetConversationRepliesContext(ctx, params)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get thread messages",
			goerr.V("channelID", opts.ChannelID),
			goerr.V("threadTS", opts.ThreadTS),
		)
	}

	return messages, nil
}

// getChannelMessages retrieves messages from a channel (non-thread messages)
func (s *MessageHistoryService) getChannelMessages(ctx context.Context, opts MessageHistoryOptions) ([]slack.Message, error) {
	params := &slack.GetConversationHistoryParameters{
		ChannelID: opts.ChannelID,
		Limit:     opts.Limit,
	}

	// Set oldest time if provided
	if opts.OldestTime != nil {
		params.Oldest = strconv.FormatInt(opts.OldestTime.Unix(), 10)
	}

	history, err := s.slackClient.GetConversationHistoryContext(ctx, params)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get channel messages",
			goerr.V("channelID", opts.ChannelID),
		)
	}

	// Filter out thread messages (messages with ThreadTimestamp that are not the parent)
	var nonThreadMessages []slack.Message
	for _, msg := range history.Messages {
		// Include messages that are not part of a thread or are the parent message of a thread
		if msg.ThreadTimestamp == "" || msg.ThreadTimestamp == msg.Timestamp {
			nonThreadMessages = append(nonThreadMessages, msg)
		}
	}

	return nonThreadMessages, nil
}
