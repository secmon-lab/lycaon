package slack_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	slack "github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/slack-go/slack/slackevents"
)

func TestEventHandlerHandleEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("handles nil event", func(t *testing.T) {
		mockUC := &mocks.SlackMessageMock{}
		handler := slack.NewEventHandler(ctx, mockUC)

		err := handler.HandleEvent(ctx, nil)
		gt.Error(t, err)
	})

	t.Run("handles message event successfully", func(t *testing.T) {
		mockUC := &mocks.SlackMessageMock{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					Type: "message",
					Text: "test message",
					User: "U123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
	})

	t.Run("skips bot messages", func(t *testing.T) {
		processCalled := false
		mockUC := &mocks.SlackMessageMock{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				processCalled = true
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					Type:  "message",
					Text:  "bot message",
					BotID: "B123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.B(t, processCalled).False()
	})

	t.Run("skips empty messages", func(t *testing.T) {
		processCalled := false
		mockUC := &mocks.SlackMessageMock{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				processCalled = true
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					Type: "message",
					Text: "",
					User: "U123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.B(t, processCalled).False()
	})

	t.Run("handles app mention event with inc command", func(t *testing.T) {
		mockUC := &mocks.SlackMessageMock{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				return nil
			},
			IsBasicIncidentTriggerFunc: func(ctx context.Context, message *model.Message) bool {
				return true
			},
			SendProcessingMessageFunc: func(ctx context.Context, channelID, messageTS string) error {
				return nil
			},
			ParseIncidentCommandFunc: func(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
				return interfaces.IncidentCommand{
					IsIncidentTrigger: true,
					Title:             "Test Incident",
					Description:       "Test Description",
					CategoryID:        "test-category",
				}
			},
			SendIncidentMessageFunc: func(ctx context.Context, channelID, messageTS, title, description, categoryID string) error {
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.AppMention),
				Data: &slackevents.AppMentionEvent{
					Type:      "app_mention",
					Text:      "<@BOT123> inc Test Incident",
					User:      "U123456",
					Channel:   "C123456",
					TimeStamp: "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
	})

	t.Run("handles app mention event without inc command", func(t *testing.T) {
		mockUC := &mocks.SlackMessageMock{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				return nil
			},
			IsBasicIncidentTriggerFunc: func(ctx context.Context, message *model.Message) bool {
				return false
			},
			ParseIncidentCommandFunc: func(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
				return interfaces.IncidentCommand{
					IsIncidentTrigger: false,
				}
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.AppMention),
				Data: &slackevents.AppMentionEvent{
					Type:      "app_mention",
					Text:      "<@BOT123> hello",
					User:      "U123456",
					Channel:   "C123456",
					TimeStamp: "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
	})

	t.Run("handles thread messages", func(t *testing.T) {
		processCalled := false
		mockUC := &mocks.SlackMessageMock{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				processCalled = true
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					Type:            "message",
					Text:            "thread message",
					User:            "U123456",
					TimeStamp:       "1234567890.123456",
					ThreadTimeStamp: "1234567890.000000",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.B(t, processCalled).False()
	})

	t.Run("handles unsupported event type", func(t *testing.T) {
		mockUC := &mocks.SlackMessageMock{}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: "unsupported_event",
				Data: "unsupported data",
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
	})

	t.Run("handles app mention event without inc command - no incident message sent", func(t *testing.T) {
		sendIncidentCalled := false
		mockUC := &mocks.SlackMessageMock{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				return nil
			},
			IsBasicIncidentTriggerFunc: func(ctx context.Context, message *model.Message) bool {
				return false
			},
			ParseIncidentCommandFunc: func(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
				return interfaces.IncidentCommand{
					IsIncidentTrigger: false,
				}
			},
			SendIncidentMessageFunc: func(ctx context.Context, channelID, messageTS, title, description, categoryID string) error {
				sendIncidentCalled = true
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.AppMention),
				Data: &slackevents.AppMentionEvent{
					Type:      "app_mention",
					Text:      "<@BOT123> hello",
					User:      "U123456",
					Channel:   "C123456",
					TimeStamp: "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.B(t, sendIncidentCalled).False()
	})
}

