package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
)

func TestStatusUseCase_UpdateStatus(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	mockSlack := &mocks.SlackClientMock{}

	statusUC := usecase.NewStatusUseCase(repo, mockSlack)

	// Create a test incident first
	incidentID := types.IncidentID(time.Now().UnixNano())
	incident, err := model.NewIncident(
		incidentID,
		"Test Incident",
		"Test Description",
		"test_category",
		types.ChannelID("C123456"),
		types.ChannelName("test-channel"),
		types.TeamID("T123456"),
		types.SlackUserID("U123456"),
		false, // not initial triage
	)
	gt.NoError(t, err)

	err = repo.PutIncident(ctx, incident)
	gt.NoError(t, err)

	// Create initial status history
	initialHistory, err := model.NewStatusHistory(incident.ID, incident.Status, incident.CreatedBy, "Incident created")
	gt.NoError(t, err)
	err = repo.AddStatusHistory(ctx, initialHistory)
	gt.NoError(t, err)

	// Test status update
	err = statusUC.UpdateStatus(ctx, incidentID, types.IncidentStatusMonitoring, "U789012", "Moving to monitoring phase")
	gt.NoError(t, err)

	// Verify status was updated
	updatedIncident, err := repo.GetIncident(ctx, incidentID)
	gt.NoError(t, err)
	gt.Equal(t, updatedIncident.Status, types.IncidentStatusMonitoring)

	// Verify status history was recorded
	histories, err := repo.GetStatusHistories(ctx, incidentID)
	gt.NoError(t, err)

	// Should have initial status + new status
	if len(histories) < 2 {
		t.Errorf("Expected at least 2 history entries, got %d", len(histories))
		return
	}

	// Check the latest history entry
	latestHistory := histories[len(histories)-1]
	gt.Equal(t, latestHistory.Status, types.IncidentStatusMonitoring)
	gt.Equal(t, latestHistory.ChangedBy, types.SlackUserID("U789012"))
	gt.Equal(t, latestHistory.Note, "Moving to monitoring phase")
}

func TestStatusUseCase_UpdateStatus_SameStatus(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	mockSlack := &mocks.SlackClientMock{}

	statusUC := usecase.NewStatusUseCase(repo, mockSlack)

	// Create a test incident
	incidentID := types.IncidentID(time.Now().UnixNano())
	incident, err := model.NewIncident(
		incidentID,
		"Test Incident",
		"Test Description",
		"test_category",
		types.ChannelID("C123456"),
		types.ChannelName("test-channel"),
		types.TeamID("T123456"),
		types.SlackUserID("U123456"),
		false,
	)
	gt.NoError(t, err)

	err = repo.PutIncident(ctx, incident)
	gt.NoError(t, err)

	// Create initial status history
	initialHistory, err := model.NewStatusHistory(incident.ID, incident.Status, incident.CreatedBy, "Incident created")
	gt.NoError(t, err)
	err = repo.AddStatusHistory(ctx, initialHistory)
	gt.NoError(t, err)

	// Try to update to the same status (should fail)
	err = statusUC.UpdateStatus(ctx, incidentID, types.IncidentStatusHandling, "U789012", "Same status")
	gt.Error(t, err)
}

func TestStatusUseCase_GetStatusHistory(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	mockSlack := &mocks.SlackClientMock{}

	statusUC := usecase.NewStatusUseCase(repo, mockSlack)

	// Create a test incident
	incidentID := types.IncidentID(time.Now().UnixNano())
	incident, err := model.NewIncident(
		incidentID,
		"Test Incident",
		"Test Description",
		"test_category",
		types.ChannelID("C123456"),
		types.ChannelName("test-channel"),
		types.TeamID("T123456"),
		types.SlackUserID("U123456"),
		false,
	)
	gt.NoError(t, err)

	err = repo.PutIncident(ctx, incident)
	gt.NoError(t, err)

	// Create initial status history
	initialHistory, err := model.NewStatusHistory(incident.ID, incident.Status, incident.CreatedBy, "Incident created")
	gt.NoError(t, err)
	err = repo.AddStatusHistory(ctx, initialHistory)
	gt.NoError(t, err)

	// Create a test user
	testUser := model.NewUser("U123456", "Test User", "test@example.com")
	err = repo.SaveUser(ctx, testUser)
	gt.NoError(t, err)

	// Add some status changes
	err = statusUC.UpdateStatus(ctx, incidentID, types.IncidentStatusMonitoring, "U123456", "Moving to monitoring")
	gt.NoError(t, err)

	err = statusUC.UpdateStatus(ctx, incidentID, types.IncidentStatusClosed, "U123456", "Incident resolved")
	gt.NoError(t, err)

	// Get status history with user information
	historiesWithUser, err := statusUC.GetStatusHistory(ctx, incidentID)
	gt.NoError(t, err)

	// Should have at least 3 entries (Initial + 2 updates)
	if len(historiesWithUser) < 3 {
		t.Errorf("Expected at least 3 history entries, got %d", len(historiesWithUser))
		return
	}

	// Check that user information is included
	for _, historyWithUser := range historiesWithUser {
		gt.NotEqual(t, historyWithUser.User, nil)
		if historyWithUser.ChangedBy == "U123456" {
			gt.Equal(t, historyWithUser.User.Name, "Test User")
		}
	}

	// Verify order (oldest first) and check last few entries
	lastIndex := len(historiesWithUser) - 1
	gt.Equal(t, historiesWithUser[lastIndex].Status, types.IncidentStatusClosed)
	gt.Equal(t, historiesWithUser[lastIndex-1].Status, types.IncidentStatusMonitoring)
}

func TestStatusUseCase_GetStatusHistory_UserNotFound(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	mockSlack := &mocks.SlackClientMock{}

	statusUC := usecase.NewStatusUseCase(repo, mockSlack)

	// Create a test incident
	incidentID := types.IncidentID(time.Now().UnixNano())
	incident, err := model.NewIncident(
		incidentID,
		"Test Incident",
		"Test Description",
		"test_category",
		types.ChannelID("C123456"),
		types.ChannelName("test-channel"),
		types.TeamID("T123456"),
		types.SlackUserID("U123456"),
		false,
	)
	gt.NoError(t, err)

	err = repo.PutIncident(ctx, incident)
	gt.NoError(t, err)

	// Create initial status history
	initialHistory, err := model.NewStatusHistory(incident.ID, incident.Status, incident.CreatedBy, "Incident created")
	gt.NoError(t, err)
	err = repo.AddStatusHistory(ctx, initialHistory)
	gt.NoError(t, err)

	// Get status history (user doesn't exist in repo)
	historiesWithUser, err := statusUC.GetStatusHistory(ctx, incidentID)
	gt.NoError(t, err)

	// Should have at least 1 entry (initial)
	if len(historiesWithUser) < 1 {
		t.Errorf("Expected at least 1 history entry, got %d", len(historiesWithUser))
		return
	}

	// Check that fallback user info is used
	historyWithUser := historiesWithUser[0]
	gt.NotEqual(t, historyWithUser.User, nil)
	gt.Equal(t, historyWithUser.User.Name, "U123456") // Fallback to slack ID
}

func TestStatusUseCase_InvalidInputs(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	mockSlack := &mocks.SlackClientMock{}

	statusUC := usecase.NewStatusUseCase(repo, mockSlack)

	// Test invalid incident ID
	err := statusUC.UpdateStatus(ctx, types.IncidentID(0), types.IncidentStatusHandling, "U123456", "test")
	gt.Error(t, err)

	// Test invalid status
	err = statusUC.UpdateStatus(ctx, types.IncidentID(123), types.IncidentStatus("invalid"), "U123456", "test")
	gt.Error(t, err)

	// Test empty user ID
	err = statusUC.UpdateStatus(ctx, types.IncidentID(123), types.IncidentStatusHandling, "", "test")
	gt.Error(t, err)

	// Test non-existent incident
	err = statusUC.UpdateStatus(ctx, types.IncidentID(999), types.IncidentStatusHandling, "U123456", "test")
	gt.Error(t, err)
}
