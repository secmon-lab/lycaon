package slack_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	slackblocks "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/slack-go/slack"
)

func TestBuildTaskEditModal_CallbackIDFormat(t *testing.T) {
	// Create test task
	task := &model.Task{
		ID:          types.TaskID("test-task-12345"),
		IncidentID:  types.IncidentID(42),
		Title:       "Test Task",
		Description: "Test Description",
		Status:      model.TaskStatusTodo,
		AssigneeID:  types.SlackUserID("U123456789"),
		CreatedBy:   types.SlackUserID("U987654321"),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Build modal
	modal := slackblocks.BuildTaskEditModal(task, []types.SlackUserID{
		"U123456789",
	})

	// Verify callback ID format
	expectedCallbackID := fmt.Sprintf("task_edit_submit:%d:%s", task.IncidentID, task.ID)
	gt.Equal(t, expectedCallbackID, modal.CallbackID)
	gt.Equal(t, "task_edit_submit:42:test-task-12345", modal.CallbackID)

	// Verify other modal properties
	gt.Equal(t, slack.ViewType("modal"), modal.Type)
	gt.Equal(t, "Edit Task", modal.Title.Text)
	gt.Equal(t, "Save", modal.Submit.Text)
	gt.Equal(t, "Cancel", modal.Close.Text)
	gt.A(t, modal.Blocks.BlockSet).Longer(0)
}

func TestBuildTaskEditModal_DifferentIncidentIDs(t *testing.T) {
	testCases := []struct {
		name       string
		incidentID types.IncidentID
		taskID     types.TaskID
		expected   string
	}{
		{
			name:       "Small incident ID",
			incidentID: 1,
			taskID:     "task123",
			expected:   "task_edit_submit:1:task123",
		},
		{
			name:       "Large incident ID",
			incidentID: 999999,
			taskID:     "task-very-long-id-12345",
			expected:   "task_edit_submit:999999:task-very-long-id-12345",
		},
		{
			name:       "Zero incident ID",
			incidentID: 0,
			taskID:     "task0",
			expected:   "task_edit_submit:0:task0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task := &model.Task{
				ID:         tc.taskID,
				IncidentID: tc.incidentID,
				Title:      "Test Task",
				Status:     model.TaskStatusTodo,
				CreatedBy:  types.SlackUserID("U123456789"),
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			modal := slackblocks.BuildTaskEditModal(task, []types.SlackUserID{})
			gt.Equal(t, tc.expected, modal.CallbackID)
		})
	}
}
