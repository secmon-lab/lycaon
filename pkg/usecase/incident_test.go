package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
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

		// Create use case with mock and default categories
		categories := model.GetDefaultCategories()
		uc := usecase.NewIncident(repo, mockSlack, categories, nil)

		// Create an incident
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "database outage",
			Description:       "",
			CategoryID:        "unknown",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "general",
			CreatedBy:         "U-CREATOR",
		})

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
		categories := model.GetDefaultCategories()
		uc := usecase.NewIncident(repo, mockSlack, categories, nil)

		// Create first incident
		incident1, _ := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "api error",
			Description:       "",
			CategoryID:        "unknown",
			OriginChannelID:   "C1",
			OriginChannelName: "channel1",
			CreatedBy:         "U1",
		})
		gt.Equal(t, 1, incident1.ID)
		gt.Equal(t, "inc-1-api-error", incident1.ChannelName)

		// Create second incident
		incident2, _ := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "database down",
			Description:       "",
			CategoryID:        "unknown",
			OriginChannelID:   "C2",
			OriginChannelName: "channel2",
			CreatedBy:         "U2",
		})
		gt.Equal(t, 2, incident2.ID)
		gt.Equal(t, "inc-2-database-down", incident2.ChannelName)

		// Create third incident
		incident3, _ := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "",
			Description:       "",
			CategoryID:        "unknown",
			OriginChannelID:   "C3",
			OriginChannelName: "channel3",
			CreatedBy:         "U3",
		})
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
		categories := model.GetDefaultCategories()
		uc := usecase.NewIncident(repo, mockSlack, categories, nil)

		// Create an incident
		created, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "test incident",
			Description:       "",
			CategoryID:        "unknown",
			OriginChannelID:   "C-TEST",
			OriginChannelName: "test-channel",
			CreatedBy:         "U-TEST",
		})
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
		categories := model.GetDefaultCategories()
		uc := usecase.NewIncident(repo, mockSlack, categories, nil)

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
		categories := model.GetDefaultCategories()
		uc := usecase.NewIncident(mockRepo, mockSlack, categories, nil)

		// Try to create incident - should fail due to repository error
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "test",
			Description:       "",
			CategoryID:        "unknown",
			OriginChannelID:   "C1",
			OriginChannelName: "channel1",
			CreatedBy:         "U1",
		})
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("failed to get next incident number")
	})

	t.Run("Create incident with category invitations", func(t *testing.T) {
		// Use memory repository for testing
		repo := repository.NewMemory()

		// Track invited users
		var invitedUsers []string

		// Create mock Slack client
		mockSlack := &mocks.SlackClientMock{
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-INCIDENT-WITH-INVITES",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				// Capture invited users
				invitedUsers = append(invitedUsers, users...)
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}

		// Create mock Invite that tracks invitations
		mockInvite := &mocks.InviteMock{
			InviteUsersByListFunc: func(ctx context.Context, users []string, groups []string, channelID types.ChannelID) (*model.InvitationResult, error) {
				// Verify correct category invitations are passed
				gt.Equal(t, 1, len(users))
				gt.Equal(t, "@security-lead", users[0])
				gt.Equal(t, 1, len(groups))
				gt.Equal(t, "@security-team", groups[0])
				
				return &model.InvitationResult{
					Details: []model.InviteDetail{
						{
							UserID:       "U-SECURITY-LEAD",
							Username:     "@security-lead",
							SourceConfig: "@security-lead",
							Status:       "success",
						},
						{
							UserID:       "U-SECURITY-MEMBER1",
							Username:     "",
							SourceConfig: "@security-team",
							Status:       "success",
						},
					},
				}, nil
			},
		}

		// Create categories with invitations
		categories := &model.CategoriesConfig{
			Categories: []model.Category{
				{
					ID:           "security_incident",
					Name:         "Security Incident",
					Description:  "Security-related incidents",
					InviteUsers:  []string{"@security-lead"},
					InviteGroups: []string{"@security-team"},
				},
				{
					ID:          "unknown",
					Name:        "Unknown",
					Description: "Unknown incidents",
				},
			},
		}

		// Create use case with mock invite
		uc := usecase.NewIncident(repo, mockSlack, categories, mockInvite)

		// Create an incident with security_incident category
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "security breach",
			Description:       "Potential security incident detected",
			CategoryID:        "security_incident",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "general",
			CreatedBy:         "U-CREATOR",
		})

		// Verify incident was created successfully
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()
		gt.Equal(t, "security_incident", incident.CategoryID)

		// Verify invite was called with correct parameters
		gt.Equal(t, 1, len(mockInvite.InviteUsersByListCalls()))
		inviteCall := mockInvite.InviteUsersByListCalls()[0]
		gt.Equal(t, []string{"@security-lead"}, inviteCall.Users)
		gt.Equal(t, []string{"@security-team"}, inviteCall.Groups)
		gt.Equal(t, incident.ChannelID, inviteCall.ChannelID)
	})

	t.Run("Create incident with no category invitations", func(t *testing.T) {
		// Use memory repository for testing
		repo := repository.NewMemory()

		// Create mock Slack client
		mockSlack := &mocks.SlackClientMock{
			CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
				return &slack.Channel{
					GroupConversation: slack.GroupConversation{
						Conversation: slack.Conversation{
							ID: "C-INCIDENT-NO-INVITES",
						},
						Name: params.ChannelName,
					},
				}, nil
			},
			SetPurposeOfConversationContextFunc: func(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			InviteUsersToConversationFunc: func(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
				return &slack.Channel{}, nil
			},
			PostMessageFunc: func(ctx context.Context, channelID string, options ...slack.MsgOption) (string, string, error) {
				return "channel", "timestamp", nil
			},
		}

		// Create mock Invite that should NOT be called
		mockInvite := &mocks.InviteMock{
			InviteUsersByListFunc: func(ctx context.Context, users []string, groups []string, channelID types.ChannelID) (*model.InvitationResult, error) {
				t.Fatal("InviteUsersByList should not be called for unknown category")
				return nil, nil
			},
		}

		// Create categories without invitations for unknown
		categories := &model.CategoriesConfig{
			Categories: []model.Category{
				{
					ID:          "unknown",
					Name:        "Unknown",
					Description: "Unknown incidents",
				},
			},
		}

		// Create use case with mock invite
		uc := usecase.NewIncident(repo, mockSlack, categories, mockInvite)

		// Create an incident with unknown category (no invitations)
		incident, err := uc.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             "unknown issue",
			Description:       "Some unknown issue",
			CategoryID:        "unknown",
			OriginChannelID:   "C-ORIGIN",
			OriginChannelName: "general",
			CreatedBy:         "U-CREATOR",
		})

		// Verify incident was created successfully
		gt.NoError(t, err).Required()
		gt.V(t, incident).NotNil()
		gt.Equal(t, "unknown", incident.CategoryID)

		// Verify invite was NOT called
		gt.Equal(t, 0, len(mockInvite.InviteUsersByListCalls()))
	})
}
