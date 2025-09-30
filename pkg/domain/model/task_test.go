package model_test

import (
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

func TestNewTask(t *testing.T) {
	t.Run("creates valid task", func(t *testing.T) {
		incidentID := types.IncidentID(1)
		title := "Test task"
		createdBy := types.SlackUserID("U123456")

		task, err := model.NewTask(incidentID, title, createdBy)
		gt.NoError(t, err)
		gt.V(t, task).NotNil()
		gt.Equal(t, task.IncidentID, incidentID)
		gt.Equal(t, task.Title, title)
		gt.Equal(t, task.CreatedBy, createdBy)
		gt.Equal(t, task.Status, model.TaskStatusTodo)
		gt.V(t, task.ID).NotEqual(types.TaskID(""))
		gt.V(t, task.CompletedAt).Nil()
	})

	t.Run("fails with invalid incident ID", func(t *testing.T) {
		_, err := model.NewTask(0, "Test task", "U123456")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("incident ID must be positive")
	})

	t.Run("fails with empty title", func(t *testing.T) {
		_, err := model.NewTask(1, "", "U123456")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("task title is required")
	})

	t.Run("fails with empty creator", func(t *testing.T) {
		_, err := model.NewTask(1, "Test task", "")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("creator user ID is required")
	})
}

func TestTaskComplete(t *testing.T) {
	t.Run("completes incomplete task", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		err = task.Complete()
		gt.NoError(t, err)
		gt.Equal(t, task.Status, model.TaskStatusCompleted)
		gt.V(t, task.CompletedAt).NotNil()
		gt.B(t, task.UpdatedAt.After(task.CreatedAt)).True()
	})

	t.Run("fails to complete already completed task", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		err = task.Complete()
		gt.NoError(t, err)

		err = task.Complete()
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("task is already completed")
	})
}

func TestTaskAssign(t *testing.T) {
	task, err := model.NewTask(1, "Test task", "U123456")
	gt.NoError(t, err)

	originalUpdatedAt := task.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	assignee := types.SlackUserID("U789012")
	task.Assign(assignee)

	gt.Equal(t, task.AssigneeID, assignee)
	gt.B(t, task.UpdatedAt.After(originalUpdatedAt)).True()
}

func TestTaskUpdateTitle(t *testing.T) {
	t.Run("updates title successfully", func(t *testing.T) {
		task, err := model.NewTask(1, "Original title", "U123456")
		gt.NoError(t, err)

		newTitle := "Updated title"
		err = task.UpdateTitle(newTitle)
		gt.NoError(t, err)
		gt.Equal(t, task.Title, newTitle)
	})

	t.Run("fails with empty title", func(t *testing.T) {
		task, err := model.NewTask(1, "Original title", "U123456")
		gt.NoError(t, err)

		err = task.UpdateTitle("")
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("task title cannot be empty")
		gt.Equal(t, task.Title, "Original title") // Title should not change
	})
}

func TestTaskUpdateDescription(t *testing.T) {
	task, err := model.NewTask(1, "Test task", "U123456")
	gt.NoError(t, err)

	originalUpdatedAt := task.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	newDescription := "This is a detailed description"
	task.UpdateDescription(newDescription)

	gt.Equal(t, task.Description, newDescription)
	gt.B(t, task.UpdatedAt.After(originalUpdatedAt)).True()
}

func TestTaskUpdateStatus(t *testing.T) {
	t.Run("updates to completed status", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		err = task.UpdateStatus(model.TaskStatusCompleted)
		gt.NoError(t, err)
		gt.Equal(t, task.Status, model.TaskStatusCompleted)
		gt.V(t, task.CompletedAt).NotNil()
	})

	t.Run("updates from completed to incompleted", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		err = task.UpdateStatus(model.TaskStatusCompleted)
		gt.NoError(t, err)
		gt.V(t, task.CompletedAt).NotNil()

		err = task.UpdateStatus(model.TaskStatusTodo)
		gt.NoError(t, err)
		gt.Equal(t, task.Status, model.TaskStatusTodo)
		gt.V(t, task.CompletedAt).Nil()
	})

	t.Run("fails with invalid status", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		err = task.UpdateStatus(model.TaskStatus("invalid"))
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("invalid task status")
	})
}

func TestTaskSetMessageTS(t *testing.T) {
	task, err := model.NewTask(1, "Test task", "U123456")
	gt.NoError(t, err)

	messageTS := "1234567890.123456"
	task.SetMessageTS(messageTS)

	gt.Equal(t, task.MessageTS, messageTS)
}

func TestTaskIsCompleted(t *testing.T) {
	task, err := model.NewTask(1, "Test task", "U123456")
	gt.NoError(t, err)

	gt.B(t, task.IsCompleted()).False()

	err = task.Complete()
	gt.NoError(t, err)

	gt.B(t, task.IsCompleted()).True()
}

func TestTaskGetSlackMessageURL(t *testing.T) {
	t.Run("generates valid URL", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		task.SetMessageTS("1234567890.123456")
		channelID := types.ChannelID("C123456")

		url := task.GetSlackMessageURL(channelID)
		gt.Equal(t, url, "https://slack.com/archives/C123456/p1234567890123456")
	})

	t.Run("returns empty string with no message TS", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		channelID := types.ChannelID("C123456")
		url := task.GetSlackMessageURL(channelID)
		gt.Equal(t, url, "")
	})

	t.Run("returns empty string with no channel ID", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		task.SetMessageTS("1234567890.123456")
		url := task.GetSlackMessageURL("")
		gt.Equal(t, url, "")
	})
}

func TestTaskStatusIsValid(t *testing.T) {
	t.Run("valid statuses", func(t *testing.T) {
		gt.B(t, model.TaskStatusTodo.IsValid()).True()
		gt.B(t, model.TaskStatusFollowUp.IsValid()).True()
		gt.B(t, model.TaskStatusCompleted.IsValid()).True()
	})

	t.Run("invalid status", func(t *testing.T) {
		invalidStatus := model.TaskStatus("invalid")
		gt.B(t, invalidStatus.IsValid()).False()
	})
}

func TestTaskStatusTransitions(t *testing.T) {
	t.Run("todo to follow_up transition", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)
		gt.Equal(t, task.Status, model.TaskStatusTodo)
		gt.V(t, task.CompletedAt).Nil()

		err = task.UpdateStatus(model.TaskStatusFollowUp)
		gt.NoError(t, err)
		gt.Equal(t, task.Status, model.TaskStatusFollowUp)
		gt.V(t, task.CompletedAt).Nil()
	})

	t.Run("follow_up to completed transition", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		err = task.UpdateStatus(model.TaskStatusFollowUp)
		gt.NoError(t, err)

		beforeUpdate := task.UpdatedAt
		err = task.UpdateStatus(model.TaskStatusCompleted)
		gt.NoError(t, err)
		gt.Equal(t, task.Status, model.TaskStatusCompleted)
		gt.V(t, task.CompletedAt).NotNil()
		gt.V(t, task.UpdatedAt).NotEqual(beforeUpdate)
	})

	t.Run("completed to follow_up transition", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		err = task.UpdateStatus(model.TaskStatusCompleted)
		gt.NoError(t, err)
		gt.V(t, task.CompletedAt).NotNil()

		err = task.UpdateStatus(model.TaskStatusFollowUp)
		gt.NoError(t, err)
		gt.Equal(t, task.Status, model.TaskStatusFollowUp)
		gt.V(t, task.CompletedAt).Nil()
	})

	t.Run("todo to completed transition", func(t *testing.T) {
		task, err := model.NewTask(1, "Test task", "U123456")
		gt.NoError(t, err)

		err = task.UpdateStatus(model.TaskStatusCompleted)
		gt.NoError(t, err)
		gt.Equal(t, task.Status, model.TaskStatusCompleted)
		gt.V(t, task.CompletedAt).NotNil()
	})
}
