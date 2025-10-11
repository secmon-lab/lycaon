package graphql_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/controller/graphql"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack"
)

func TestIncidentResolverPrivateFiltering(t *testing.T) {
	// Setup
	repo := repository.NewMemory()
	config := &model.Config{
		Categories: []model.Category{
			{ID: "test-category", Name: "Test Category", Description: "Test"},
		},
		Severities: []model.Severity{
			{ID: "high", Name: "High", Level: 80, Description: "High severity"},
		},
	}

	// Create mock Slack client
	mockSlack := &mocks.SlackClientMock{
		AuthTestContextFunc: func(ctx context.Context) (*slack.AuthTestResponse, error) {
			return &slack.AuthTestResponse{TeamID: "T123"}, nil
		},
		CreateConversationFunc: func(ctx context.Context, params slack.CreateConversationParams) (*slack.Channel, error) {
			return &slack.Channel{GroupConversation: slack.GroupConversation{Conversation: slack.Conversation{ID: "C-TEST"}}}, nil
		},
	}

	incidentConfig := usecase.NewIncidentConfig(usecase.WithChannelPrefix("inc"))
	slackUIService := slackSvc.NewUIService(mockSlack, config)
	incidentUC := usecase.NewIncident(repo, mockSlack, slackUIService, config, nil, incidentConfig)

	useCases := &graphql.UseCases{
		IncidentUC: incidentUC,
		TaskUC:     nil,
		AuthUC:     nil,
	}

	resolver := graphql.NewResolver(repo, mockSlack, useCases, config)

	ctx := context.Background()

	// Create test incidents with random IDs to avoid test conflicts
	now := time.Now()
	nanoSuffix := now.UnixNano()

	publicIncidentID := types.IncidentID(nanoSuffix)
	publicIncident := &model.Incident{
		ID:          publicIncidentID,
		Title:       "Public Incident",
		Description: "This is public",
		CategoryID:  "test-category",
		SeverityID:  types.SeverityID("high"),
		ChannelID:   types.ChannelID(fmt.Sprintf("C-PUBLIC-%d", nanoSuffix)),
		ChannelName: types.ChannelName("public-channel"),
		Status:      types.IncidentStatusTriage,
		Private:     false,
		CreatedBy:   types.SlackUserID("U-CREATOR"),
	}
	gt.NoError(t, repo.PutIncident(ctx, publicIncident))

	privateIncidentID := types.IncidentID(nanoSuffix + 1)
	privateIncident := &model.Incident{
		ID:              privateIncidentID,
		Title:           "Private Incident Real Title",
		Description:     "This is private and sensitive",
		CategoryID:      "test-category",
		SeverityID:      types.SeverityID("high"),
		ChannelID:       types.ChannelID(fmt.Sprintf("C-PRIVATE-%d", nanoSuffix)),
		ChannelName:     types.ChannelName("private-channel"),
		Status:          types.IncidentStatusHandling,
		Private:         true,
		JoinedMemberIDs: []types.SlackUserID{"U-MEMBER1", "U-MEMBER2"},
		CreatedBy:       types.SlackUserID("U-CREATOR"),
	}
	gt.NoError(t, repo.PutIncident(ctx, privateIncident))

	t.Run("Public incident is visible to all users", func(t *testing.T) {
		// Create context with any user
		userCtx := model.WithAuthContext(ctx, &model.AuthContext{
			SlackUserID: "U-RANDOM-USER",
		})

		// Get incident
		incident, err := resolver.Query().Incident(userCtx, fmt.Sprintf("%d", publicIncidentID))
		gt.NoError(t, err)
		gt.V(t, incident).NotNil()
		gt.Equal(t, incident.Title, "Public Incident")
		gt.Equal(t, incident.Description, "This is public")

		// Verify viewerCanAccess for public incident
		canAccess, err := resolver.Incident().ViewerCanAccess(userCtx, incident)
		gt.NoError(t, err)
		gt.True(t, canAccess)
	})

	t.Run("Private incident is filtered for non-members", func(t *testing.T) {
		// Create context with non-member user
		nonMemberCtx := model.WithAuthContext(ctx, &model.AuthContext{
			SlackUserID: "U-NON-MEMBER",
		})

		// Get incident
		incident, err := resolver.Query().Incident(nonMemberCtx, fmt.Sprintf("%d", privateIncidentID))
		gt.NoError(t, err)
		gt.V(t, incident).NotNil()

		// Should be filtered
		gt.Equal(t, incident.Title, "Private Incident")
		gt.Equal(t, incident.Description, "")

		// Verify viewerCanAccess returns false for non-member
		canAccess, err := resolver.Incident().ViewerCanAccess(nonMemberCtx, incident)
		gt.NoError(t, err)
		gt.False(t, canAccess)
	})

	t.Run("Private incident is fully visible to members", func(t *testing.T) {
		// Create context with member user
		memberCtx := model.WithAuthContext(ctx, &model.AuthContext{
			SlackUserID: "U-MEMBER1",
		})

		// Get incident
		incident, err := resolver.Query().Incident(memberCtx, fmt.Sprintf("%d", privateIncidentID))
		gt.NoError(t, err)
		gt.V(t, incident).NotNil()

		// Should NOT be filtered
		gt.Equal(t, incident.Title, "Private Incident Real Title")
		gt.Equal(t, incident.Description, "This is private and sensitive")

		// Verify viewerCanAccess returns true for member
		canAccess, err := resolver.Incident().ViewerCanAccess(memberCtx, incident)
		gt.NoError(t, err)
		gt.True(t, canAccess)
	})

	t.Run("Incidents list filters private incidents for non-members", func(t *testing.T) {
		// Create context with non-member user
		nonMemberCtx := model.WithAuthContext(ctx, &model.AuthContext{
			SlackUserID: "U-NON-MEMBER",
		})

		// Get incidents list
		first := 10
		result, err := resolver.Query().Incidents(nonMemberCtx, &first, nil)
		gt.NoError(t, err)
		gt.V(t, result).NotNil()
		gt.Equal(t, len(result.Edges), 2)

		// Find incidents
		var publicInc, privateInc *model.Incident
		for _, edge := range result.Edges {
			if edge.Node.ID == publicIncidentID {
				publicInc = edge.Node
			} else if edge.Node.ID == privateIncidentID {
				privateInc = edge.Node
			}
		}

		// Public incident should be visible
		gt.V(t, publicInc).NotNil()
		gt.Equal(t, publicInc.Title, "Public Incident")
		gt.Equal(t, publicInc.Description, "This is public")

		// Private incident should be filtered
		gt.V(t, privateInc).NotNil()
		gt.Equal(t, privateInc.Title, "Private Incident")
		gt.Equal(t, privateInc.Description, "")
	})

	t.Run("Incidents list shows full details to members", func(t *testing.T) {
		// Create context with member user
		memberCtx := model.WithAuthContext(ctx, &model.AuthContext{
			SlackUserID: "U-MEMBER1",
		})

		// Get incidents list
		first := 10
		result, err := resolver.Query().Incidents(memberCtx, &first, nil)
		gt.NoError(t, err)
		gt.V(t, result).NotNil()
		gt.Equal(t, len(result.Edges), 2)

		// Find private incident
		var privateInc *model.Incident
		for _, edge := range result.Edges {
			if edge.Node.ID == privateIncidentID {
				privateInc = edge.Node
				break
			}
		}

		// Private incident should be fully visible
		gt.V(t, privateInc).NotNil()
		gt.Equal(t, privateInc.Title, "Private Incident Real Title")
		gt.Equal(t, privateInc.Description, "This is private and sensitive")
	})

	t.Run("No auth context returns incidents as-is", func(t *testing.T) {
		// No auth context (backward compatibility)
		noAuthCtx := context.Background()

		// Get incident
		incident, err := resolver.Query().Incident(noAuthCtx, fmt.Sprintf("%d", privateIncidentID))
		gt.NoError(t, err)
		gt.V(t, incident).NotNil()

		// Should return as-is (no filtering)
		gt.Equal(t, incident.Title, "Private Incident Real Title")
		gt.Equal(t, incident.Description, "This is private and sensitive")
	})
}

func TestTaskResolverPrivateFiltering(t *testing.T) {
	// Setup
	repo := repository.NewMemory()

	mockSlack := &mocks.SlackClientMock{
		GetUsersInConversationContextFunc: func(ctx context.Context, params *slack.GetUsersInConversationParameters) ([]string, string, error) {
			// Return member list for private channel
			if params.ChannelID == "C-PRIVATE" {
				return []string{"U-MEMBER1", "U-MEMBER2"}, "", nil
			}
			return []string{}, "", nil
		},
	}

	config := &model.Config{
		Categories: []model.Category{
			{
				ID:          "test-category",
				Name:        "Test Category",
				Description: "Test category for testing",
			},
		},
	}

	slackUIService := slackSvc.NewUIService(mockSlack, config)
	incidentConfig := usecase.NewIncidentConfig()
	incidentUC := usecase.NewIncident(repo, mockSlack, slackUIService, config, nil, incidentConfig)
	taskUC := usecase.NewTaskUseCase(repo, mockSlack)

	useCases := &graphql.UseCases{
		IncidentUC: incidentUC,
		TaskUC:     taskUC,
		AuthUC:     nil,
	}
	resolver := graphql.NewResolver(repo, mockSlack, useCases, config)

	ctx := context.Background()

	// Create test incidents with random IDs to avoid test conflicts
	now := time.Now()
	nanoSuffix := now.UnixNano()

	publicIncidentID := types.IncidentID(nanoSuffix)
	publicIncident := &model.Incident{
		ID:          publicIncidentID,
		Title:       "Public Incident",
		Description: "This is public",
		CategoryID:  "test-category",
		SeverityID:  types.SeverityID("high"),
		ChannelID:   types.ChannelID(fmt.Sprintf("C-PUBLIC-%d", nanoSuffix)),
		ChannelName: types.ChannelName("public-channel"),
		Status:      types.IncidentStatusTriage,
		Private:     false,
		CreatedBy:   types.SlackUserID("U-CREATOR"),
	}
	gt.NoError(t, repo.PutIncident(ctx, publicIncident))

	privateIncidentID := types.IncidentID(nanoSuffix + 1)
	privateIncident := &model.Incident{
		ID:              privateIncidentID,
		Title:           "Private Incident Real Title",
		Description:     "This is private and sensitive",
		CategoryID:      "test-category",
		SeverityID:      types.SeverityID("high"),
		ChannelID:       types.ChannelID("C-PRIVATE"),
		ChannelName:     types.ChannelName("private-channel"),
		Status:          types.IncidentStatusHandling,
		Private:         true,
		JoinedMemberIDs: []types.SlackUserID{"U-MEMBER1", "U-MEMBER2"},
		CreatedBy:       types.SlackUserID("U-CREATOR"),
	}
	gt.NoError(t, repo.PutIncident(ctx, privateIncident))

	// Create tasks for both incidents
	publicTaskID := types.TaskID(fmt.Sprintf("task-public-%d", nanoSuffix))
	publicTask := &model.Task{
		ID:          publicTaskID,
		IncidentID:  publicIncidentID,
		Title:       "Public Task",
		Description: "Task for public incident",
		Status:      "todo",
		CreatedBy:   types.SlackUserID("U-CREATOR"),
		CreatedAt:   now,
	}
	gt.NoError(t, repo.CreateTask(ctx, publicTask))

	privateTaskID := types.TaskID(fmt.Sprintf("task-private-%d", nanoSuffix))
	privateTask := &model.Task{
		ID:          privateTaskID,
		IncidentID:  privateIncidentID,
		Title:       "Private Task",
		Description: "Sensitive task information",
		Status:      "todo",
		CreatedBy:   types.SlackUserID("U-CREATOR"),
		CreatedAt:   now,
	}
	gt.NoError(t, repo.CreateTask(ctx, privateTask))

	t.Run("Non-member cannot access tasks of private incident", func(t *testing.T) {
		// Create context with non-member user
		nonMemberCtx := model.WithAuthContext(ctx, &model.AuthContext{
			SlackUserID: "U-NON-MEMBER",
		})

		// Try to get tasks for private incident
		tasks, err := resolver.Query().Tasks(nonMemberCtx, fmt.Sprintf("%d", privateIncidentID))
		gt.NoError(t, err)

		// Should return empty list for non-members
		gt.Equal(t, 0, len(tasks))
	})

	t.Run("Member can access tasks of private incident", func(t *testing.T) {
		// Create context with member user
		memberCtx := model.WithAuthContext(ctx, &model.AuthContext{
			SlackUserID: "U-MEMBER1",
		})

		// Get tasks for private incident
		tasks, err := resolver.Query().Tasks(memberCtx, fmt.Sprintf("%d", privateIncidentID))
		gt.NoError(t, err)
		gt.Equal(t, 1, len(tasks))
		gt.Equal(t, tasks[0].Title, "Private Task")
		gt.Equal(t, tasks[0].Description, "Sensitive task information")
	})

	t.Run("All users can access tasks of public incident", func(t *testing.T) {
		// Create context with random user
		userCtx := model.WithAuthContext(ctx, &model.AuthContext{
			SlackUserID: "U-RANDOM-USER",
		})

		// Get tasks for public incident
		tasks, err := resolver.Query().Tasks(userCtx, fmt.Sprintf("%d", publicIncidentID))
		gt.NoError(t, err)
		gt.Equal(t, 1, len(tasks))
		gt.Equal(t, tasks[0].Title, "Public Task")
	})

	t.Run("Non-member cannot access individual task of private incident", func(t *testing.T) {
		// Create context with non-member user
		nonMemberCtx := model.WithAuthContext(ctx, &model.AuthContext{
			SlackUserID: "U-NON-MEMBER",
		})

		// Try to get specific task from private incident
		task, err := resolver.Query().Task(nonMemberCtx, string(privateTaskID))

		// Should return error (access denied)
		gt.Error(t, err)
		gt.V(t, task).Nil()
	})
}
