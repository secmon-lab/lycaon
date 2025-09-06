package slack

import (
	"context"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack/slackevents"
)

// EventHandler handles Slack events
type EventHandler struct {
	messageUC usecase.SlackMessageUseCase
}

// NewEventHandler creates a new event handler
func NewEventHandler(ctx context.Context, messageUC usecase.SlackMessageUseCase) *EventHandler {
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

	// Process the message
	response, err := h.messageUC.SaveAndRespond(ctx, event)
	if err != nil {
		return goerr.Wrap(err, "failed to process message")
	}

	ctxlog.From(ctx).Info("Message processed successfully",
		"user", event.User,
		"response", response,
	)

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

	// Process as a message
	response, err := h.messageUC.SaveAndRespond(ctx, messageEvent)
	if err != nil {
		return goerr.Wrap(err, "failed to process app mention")
	}

	ctxlog.From(ctx).Info("App mention processed successfully",
		"user", event.User,
		"response", response,
	)

	return nil
}
