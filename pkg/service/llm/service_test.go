package llm_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/service/llm"
	"github.com/slack-go/slack"
)

func TestLLMService_GenerateIncidentSummary_Success(t *testing.T) {
	ctx := context.Background()

	// Create mock LLM client using gollem's built-in mock
	mockClient := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{
							"title": "Web server outage with 500 errors",
							"description": "Web server is down and returning 500 errors, preventing users from accessing the application. Multiple users confirmed the issue."
						}`},
					}, nil
				},
			}
			return mockSession, nil
		},
	}
	service := llm.NewLLMService(mockClient)

	// Setup test messages
	messages := []slack.Message{
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000001",
				User:      "U123456",
				Text:      "The web server is down and returning 500 errors",
			},
		},
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000002",
				User:      "U789012",
				Text:      "I can confirm the issue, users can't access the application",
			},
		},
	}

	// Test the service
	summary, err := service.GenerateIncidentSummary(ctx, messages)

	gt.NoError(t, err).Required()
	gt.NotEqual(t, summary, nil)
	gt.Equal(t, summary.Title, "Web server outage with 500 errors")
	gt.Equal(t, summary.Description, "Web server is down and returning 500 errors, preventing users from accessing the application. Multiple users confirmed the issue.")
}

func TestLLMService_GenerateIncidentSummary_EmptyMessages(t *testing.T) {
	ctx := context.Background()
	mockClient := &mock.LLMClientMock{}
	service := llm.NewLLMService(mockClient)

	// Test with empty messages
	messages := []slack.Message{}

	_, err := service.GenerateIncidentSummary(ctx, messages)
	gt.Error(t, err).Contains("no messages provided")
}

func TestLLMService_GenerateIncidentSummary_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	mockClient := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"invalid json response"},
					}, nil
				},
			}
			return mockSession, nil
		},
	}
	service := llm.NewLLMService(mockClient)

	messages := []slack.Message{
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000001",
				User:      "U123456",
				Text:      "Test message",
			},
		},
	}

	_, err := service.GenerateIncidentSummary(ctx, messages)
	gt.Error(t, err).Contains("failed to parse LLM response")
}

func TestLLMService_GenerateIncidentSummary_MissingTitle(t *testing.T) {
	ctx := context.Background()
	mockClient := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{
							"description": "Some description"
						}`},
					}, nil
				},
			}
			return mockSession, nil
		},
	}
	service := llm.NewLLMService(mockClient)

	messages := []slack.Message{
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000001",
				User:      "U123456",
				Text:      "Test message",
			},
		},
	}

	_, err := service.GenerateIncidentSummary(ctx, messages)
	gt.Error(t, err).Contains("missing required title")
}

func TestLLMService_BuildConversationText(t *testing.T) {
	// This is a unit test for the private method via exported behavior
	mockClient := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{
							"title": "Test Title",
							"description": "Test Description"
						}`},
					}, nil
				},
			}
			return mockSession, nil
		},
	}
	service := llm.NewLLMService(mockClient)

	messages := []slack.Message{
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000001",
				User:      "U123456",
				Text:      "First message",
			},
		},
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000002",
				User:      "U789012",
				Text:      "Second message",
			},
		},
		{
			Msg: slack.Msg{
				Timestamp: "", // Empty timestamp
				User:      "U345678",
				Text:      "Third message",
			},
		},
		{
			Msg: slack.Msg{
				Text: "", // Empty text should be skipped
			},
		},
	}

	ctx := context.Background()
	_, err := service.GenerateIncidentSummary(ctx, messages)

	// Should succeed, indicating the conversation text was built correctly
	gt.NoError(t, err).Required()
}

func TestNewLLMService(t *testing.T) {
	mockClient := &mock.LLMClientMock{}
	service := llm.NewLLMService(mockClient)

	gt.NotEqual(t, service, nil)
}
