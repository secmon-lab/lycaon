package slack_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/slack-go/slack/slackevents"
)

// MockSlackMessageUseCase mocks the SlackMessageUseCase interface
type MockSlackMessageUseCase struct {
	ProcessMessageFunc       func(ctx context.Context, event *slackevents.MessageEvent) error
	SaveAndRespondFunc       func(ctx context.Context, event *slackevents.MessageEvent) (string, error)
	GenerateResponseFunc     func(ctx context.Context, message *model.Message) (string, error)
	ParseIncidentCommandFunc func(ctx context.Context, message *model.Message) interfaces.IncidentCommand
	SendIncidentMessageFunc  func(ctx context.Context, channelID, messageTS, title string) error
}

func (m *MockSlackMessageUseCase) ProcessMessage(ctx context.Context, event *slackevents.MessageEvent) error {
	if m.ProcessMessageFunc != nil {
		return m.ProcessMessageFunc(ctx, event)
	}
	return nil
}

func (m *MockSlackMessageUseCase) SaveAndRespond(ctx context.Context, event *slackevents.MessageEvent) (string, error) {
	if m.SaveAndRespondFunc != nil {
		return m.SaveAndRespondFunc(ctx, event)
	}
	return "mock response", nil
}

func (m *MockSlackMessageUseCase) GenerateResponse(ctx context.Context, message *model.Message) (string, error) {
	if m.GenerateResponseFunc != nil {
		return m.GenerateResponseFunc(ctx, message)
	}
	return "mock response", nil
}

func (m *MockSlackMessageUseCase) ParseIncidentCommand(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
	if m.ParseIncidentCommandFunc != nil {
		return m.ParseIncidentCommandFunc(ctx, message)
	}
	return interfaces.IncidentCommand{IsIncidentTrigger: false, Title: ""}
}

func (m *MockSlackMessageUseCase) SendIncidentMessage(ctx context.Context, channelID, messageTS, title string) error {
	if m.SendIncidentMessageFunc != nil {
		return m.SendIncidentMessageFunc(ctx, channelID, messageTS, title)
	}
	return nil
}

func TestEventHandlerHandleEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("Handle nil event", func(t *testing.T) {
		mockUC := &MockSlackMessageUseCase{}
		handler := slack.NewEventHandler(ctx, mockUC)

		err := handler.HandleEvent(ctx, nil)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("event is nil")
	})

	t.Run("Handle message event", func(t *testing.T) {
		var processedMessage *slackevents.MessageEvent
		mockUC := &MockSlackMessageUseCase{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				processedMessage = event
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					ClientMsgID: "msg-001",
					User:        "U12345",
					Channel:     "C12345",
					Text:        "Test message",
					TimeStamp:   "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.V(t, processedMessage).NotNil()
		gt.Equal(t, "Test message", processedMessage.Text)
	})

	t.Run("Skip bot messages", func(t *testing.T) {
		var processedMessage *slackevents.MessageEvent
		mockUC := &MockSlackMessageUseCase{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				processedMessage = event
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					BotID:     "B12345",
					User:      "U12345",
					Channel:   "C12345",
					Text:      "Bot message",
					TimeStamp: "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.V(t, processedMessage).Nil() // Message should not be processed
	})

	t.Run("Handle app mention event", func(t *testing.T) {
		var savedEvent *slackevents.MessageEvent
		mockUC := &MockSlackMessageUseCase{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				savedEvent = event
				return nil
			},
			ParseIncidentCommandFunc: func(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
				// Not an incident trigger
				return interfaces.IncidentCommand{IsIncidentTrigger: false}
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.AppMention),
				Data: &slackevents.AppMentionEvent{
					User:      "U12345",
					Channel:   "C12345",
					Text:      "<@U99999> help me",
					TimeStamp: "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.V(t, savedEvent).NotNil()
		gt.Equal(t, "<@U99999> help me", savedEvent.Text)
	})

	t.Run("Handle app mention with incident trigger", func(t *testing.T) {
		var savedEvent *slackevents.MessageEvent
		var incidentMessageSent bool
		mockUC := &MockSlackMessageUseCase{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				savedEvent = event
				return nil
			},
			ParseIncidentCommandFunc: func(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
				// This is an incident trigger
				return interfaces.IncidentCommand{
					IsIncidentTrigger: true,
					Title:             "database issue",
				}
			},
			SendIncidentMessageFunc: func(ctx context.Context, channelID, messageTS, title string) error {
				incidentMessageSent = true
				gt.Equal(t, "C12345", channelID)
				gt.Equal(t, "1234567890.123456", messageTS)
				gt.Equal(t, "database issue", title)
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.AppMention),
				Data: &slackevents.AppMentionEvent{
					User:      "U12345",
					Channel:   "C12345",
					Text:      "<@U99999> inc database issue",
					TimeStamp: "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.V(t, savedEvent).NotNil()
		gt.Equal(t, "<@U99999> inc database issue", savedEvent.Text)
		gt.True(t, incidentMessageSent)
	})
}

func TestEventHandlerIncidentTrigger(t *testing.T) {
	ctx := context.Background()

	t.Run("Regular message does not trigger incident", func(t *testing.T) {
		var incidentMessageSent bool

		mockUC := &MockSlackMessageUseCase{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				return nil
			},
			ParseIncidentCommandFunc: func(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
				// This shouldn't be called for regular messages anymore
				return interfaces.IncidentCommand{IsIncidentTrigger: true, Title: "something happened"}
			},
			SendIncidentMessageFunc: func(ctx context.Context, channelID, messageTS, title string) error {
				incidentMessageSent = true
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					ClientMsgID: "msg-inc-1",
					User:        "U12345",
					Channel:     "C12345",
					Text:        "inc something happened",
					TimeStamp:   "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		// Incident should NOT be triggered from regular message events
		gt.False(t, incidentMessageSent)
	})

	t.Run("No trigger for normal message", func(t *testing.T) {
		var incidentMessageSent bool

		mockUC := &MockSlackMessageUseCase{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				return nil
			},
			ParseIncidentCommandFunc: func(ctx context.Context, message *model.Message) interfaces.IncidentCommand {
				return interfaces.IncidentCommand{IsIncidentTrigger: false, Title: ""}
			},
			SendIncidentMessageFunc: func(ctx context.Context, channelID, messageTS, title string) error {
				incidentMessageSent = true
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					ClientMsgID: "msg-normal-001",
					User:        "U12345",
					Channel:     "C12345",
					Text:        "normal message",
					TimeStamp:   "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.False(t, incidentMessageSent)
	})

	t.Run("Skip empty messages", func(t *testing.T) {
		var processedMessage *slackevents.MessageEvent
		mockUC := &MockSlackMessageUseCase{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				processedMessage = event
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					ClientMsgID: "msg-empty-001",
					User:        "U12345",
					Channel:     "C12345",
					Text:        "",
					TimeStamp:   "1234567890.123456",
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.V(t, processedMessage).Nil() // Empty message should not be processed
	})

	t.Run("Skip thread messages", func(t *testing.T) {
		var processedMessage *slackevents.MessageEvent
		mockUC := &MockSlackMessageUseCase{
			ProcessMessageFunc: func(ctx context.Context, event *slackevents.MessageEvent) error {
				processedMessage = event
				return nil
			},
		}
		handler := slack.NewEventHandler(ctx, mockUC)

		event := &slackevents.EventsAPIEvent{
			Type: slackevents.CallbackEvent,
			InnerEvent: slackevents.EventsAPIInnerEvent{
				Type: string(slackevents.Message),
				Data: &slackevents.MessageEvent{
					ClientMsgID:     "msg-thread-001",
					User:            "U12345",
					Channel:         "C12345",
					Text:            "Thread reply",
					TimeStamp:       "1234567890.123456",
					ThreadTimeStamp: "1234567890.000000", // Different from TimeStamp
				},
			},
		}

		err := handler.HandleEvent(ctx, event)
		gt.NoError(t, err)
		gt.V(t, processedMessage).Nil() // Thread message should not be processed
	})
}
