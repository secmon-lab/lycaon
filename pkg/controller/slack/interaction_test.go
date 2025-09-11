package slack_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/controller/slack"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	slackgo "github.com/slack-go/slack"
)



func TestInteractionHandlerHandleInteraction(t *testing.T) {
	ctx := context.Background()

	t.Run("Handle invalid JSON payload", func(t *testing.T) {
		mockUC := &mocks.IncidentMock{}
		mockTaskUC := &mocks.TaskMock{}
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
		mockUC := &mocks.IncidentMock{
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

		mockTaskUC := &mocks.TaskMock{}
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
		mockUC := &mocks.IncidentMock{}
		mockTaskUC := &mocks.TaskMock{}
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
		mockUC := &mocks.IncidentMock{
			HandleCreateIncidentActionAsyncFunc: func(ctx context.Context, requestID, userID, channelID string) {
				failed <- true
				// In real implementation, this would handle the error internally
			},
		}

		mockTaskUC := &mocks.TaskMock{}
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
		mockUC := &mocks.IncidentMock{}
		mockTaskUC := &mocks.TaskMock{}
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
		mockUC := &mocks.IncidentMock{}
		mockTaskUC := &mocks.TaskMock{}
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
		mockUC := &mocks.IncidentMock{}
		mockTaskUC := &mocks.TaskMock{}
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
		mockUC := &mocks.IncidentMock{}
		mockTaskUC := &mocks.TaskMock{}
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
		mockUC := &mocks.IncidentMock{}
		mockTaskUC := &mocks.TaskMock{}
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
