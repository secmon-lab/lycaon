package slack

import (
	"context"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/slack-go/slack/slackevents"
)

// EventHandler handles Slack events
type EventHandler struct {
	messageUC interfaces.SlackMessage
}

// NewEventHandler creates a new event handler
func NewEventHandler(ctx context.Context, messageUC interfaces.SlackMessage) *EventHandler {
	return &EventHandler{
		messageUC: messageUC,
	}
}

// HandleEvent handles a Slack event
func (h *EventHandler) HandleEvent(ctx context.Context, event *slackevents.EventsAPIEvent) error {
	if event == nil {
		return goerr.New("event is nil")
	}

	ctxlog.From(ctx).Debug("Handling Slack event",
		"type", event.Type,
		"innerEvent", event.InnerEvent.Type,
	)

	// Handle different event types
	switch ev := event.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		return h.handleMessageEvent(ctx, ev)

	case *slackevents.AppMentionEvent:
		return h.handleAppMentionEvent(ctx, ev)

	default:
		ctxlog.From(ctx).Debug("Unhandled event type",
			"type", event.InnerEvent.Type,
		)
		return nil
	}
}

// handleMessageEvent handles message events
func (h *EventHandler) handleMessageEvent(ctx context.Context, event *slackevents.MessageEvent) error {
	// Skip bot messages to prevent loops
	if event.BotID != "" {
		ctxlog.From(ctx).Debug("Skipping bot message", "botID", event.BotID)
		return nil
	}

	// Skip messages without text
	if event.Text == "" {
		ctxlog.From(ctx).Debug("Skipping empty message")
		return nil
	}

	// Skip thread messages (optional - depends on requirements)
	if event.ThreadTimeStamp != "" && event.ThreadTimeStamp != event.TimeStamp {
		ctxlog.From(ctx).Debug("Skipping thread message",
			"threadTS", event.ThreadTimeStamp,
			"messageTS", event.TimeStamp,
		)
		return nil
	}

	ctxlog.From(ctx).Info("Processing message event",
		"user", event.User,
		"channel", event.Channel,
		"text", event.Text,
		"ts", event.TimeStamp,
	)

	// Save the message first
	if err := h.messageUC.ProcessMessage(ctx, event); err != nil {
		return goerr.Wrap(err, "failed to save message")
	}

	return nil
}

// handleAppMentionEvent handles app mention events
func (h *EventHandler) handleAppMentionEvent(ctx context.Context, event *slackevents.AppMentionEvent) error {
	ctxlog.From(ctx).Info("App mentioned",
		"user", event.User,
		"channel", event.Channel,
		"text", event.Text,
		"ts", event.TimeStamp,
	)

	// Convert AppMentionEvent to MessageEvent for processing
	messageEvent := &slackevents.MessageEvent{
		Type:            "message",
		User:            event.User,
		Text:            event.Text,
		TimeStamp:       event.TimeStamp,
		ThreadTimeStamp: event.ThreadTimeStamp,
		Channel:         event.Channel,
	}

	// Save the message
	if err := h.messageUC.ProcessMessage(ctx, messageEvent); err != nil {
		return goerr.Wrap(err, "failed to save message")
	}

	// Convert to domain model for incident trigger check
	message := &model.Message{
		ID:        types.MessageID(messageEvent.ClientMsgID),
		UserID:    types.SlackUserID(event.User),
		ChannelID: types.ChannelID(event.Channel),
		Text:      event.Text,
		EventTS:   types.EventTS(event.TimeStamp),
	}

	// First check if it's a basic incident trigger (before any heavy processing)
	if h.messageUC.IsBasicIncidentTrigger(ctx, message) {
		// Send immediate context message to acknowledge the command
		if err := h.messageUC.SendProcessingMessage(ctx, event.Channel, event.TimeStamp); err != nil {
			ctxlog.From(ctx).Warn("Failed to send processing message",
				"error", err,
				"channel", event.Channel,
			)
		}
	}

	// Check if message triggers incident creation (this may do LLM analysis)
	cmd := h.messageUC.ParseIncidentCommand(ctx, message)
	if cmd.IsIncidentTrigger {
		ctxlog.From(ctx).Info("Incident trigger detected from mention",
			"user", event.User,
			"channel", event.Channel,
			"title", cmd.Title,
		)

		// Send incident creation prompt with title and description
		if err := h.messageUC.SendIncidentMessage(ctx, event.Channel, event.TimeStamp, cmd.Title, cmd.Description, cmd.CategoryID); err != nil {
			ctxlog.From(ctx).Error("Failed to send incident prompt",
				"error", err,
				"channel", event.Channel,
			)
		}
	}
	// No response for non-incident mentions - just save the message

	return nil
}
