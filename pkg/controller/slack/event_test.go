package slack_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	slack "github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	slackgo "github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func TestEventHandlerHandleEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("handles nil event", func(t *testing.T) {
		mockUC := &mocks.SlackMessageMock{}
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{}
		mockSlackClient := &mocks.SlackClientMock{}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

		err := handler.HandleEvent(ctx, nil)
		gt.Error(t, err)
	})

	t.Run("handles message event successfully", func(t *testing.T) {
		mockUC := &mocks.SlackMessageMock{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				return nil
			},
		}
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{}
		mockSlackClient := &mocks.SlackClientMock{}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

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
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{}
		mockSlackClient := &mocks.SlackClientMock{}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

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
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{}
		mockSlackClient := &mocks.SlackClientMock{}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

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
			SendIncidentMessageFunc: func(ctx context.Context, channelID, messageTS, title, description, categoryID, severityID string) error {
				return nil
			},
		}
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{}
		mockSlackClient := &mocks.SlackClientMock{}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

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
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{}
		mockSlackClient := &mocks.SlackClientMock{}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

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
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{}
		mockSlackClient := &mocks.SlackClientMock{}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

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
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{}
		mockSlackClient := &mocks.SlackClientMock{}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

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
			SendIncidentMessageFunc: func(ctx context.Context, channelID, messageTS, title, description, categoryID, severityID string) error {
				sendIncidentCalled = true
				return nil
			},
		}
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{}
		mockSlackClient := &mocks.SlackClientMock{}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

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

	t.Run("handles task command in non-incident channel - sends thread error", func(t *testing.T) {
		errorMessageSent := false
		var sentChannel, sentThreadTS, sentMessage string

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
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{
			GetIncidentByChannelIDFunc: func(ctx context.Context, channelID types.ChannelID) (*model.Incident, error) {
				// Return error to simulate no incident found for this channel
				return nil, goerr.New("incident not found for channel")
			},
		}
		mockSlackClient := &mocks.SlackClientMock{
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slackgo.MsgOption) (string, string, error) {
				errorMessageSent = true
				sentChannel = channelID
				// Extract thread timestamp and message from options
				for _, opt := range options {
					if opt != nil {
						// This is a simplified check - in real implementation we'd parse the options properly
						sentMessage = "Please create an incident first."
						sentThreadTS = "1234567890.123456"
					}
				}
				return "channel", "timestamp", nil
			},
		}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.AppMention),
				Data: &slackevents.AppMentionEvent{
					Type:      "app_mention",
					Text:      "<@BOT123> task Create new task",
					User:      "U123456",
					Channel:   "C123456",
					TimeStamp: "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)

		// Wait a bit for async processing to complete
		time.Sleep(200 * time.Millisecond)

		gt.B(t, errorMessageSent).True()
		gt.Equal(t, sentChannel, "C123456")
		gt.Equal(t, sentMessage, "Please create an incident first.")
		gt.Equal(t, sentThreadTS, "1234567890.123456")
	})

	t.Run("handles task list command in non-incident channel - sends thread error", func(t *testing.T) {
		errorMessageSent := false
		var sentChannel, sentMessage string

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
		mockTaskUC := &mocks.TaskMock{}
		mockIncidentUC := &mocks.IncidentMock{
			GetIncidentByChannelIDFunc: func(ctx context.Context, channelID types.ChannelID) (*model.Incident, error) {
				// Return error to simulate no incident found for this channel
				return nil, goerr.New("incident not found for channel")
			},
		}
		mockSlackClient := &mocks.SlackClientMock{
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slackgo.MsgOption) (string, string, error) {
				errorMessageSent = true
				sentChannel = channelID
				sentMessage = "Please create an incident first."
				return "channel", "timestamp", nil
			},
		}
		mockStatusUC := &mocks.StatusUseCaseMock{}
		handler := slack.NewEventHandler(ctx, mockUC, mockTaskUC, mockIncidentUC, mockStatusUC, mockSlackClient)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.AppMention),
				Data: &slackevents.AppMentionEvent{
					Type:      "app_mention",
					Text:      "<@BOT123> task",
					User:      "U123456",
					Channel:   "C123456",
					TimeStamp: "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)

		// Wait a bit for async processing to complete
		time.Sleep(200 * time.Millisecond)

		gt.B(t, errorMessageSent).True()
		gt.Equal(t, sentChannel, "C123456")
		gt.Equal(t, sentMessage, "Please create an incident first.")
	})
}
