package slack_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	slackgo "github.com/slack-go/slack"
)

// MockTaskUseCase mocks the Task interface
type MockTaskUseCase struct {
	CreateTaskFunc     func(ctx context.Context, incidentID types.IncidentID, title string, createdBy types.SlackUserID, channelID types.ChannelID, messageTS string) (*model.Task, error)
	ListTasksFunc      func(ctx context.Context, incidentID types.IncidentID) ([]*model.Task, error)
	UpdateTaskFunc     func(ctx context.Context, taskID types.TaskID, req interfaces.TaskUpdateRequest) (*model.Task, error)
	CompleteTaskFunc   func(ctx context.Context, taskID types.TaskID) (*model.Task, error)
	UncompleteTaskFunc func(ctx context.Context, taskID types.TaskID) (*model.Task, error)
	GetTaskFunc        func(ctx context.Context, taskID types.TaskID) (*model.Task, error)
}

func (m *MockTaskUseCase) CreateTask(ctx context.Context, incidentID types.IncidentID, title string, createdBy types.SlackUserID, channelID types.ChannelID, messageTS string) (*model.Task, error) {
	if m.CreateTaskFunc != nil {
		return m.CreateTaskFunc(ctx, incidentID, title, createdBy, channelID, messageTS)
	}
	return &model.Task{
		ID:         types.TaskID("task-1"),
		IncidentID: incidentID,
		Title:      title,
		Status:     model.TaskStatusIncompleted,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
	}, nil
}

func (m *MockTaskUseCase) ListTasks(ctx context.Context, incidentID types.IncidentID) ([]*model.Task, error) {
	if m.ListTasksFunc != nil {
		return m.ListTasksFunc(ctx, incidentID)
	}
	return []*model.Task{}, nil
}

func (m *MockTaskUseCase) UpdateTask(ctx context.Context, taskID types.TaskID, req interfaces.TaskUpdateRequest) (*model.Task, error) {
	if m.UpdateTaskFunc != nil {
		return m.UpdateTaskFunc(ctx, taskID, req)
	}
	return &model.Task{
		ID:        taskID,
		Title:     "Updated Task",
		Status:    model.TaskStatusCompleted,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockTaskUseCase) CompleteTask(ctx context.Context, taskID types.TaskID) (*model.Task, error) {
	if m.CompleteTaskFunc != nil {
		return m.CompleteTaskFunc(ctx, taskID)
	}
	return &model.Task{
		ID:        taskID,
		Title:     "Completed Task",
		Status:    model.TaskStatusCompleted,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockTaskUseCase) UncompleteTask(ctx context.Context, taskID types.TaskID) (*model.Task, error) {
	if m.UncompleteTaskFunc != nil {
		return m.UncompleteTaskFunc(ctx, taskID)
	}
	return &model.Task{
		ID:        taskID,
		Title:     "Uncompleted Task",
		Status:    model.TaskStatusIncompleted,
		CreatedAt: time.Now(),
	}, nil
}

func (m *MockTaskUseCase) GetTask(ctx context.Context, taskID types.TaskID) (*model.Task, error) {
	if m.GetTaskFunc != nil {
		return m.GetTaskFunc(ctx, taskID)
	}
	return &model.Task{
		ID:        taskID,
		Title:     "Test Task",
		Status:    model.TaskStatusIncompleted,
		CreatedAt: time.Now(),
	}, nil
}

// MockIncidentUseCase mocks the IncidentUseCase interface
type MockIncidentUseCase struct {
	CreateIncidentFunc                  func(ctx context.Context, req *model.CreateIncidentRequest) (*model.Incident, error)
	GetIncidentFunc                     func(ctx context.Context, id int) (*model.Incident, error)
	GetIncidentByChannelIDFunc          func(ctx context.Context, channelID types.ChannelID) (*model.Incident, error)
	CreateIncidentFromInteractionFunc   func(ctx context.Context, originChannelID, title, userID string) (*model.Incident, error)
	HandleCreateIncidentActionFunc      func(ctx context.Context, requestID, userID string) (*model.Incident, error)
	HandleCreateIncidentWithDetailsFunc func(ctx context.Context, requestID, title, description, categoryID, userID string) (*model.Incident, error)
	GetIncidentRequestFunc              func(ctx context.Context, requestID string) (*model.IncidentRequest, error)
	HandleEditIncidentActionFunc        func(ctx context.Context, requestID, userID, triggerID string) error
	HandleCreateIncidentActionAsyncFunc func(ctx context.Context, requestID, userID, channelID string)
}

func (m *MockIncidentUseCase) CreateIncident(ctx context.Context, req *model.CreateIncidentRequest) (*model.Incident, error) {
	if m.CreateIncidentFunc != nil {
		return m.CreateIncidentFunc(ctx, req)
	}

	channelName := "inc-1"
	if req.Title != "" {
		channelName = "inc-1-" + req.Title
	}

	return &model.Incident{
		ID:              1,
		Title:           req.Title,
		ChannelName:     types.ChannelName(channelName),
		OriginChannelID: types.ChannelID(req.OriginChannelID),
		CreatedBy:       types.SlackUserID(req.CreatedBy),
	}, nil
}

func (m *MockIncidentUseCase) GetIncident(ctx context.Context, id int) (*model.Incident, error) {
	if m.GetIncidentFunc != nil {
		return m.GetIncidentFunc(ctx, id)
	}
	return nil, goerr.New("incident not found")
}

func (m *MockIncidentUseCase) GetIncidentByChannelID(ctx context.Context, channelID types.ChannelID) (*model.Incident, error) {
	if m.GetIncidentByChannelIDFunc != nil {
		return m.GetIncidentByChannelIDFunc(ctx, channelID)
	}
	return nil, goerr.New("incident not found")
}

func (m *MockIncidentUseCase) CreateIncidentFromInteraction(ctx context.Context, originChannelID, title, userID string) (*model.Incident, error) {
	if m.CreateIncidentFromInteractionFunc != nil {
		return m.CreateIncidentFromInteractionFunc(ctx, originChannelID, title, userID)
	}

	// Default to calling CreateIncident with a dummy channel name
	return m.CreateIncident(ctx, &model.CreateIncidentRequest{
		Title:             title,
		Description:       "",
		CategoryID:        "unknown",
		OriginChannelID:   originChannelID,
		OriginChannelName: "general",
		CreatedBy:         userID,
	})
}

func (m *MockIncidentUseCase) HandleCreateIncidentAction(ctx context.Context, requestID, userID string) (*model.Incident, error) {
	if m.HandleCreateIncidentActionFunc != nil {
		return m.HandleCreateIncidentActionFunc(ctx, requestID, userID)
	}

	// Default implementation
	return &model.Incident{
		ID:                1,
		ChannelID:         types.ChannelID("C-INC-001"),
		ChannelName:       types.ChannelName("inc-1"),
		Title:             "Test Incident",
		OriginChannelID:   types.ChannelID("C67890"),
		OriginChannelName: types.ChannelName("general"),
		CreatedBy:         types.SlackUserID(userID),
	}, nil
}

func (m *MockIncidentUseCase) HandleCreateIncidentWithDetails(ctx context.Context, requestID, title, description, categoryID, userID string) (*model.Incident, error) {
	if m.HandleCreateIncidentWithDetailsFunc != nil {
		return m.HandleCreateIncidentWithDetailsFunc(ctx, requestID, title, description, categoryID, userID)
	}

	// Default implementation
	return &model.Incident{
		ID:                1,
		ChannelID:         types.ChannelID("C-INC-001"),
		ChannelName:       types.ChannelName("inc-1"),
		Title:             title,
		Description:       description,
		OriginChannelID:   types.ChannelID("C67890"),
		OriginChannelName: types.ChannelName("general"),
		CreatedBy:         types.SlackUserID(userID),
	}, nil
}

func (m *MockIncidentUseCase) GetIncidentRequest(ctx context.Context, requestID string) (*model.IncidentRequest, error) {
	if m.GetIncidentRequestFunc != nil {
		return m.GetIncidentRequestFunc(ctx, requestID)
	}

	// Default implementation
	return &model.IncidentRequest{
		ID:        types.IncidentRequestID(requestID),
		ChannelID: types.ChannelID("C67890"),
		Title:     "Test Incident",
	}, nil
}

func (m *MockIncidentUseCase) HandleEditIncidentAction(ctx context.Context, requestID, userID, triggerID string) error {
	if m.HandleEditIncidentActionFunc != nil {
		return m.HandleEditIncidentActionFunc(ctx, requestID, userID, triggerID)
	}

	// Default implementation - just return nil
	return nil
}

func (m *MockIncidentUseCase) HandleCreateIncidentActionAsync(ctx context.Context, requestID, userID, channelID string) {
	if m.HandleCreateIncidentActionAsyncFunc != nil {
		m.HandleCreateIncidentActionAsyncFunc(ctx, requestID, userID, channelID)
		return
	}

	// Default implementation - just return
}

func TestInteractionHandlerHandleInteraction(t *testing.T) {
	ctx := context.Background()

	t.Run("Handle invalid JSON payload", func(t *testing.T) {
		mockUC := &MockIncidentUseCase{}
		mockTaskUC := &MockTaskUseCase{}
		mockSlack := &mocks.SlackClientMock{}
		slackInteractionUC := usecase.NewSlackInteraction(mockUC, mockTaskUC, mockSlack)
		handler := slack.NewInteractionHandler(ctx, slackInteractionUC)

		invalidPayload := []byte("invalid json")
		err := handler.HandleInteraction(ctx, invalidPayload)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to unmarshal")
	})

	t.Run("Handle create_incident action", func(t *testing.T) {
		var createdIncident *model.Incident
		created := make(chan bool, 1)
		mockUC := &MockIncidentUseCase{
			HandleCreateIncidentActionAsyncFunc: func(ctx context.Context, requestID, userID, channelID string) {
				createdIncident = &model.Incident{
					ID:                1,
					ChannelID:         types.ChannelID("C-INC-001"),
					ChannelName:       types.ChannelName("inc-1"),
					Title:             "Test Incident",
					OriginChannelID:   types.ChannelID("C67890"),
					OriginChannelName: types.ChannelName("general"),
					CreatedBy:         types.SlackUserID(userID),
				}
				created <- true
			},
		}

		mockTaskUC := &MockTaskUseCase{}
		mockSlack := &mocks.SlackClientMock{}
		slackInteractionUC := usecase.NewSlackInteraction(mockUC, mockTaskUC, mockSlack)
		handler := slack.NewInteractionHandler(ctx, slackInteractionUC)

		interaction := slackgo.InteractionCallback{
			Type: slackgo.InteractionTypeBlockActions,
			User: slackgo.User{
				ID:   "U12345",
				Name: "testuser",
			},
			Team: slackgo.Team{
				ID: "T12345",
			},
			Channel: slackgo.Channel{
				GroupConversation: slackgo.GroupConversation{
					Conversation: slackgo.Conversation{
						ID: "C12345",
					},
				},
			},
			ActionCallback: slackgo.ActionCallbacks{
				BlockActions: []*slackgo.BlockAction{
					{
						ActionID: "create_incident",
						BlockID:  "incident_creation",
						Value:    "test-request-id-123", // Mock request ID
						Type:     slackgo.ActionType("button"),
					},
				},
			},
		}

		payload, err := json.Marshal(interaction)
		gt.NoError(t, err).Required()

		err = handler.HandleInteraction(ctx, payload)
		gt.NoError(t, err).Required()

		// Wait for async processing to complete
		select {
		case <-created:
			gt.V(t, createdIncident).NotNil()
			gt.Equal(t, "C67890", createdIncident.OriginChannelID)
			gt.Equal(t, "U12345", createdIncident.CreatedBy)
		case <-time.After(1 * time.Second):
			t.Fatal("Incident creation did not complete within timeout")
		}
	})

	t.Run("Handle empty request ID", func(t *testing.T) {
		mockUC := &MockIncidentUseCase{}
		mockTaskUC := &MockTaskUseCase{}
		mockSlack := &mocks.SlackClientMock{}
		slackInteractionUC := usecase.NewSlackInteraction(mockUC, mockTaskUC, mockSlack)
		handler := slack.NewInteractionHandler(ctx, slackInteractionUC)

		interaction := slackgo.InteractionCallback{
			Type: slackgo.InteractionTypeBlockActions,
			User: slackgo.User{
				ID:   "U12345",
				Name: "testuser",
			},
			Team: slackgo.Team{
				ID: "T12345",
			},
			Channel: slackgo.Channel{
				GroupConversation: slackgo.GroupConversation{
					Conversation: slackgo.Conversation{
						ID: "C12345",
					},
				},
			},
			ActionCallback: slackgo.ActionCallbacks{
				BlockActions: []*slackgo.BlockAction{
					{
						ActionID: "create_incident",
						BlockID:  "incident_creation",
						Value:    "", // Empty request ID
						Type:     slackgo.ActionType("button"),
					},
				},
			},
		}

		payload, err := json.Marshal(interaction)
		gt.NoError(t, err).Required()

		err = handler.HandleInteraction(ctx, payload)
		gt.Error(t, err)
		if err != nil {
			gt.S(t, err.Error()).Contains("empty request ID")
		}
	})

	t.Run("Handle incident creation failure", func(t *testing.T) {
		failed := make(chan bool, 1)
		mockUC := &MockIncidentUseCase{
			HandleCreateIncidentActionAsyncFunc: func(ctx context.Context, requestID, userID, channelID string) {
				failed <- true
				// In real implementation, this would handle the error internally
			},
		}

		mockTaskUC := &MockTaskUseCase{}
		mockSlack := &mocks.SlackClientMock{}
		slackInteractionUC := usecase.NewSlackInteraction(mockUC, mockTaskUC, mockSlack)
		handler := slack.NewInteractionHandler(ctx, slackInteractionUC)

		interaction := slackgo.InteractionCallback{
			Type: slackgo.InteractionTypeBlockActions,
			User: slackgo.User{
				ID:   "U12345",
				Name: "testuser",
			},
			Team: slackgo.Team{
				ID: "T12345",
			},
			Channel: slackgo.Channel{
				GroupConversation: slackgo.GroupConversation{
					Conversation: slackgo.Conversation{
						ID: "C12345",
					},
				},
			},
			ActionCallback: slackgo.ActionCallbacks{
				BlockActions: []*slackgo.BlockAction{
					{
						ActionID: "create_incident",
						BlockID:  "incident_creation",
						Value:    "test-request-id-123", // Mock request ID
						Type:     slackgo.ActionType("button"),
					},
				},
			},
		}

		payload, err := json.Marshal(interaction)
		gt.NoError(t, err).Required()

		err = handler.HandleInteraction(ctx, payload)
		gt.NoError(t, err).Required() // The handler itself should not return error (responds immediately)

		// Wait for async processing to complete
		select {
		case <-failed:
			// Test passes if incident creation was attempted
		case <-time.After(1 * time.Second):
			t.Fatal("Incident creation was not attempted within timeout")
		}
	})

	t.Run("Handle unknown action", func(t *testing.T) {
		mockUC := &MockIncidentUseCase{}
		mockTaskUC := &MockTaskUseCase{}
		mockSlack := &mocks.SlackClientMock{}
		slackInteractionUC := usecase.NewSlackInteraction(mockUC, mockTaskUC, mockSlack)
		handler := slack.NewInteractionHandler(ctx, slackInteractionUC)

		interaction := slackgo.InteractionCallback{
			Type: slackgo.InteractionTypeBlockActions,
			User: slackgo.User{
				ID:   "U12345",
				Name: "testuser",
			},
			Team: slackgo.Team{
				ID: "T12345",
			},
			ActionCallback: slackgo.ActionCallbacks{
				BlockActions: []*slackgo.BlockAction{
					{
						ActionID: "unknown_action",
						BlockID:  "unknown_block",
						Value:    "some_value",
						Type:     slackgo.ActionType("button"),
					},
				},
			},
		}

		payload, err := json.Marshal(interaction)
		gt.NoError(t, err).Required()

		// Should not error for unknown actions, just log and continue
		err = handler.HandleInteraction(ctx, payload)
		gt.NoError(t, err).Required()
	})

	t.Run("Handle shortcut interaction", func(t *testing.T) {
		mockUC := &MockIncidentUseCase{}
		mockTaskUC := &MockTaskUseCase{}
		mockSlack := &mocks.SlackClientMock{}
		slackInteractionUC := usecase.NewSlackInteraction(mockUC, mockTaskUC, mockSlack)
		handler := slack.NewInteractionHandler(ctx, slackInteractionUC)

		interaction := slackgo.InteractionCallback{
			Type:       slackgo.InteractionTypeShortcut,
			CallbackID: "create_incident_shortcut",
			TriggerID:  "trigger_123",
			User: slackgo.User{
				ID:   "U12345",
				Name: "testuser",
			},
			Team: slackgo.Team{
				ID: "T12345",
			},
		}

		payload, err := json.Marshal(interaction)
		gt.NoError(t, err).Required()

		// Should handle shortcut without error (even if not fully implemented)
		err = handler.HandleInteraction(ctx, payload)
		gt.NoError(t, err).Required()
	})

	t.Run("Handle view submission", func(t *testing.T) {
		mockUC := &MockIncidentUseCase{}
		mockTaskUC := &MockTaskUseCase{}
		mockSlack := &mocks.SlackClientMock{}
		slackInteractionUC := usecase.NewSlackInteraction(mockUC, mockTaskUC, mockSlack)
		handler := slack.NewInteractionHandler(ctx, slackInteractionUC)

		interaction := slackgo.InteractionCallback{
			Type: slackgo.InteractionTypeViewSubmission,
			View: slackgo.View{
				ID:              "view_123",
				CallbackID:      "incident_creation_modal",
				PrivateMetadata: "test-request-id-123", // Add request ID in private metadata
				State: &slackgo.ViewState{
					Values: map[string]map[string]slackgo.BlockAction{
						"title_block": {
							"title_input": slackgo.BlockAction{
								Type:  "plain_text_input",
								Value: "Test Incident",
							},
						},
						"description_block": {
							"description_input": slackgo.BlockAction{
								Type:  "plain_text_input",
								Value: "Test Description",
							},
						},
						"category_block": {
							"category_select": slackgo.BlockAction{
								Type: "static_select",
								SelectedOption: slackgo.OptionBlockObject{
									Value: "system_failure",
								},
							},
						},
					},
				},
			},
			User: slackgo.User{
				ID:   "U12345",
				Name: "testuser",
			},
			Team: slackgo.Team{
				ID: "T12345",
			},
		}

		payload, err := json.Marshal(interaction)
		gt.NoError(t, err).Required()

		// Should handle view submission without error
		err = handler.HandleInteraction(ctx, payload)
		gt.NoError(t, err).Required()
	})

	t.Run("Handle view submission with missing fields", func(t *testing.T) {
		mockUC := &MockIncidentUseCase{}
		mockTaskUC := &MockTaskUseCase{}
		mockSlack := &mocks.SlackClientMock{}
		slackInteractionUC := usecase.NewSlackInteraction(mockUC, mockTaskUC, mockSlack)
		handler := slack.NewInteractionHandler(ctx, slackInteractionUC)

		interaction := slackgo.InteractionCallback{
			Type: slackgo.InteractionTypeViewSubmission,
			View: slackgo.View{
				ID:              "view_123",
				CallbackID:      "incident_creation_modal",
				PrivateMetadata: "test-request-id-123",
				State: &slackgo.ViewState{
					Values: map[string]map[string]slackgo.BlockAction{
						// Missing title_block and description_block entirely
					},
				},
			},
			User: slackgo.User{
				ID:   "U12345",
				Name: "testuser",
			},
			Team: slackgo.Team{
				ID: "T12345",
			},
		}

		payload, err := json.Marshal(interaction)
		gt.NoError(t, err).Required()

		// Should handle missing fields gracefully and return error for empty title
		err = handler.HandleInteraction(ctx, payload)
		gt.Error(t, err)
		if err != nil {
			gt.S(t, err.Error()).Contains("incident title is required")
		}
	})

	t.Run("Handle view closed", func(t *testing.T) {
		mockUC := &MockIncidentUseCase{}
		mockTaskUC := &MockTaskUseCase{}
		mockSlack := &mocks.SlackClientMock{}
		slackInteractionUC := usecase.NewSlackInteraction(mockUC, mockTaskUC, mockSlack)
		handler := slack.NewInteractionHandler(ctx, slackInteractionUC)

		interaction := slackgo.InteractionCallback{
			Type: slackgo.InteractionTypeViewClosed,
			View: slackgo.View{
				ID: "view_123",
			},
			User: slackgo.User{
				ID:   "U12345",
				Name: "testuser",
			},
			Team: slackgo.Team{
				ID: "T12345",
			},
		}

		payload, err := json.Marshal(interaction)
		gt.NoError(t, err).Required()

		// Should handle view closed without error
		err = handler.HandleInteraction(ctx, payload)
		gt.NoError(t, err).Required()
	})
}
