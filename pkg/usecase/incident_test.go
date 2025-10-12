package usecase_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack"
)

// Helper function to create a test model.Config
func testConfig() *model.Config {
	return &model.Config{
		Categories: []model.Category{
			{
				ID:           "security_incident",
				Name:         "Security Incident",
				Description:  "Security-related incidents",
				InviteUsers:  []string{"@security-lead"},
				InviteGroups: []string{"@security-team"},
			},
			{
				ID:          "system_failure",
				Name:        "System Failure",
				Description: "System or service failures and outages",
			},
			{
				ID:          "unknown",
				Name:        "Unknown",
				Description: "Unknown incidents",
			},
		},
	}
}

func TestSyncIncidentMemberWithEvent(t *testing.T) {
	ctx := context.Background()

	t.Run("Skip sync for public incident", func(t *testing.T) {
		repo := &mocks.RepositoryMock{
			GetIncidentFunc: func(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
				return &model.Incident{
					ID:      id,
					Private: false,
					Title:   "Public Incident",
				}, nil
			},
		}
		slackClient := &mocks.SlackClientMock{}

		uc := usecase.NewIncident(repo, slackClient, nil, &model.Config{}, nil, &usecase.IncidentConfig{})

		// Should skip sync without calling Slack API
		err := uc.SyncIncidentMemberWithEvent(ctx, types.IncidentID(1), types.ChannelID("C123"), types.SlackUserID("U001"), true)
		gt.NoError(t, err)

		// Verify no Slack API calls were made
		gt.Equal(t, len(slackClient.GetUsersInConversationContextCalls()), 0)
	})

	t.Run("Skip sync when user joins but already a member", func(t *testing.T) {
		repo := &mocks.RepositoryMock{
			GetIncidentFunc: func(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
				return &model.Incident{
					ID:              id,
					Private:         true,
					JoinedMemberIDs: []types.SlackUserID{"U001", "U002"},
				}, nil
			},
		}
		slackClient := &mocks.SlackClientMock{}

		uc := usecase.NewIncident(repo, slackClient, nil, &model.Config{}, nil, &usecase.IncidentConfig{})

		// User U001 joins (but already a member) - should skip sync
		err := uc.SyncIncidentMemberWithEvent(ctx, types.IncidentID(1), types.ChannelID("C123"), types.SlackUserID("U001"), true)
		gt.NoError(t, err)

		// Verify no Slack API calls were made (needsSync = false)
		gt.Equal(t, len(slackClient.GetUsersInConversationContextCalls()), 0)
		// Verify no PutIncident calls
		gt.Equal(t, len(repo.PutIncidentCalls()), 0)
	})

	t.Run("Skip sync when user leaves but not a member", func(t *testing.T) {
		repo := &mocks.RepositoryMock{
			GetIncidentFunc: func(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
				return &model.Incident{
					ID:              id,
					Private:         true,
					JoinedMemberIDs: []types.SlackUserID{"U001", "U002"},
				}, nil
			},
		}
		slackClient := &mocks.SlackClientMock{}

		uc := usecase.NewIncident(repo, slackClient, nil, &model.Config{}, nil, &usecase.IncidentConfig{})

		// User U003 leaves (but not a member) - should skip sync
		err := uc.SyncIncidentMemberWithEvent(ctx, types.IncidentID(1), types.ChannelID("C123"), types.SlackUserID("U003"), false)
		gt.NoError(t, err)

		// Verify no Slack API calls were made (needsSync = false)
		gt.Equal(t, len(slackClient.GetUsersInConversationContextCalls()), 0)
		// Verify no PutIncident calls
		gt.Equal(t, len(repo.PutIncidentCalls()), 0)
	})

	t.Run("Sync when user joins and not a member", func(t *testing.T) {
		repo := &mocks.RepositoryMock{
			GetIncidentFunc: func(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
				return &model.Incident{
					ID:              id,
					Private:         true,
					JoinedMemberIDs: []types.SlackUserID{"U001"},
				}, nil
			},
			PutIncidentFunc: func(ctx context.Context, incident *model.Incident) error {
				return nil
			},
		}
		slackClient := &mocks.SlackClientMock{
			GetUsersInConversationContextFunc: func(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
				return []string{"U001", "U002"}, "", nil
			},
		}

		uc := usecase.NewIncident(repo, slackClient, nil, &model.Config{}, nil, &usecase.IncidentConfig{})

		// User U002 joins (new member) - should sync
		err := uc.SyncIncidentMemberWithEvent(ctx, types.IncidentID(1), types.ChannelID("C123"), types.SlackUserID("U002"), true)
		gt.NoError(t, err)

		// Verify Slack API was called (needsSync = true)
		gt.Equal(t, len(slackClient.GetUsersInConversationContextCalls()), 1)
		// Verify PutIncident was called to update member list
		gt.Equal(t, len(repo.PutIncidentCalls()), 1)

		// Verify the updated incident has correct member list
		putCall := repo.PutIncidentCalls()[0]
		gt.A(t, putCall.Incident.JoinedMemberIDs).Length(2)
	})

	t.Run("Sync when user leaves and is a member", func(t *testing.T) {
		repo := &mocks.RepositoryMock{
			GetIncidentFunc: func(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
				return &model.Incident{
					ID:              id,
					Private:         true,
					JoinedMemberIDs: []types.SlackUserID{"U001", "U002"},
				}, nil
			},
			PutIncidentFunc: func(ctx context.Context, incident *model.Incident) error {
				return nil
			},
		}
		slackClient := &mocks.SlackClientMock{
			GetUsersInConversationContextFunc: func(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
				return []string{"U001"}, "", nil
			},
		}

		uc := usecase.NewIncident(repo, slackClient, nil, &model.Config{}, nil, &usecase.IncidentConfig{})

		// User U002 leaves (existing member) - should sync
		err := uc.SyncIncidentMemberWithEvent(ctx, types.IncidentID(1), types.ChannelID("C123"), types.SlackUserID("U002"), false)
		gt.NoError(t, err)

		// Verify Slack API was called (needsSync = true)
		gt.Equal(t, len(slackClient.GetUsersInConversationContextCalls()), 1)
		// Verify PutIncident was called to update member list
		gt.Equal(t, len(repo.PutIncidentCalls()), 1)

		// Verify the updated incident has correct member list (U002 removed)
		putCall := repo.PutIncidentCalls()[0]
		gt.A(t, putCall.Incident.JoinedMemberIDs).Length(1)
	})
}
