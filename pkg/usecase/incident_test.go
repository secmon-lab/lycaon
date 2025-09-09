package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack"
)


func TestIncidentUseCaseCreateIncident(t *testing.T) {
	ctx := context.Background()

	t.Run("Successful incident creation", func(t *testing.T) {
		// Use memory repository for testing
		repo := repository.NewMemory()

		// Create mock Slack client with default behavior
		mockSlack := &mocks.SlackClientMock{
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-NEW-INCIDENT",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: channelID,
						},
						Purpose: slack.Purpose{
							Value: purpose,
						},
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}

		// Create use case with mock
		uc := usecase.NewIncident(repo, mockSlack)

		// Create an incident
		incident, err := uc.CreateIncident(
			ctx,
			"database outage",
			"",
			"C-ORIGIN",
			"general",
			"U-CREATOR",
		)

		// Verify incident was created successfully
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()
		gt.Equal(t, 1, incident.ID)
		gt.Equal(t, "inc-1-database-outage", incident.ChannelName)
		gt.Equal(t, "database outage", incident.Title)
		gt.Equal(t, "C-ORIGIN", incident.OriginChannelID)
		gt.Equal(t, "general", incident.OriginChannelName)
		gt.Equal(t, "U-CREATOR", incident.CreatedBy)

		// Verify channel ID was set by mock
		gt.Equal(t, "C-NEW-INCIDENT", incident.ChannelID)

		// Verify incident was saved to repository
		savedIncident, err := repo.GetIncident(ctx, 1)
		gt.NoError(t, err).Required()
		gt.Equal(t, incident.ID, savedIncident.ID)
	})

	t.Run("Multiple incidents get sequential IDs", func(t *testing.T) {
		repo := repository.NewMemory()
		mockSlack := &mocks.SlackClientMock{
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-NEW-INCIDENT",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: channelID,
						},
						Purpose: slack.Purpose{
							Value: purpose,
						},
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}
		uc := usecase.NewIncident(repo, mockSlack)

		// Create first incident
		incident1, _ := uc.CreateIncident(ctx, "api error", "", "C1", "channel1", "U1")
		gt.Equal(t, 1, incident1.ID)
		gt.Equal(t, "inc-1-api-error", incident1.ChannelName)

		// Create second incident
		incident2, _ := uc.CreateIncident(ctx, "database down", "", "C2", "channel2", "U2")
		gt.Equal(t, 2, incident2.ID)
		gt.Equal(t, "inc-2-database-down", incident2.ChannelName)

		// Create third incident
		incident3, _ := uc.CreateIncident(ctx, "", "", "C3", "channel3", "U3")
		gt.Equal(t, 3, incident3.ID)
		gt.Equal(t, "inc-3", incident3.ChannelName)
	})

	t.Run("GetIncident retrieves correct incident", func(t *testing.T) {
		repo := repository.NewMemory()
		mockSlack := &mocks.SlackClientMock{
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-NEW-INCIDENT",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: channelID,
						},
						Purpose: slack.Purpose{
							Value: purpose,
						},
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}
		uc := usecase.NewIncident(repo, mockSlack)

		// Create an incident
		created, err := uc.CreateIncident(ctx, "test incident", "", "C-TEST", "test-channel", "U-TEST")
		gt.NoError(t, err).Required()

		// Retrieve the incident
		retrieved, err := uc.GetIncident(ctx, created.ID.Int())
		gt.NoError(t, err).Required()
		gt.Equal(t, created.ID, retrieved.ID)
		gt.Equal(t, created.ChannelName, retrieved.ChannelName)
		gt.Equal(t, created.OriginChannelID, retrieved.OriginChannelID)
		gt.Equal(t, created.CreatedBy, retrieved.CreatedBy)
	})

	t.Run("GetIncident returns error for non-existent ID", func(t *testing.T) {
		repo := repository.NewMemory()
		mockSlack := &mocks.SlackClientMock{
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-NEW-INCIDENT",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: channelID,
						},
						Purpose: slack.Purpose{
							Value: purpose,
						},
					},
				}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}
		uc := usecase.NewIncident(repo, mockSlack)

		// Try to get non-existent incident
		incident, err := uc.GetIncident(ctx, 999)
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("incident not found")
	})
}

func TestIncidentUseCaseWithMockRepository(t *testing.T) {
	ctx := context.Background()

	t.Run("Repository error handling", func(t *testing.T) {
		// Create mock repository
		mockRepo := &mocks.RepositoryMock{
			GetNextIncidentNumberFunc: func(ctx context.Context) (types.IncidentID, error) {
				return 0, goerr.New("database error")
			},
		}

		mockSlack := &mocks.SlackClientMock{}
		uc := usecase.NewIncident(mockRepo, mockSlack)

		// Try to create incident - should fail due to repository error
		incident, err := uc.CreateIncident(ctx, "test", "", "C1", "channel1", "U1")
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("failed to get next incident number")
	})
}
