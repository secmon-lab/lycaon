package llm_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/m-mizutani/gollem/mock"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/service/llm"
	"github.com/slack-go/slack"
)

func TestLLMService_AnalyzeIncident_Success(t *testing.T) {
	ctx := context.Background()

	// Create mock LLM client using gollem's built-in mock
	mockClient := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{
							"title": "Web server outage with 500 errors",
							"description": "Web server is down and returning 500 errors, preventing users from accessing the application. Multiple users confirmed the issue.",
							"category_id": "system_outage"
						}`},
					}, nil
				},
			}
			return mockSession, nil
		},
	}
	service := llm.NewLLMService(mockClient)

	// Setup test categories
	categories := &model.CategoriesConfig{
		Categories: []model.Category{
			{ID: "system_outage", Name: "System Outage", Description: "System down or unreachable"},
			{ID: "performance", Name: "Performance", Description: "System performance degradation"},
		},
	}

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

	// Call the method
	summary, err := service.AnalyzeIncident(ctx, messages, categories, nil)

	// Verify results
	gt.NoError(t, err)
	gt.NotNil(t, summary)
	gt.Equal(t, summary.Title, "Web server outage with 500 errors")
	gt.Equal(t, summary.Description, "Web server is down and returning 500 errors, preventing users from accessing the application. Multiple users confirmed the issue.")
	gt.Equal(t, summary.CategoryID, "system_outage")
}

func TestLLMService_AnalyzeIncident_InvalidJSON(t *testing.T) {
	ctx := context.Background()

	// Create mock LLM client that returns invalid JSON
	mockClient := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{"not valid json"},
					}, nil
				},
			}
			return mockSession, nil
		},
	}
	service := llm.NewLLMService(mockClient)

	categories := &model.CategoriesConfig{
		Categories: []model.Category{
			{ID: "system_outage", Name: "System Outage", Description: "System down or unreachable"},
		},
	}

	messages := []slack.Message{
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000001",
				User:      "U123456",
				Text:      "Test message",
			},
		},
	}

	// Call the method
	summary, err := service.AnalyzeIncident(ctx, messages, categories, nil)

	// Verify error occurs with correct tag
	gt.Error(t, err)
	gt.B(t, goerr.HasTag(err, llm.ErrTagInvalidJSON)).True()
	gt.Nil(t, summary)
}

func TestLLMService_AnalyzeIncident_MissingTitle(t *testing.T) {
	ctx := context.Background()

	// Create mock LLM client that returns JSON without title
	mockClient := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{
							"description": "Some description",
							"category_id": "system_outage"
						}`},
					}, nil
				},
			}
			return mockSession, nil
		},
	}
	service := llm.NewLLMService(mockClient)

	categories := &model.CategoriesConfig{
		Categories: []model.Category{
			{ID: "system_outage", Name: "System Outage", Description: "System down or unreachable"},
		},
	}

	messages := []slack.Message{
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000001",
				User:      "U123456",
				Text:      "Test message",
			},
		},
	}

	// Call the method
	summary, err := service.AnalyzeIncident(ctx, messages, categories, nil)

	// Verify error occurs with correct tag and field info
	gt.Error(t, err)
	gt.B(t, goerr.HasTag(err, llm.ErrTagMissingField)).True()
	// Check that the error contains field information
	values := goerr.Values(err)
	gt.V(t, values["field"]).Equal("title")
	gt.Nil(t, summary)
}

func TestLLMService_AnalyzeIncident_InvalidCategory(t *testing.T) {
	ctx := context.Background()

	// Create mock LLM client that returns invalid category
	mockClient := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{`{
							"title": "Test incident",
							"description": "Test description",
							"category_id": "nonexistent_category"
						}`},
					}, nil
				},
			}
			return mockSession, nil
		},
	}
	service := llm.NewLLMService(mockClient)

	categories := &model.CategoriesConfig{
		Categories: []model.Category{
			{ID: "system_outage", Name: "System Outage", Description: "System down or unreachable"},
		},
	}

	messages := []slack.Message{
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000001",
				User:      "U123456",
				Text:      "Test message",
			},
		},
	}

	// Call the method
	summary, err := service.AnalyzeIncident(ctx, messages, categories, nil)

	// Verify that invalid category falls back to "unknown"
	gt.NoError(t, err)
	gt.NotNil(t, summary)
	gt.Equal(t, summary.CategoryID, "unknown")
}

func TestLLMService_AnalyzeIncident_EmptyResponse(t *testing.T) {
	ctx := context.Background()

	// Create mock LLM client that returns empty response
	mockClient := &mock.LLMClientMock{
		NewSessionFunc: func(ctx context.Context, options ...gollem.SessionOption) (gollem.Session, error) {
			mockSession := &mock.SessionMock{
				GenerateContentFunc: func(ctx context.Context, input ...gollem.Input) (*gollem.Response, error) {
					return &gollem.Response{
						Texts: []string{},
					}, nil
				},
			}
			return mockSession, nil
		},
	}
	service := llm.NewLLMService(mockClient)

	categories := &model.CategoriesConfig{
		Categories: []model.Category{
			{ID: "system_outage", Name: "System Outage", Description: "System down or unreachable"},
		},
	}

	messages := []slack.Message{
		{
			Msg: slack.Msg{
				Timestamp: "1234567890.000001",
				User:      "U123456",
				Text:      "Test message",
			},
		},
	}

	// Call the method
	summary, err := service.AnalyzeIncident(ctx, messages, categories, nil)

	// Verify error occurs with correct tag
	gt.Error(t, err)
	gt.B(t, goerr.HasTag(err, llm.ErrTagEmptyResponse)).True()
	gt.Nil(t, summary)
}
