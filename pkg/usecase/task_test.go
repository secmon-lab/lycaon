package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/usecase"
)

func TestTaskUseCase_CreateTask(t *testing.T) {
	ctx := context.Background()

	t.Run("creates task successfully", func(t *testing.T) {
		// Setup mocks
		repo := &mocks.RepositoryMock{
			GetIncidentFunc: func(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
				return &model.Incident{ID: id}, nil
			},
			CreateTaskFunc: func(ctx context.Context, task *model.Task) error {
				return nil
			},
		}
		slackRepo := &mocks.SlackClientMock{}

		// Create use case
		uc := usecase.NewTaskUseCase(repo, slackRepo)

		// Execute
		incidentID := types.IncidentID(1)
		task, err := uc.CreateTask(ctx, incidentID, "Test task", "U123456", "C123456", "1234567890.123456")

		// Verify
		gt.NoError(t, err)
		gt.V(t, task).NotNil()
		gt.Equal(t, task.Title, "Test task")
		gt.Equal(t, task.IncidentID, incidentID)
		gt.Equal(t, task.CreatedBy, types.SlackUserID("U123456"))
		gt.Equal(t, task.MessageTS, "1234567890.123456")
		gt.Equal(t, task.Status, model.TaskStatusIncompleted)

		// Verify mock calls
		gt.Equal(t, len(repo.GetIncidentCalls()), 1)
		gt.Equal(t, len(repo.CreateTaskCalls()), 1)
	})

	t.Run("fails when incident not found", func(t *testing.T) {
		// Setup mocks
		repo := &mocks.RepositoryMock{
			GetIncidentFunc: func(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
				return nil, model.ErrIncidentNotFound
			},
		}
		slackRepo := &mocks.SlackClientMock{}

		// Create use case
		uc := usecase.NewTaskUseCase(repo, slackRepo)

		// Execute
		_, err := uc.CreateTask(ctx, types.IncidentID(999), "Test task", "U123456", "C123456", "")

		// Verify
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to get incident")
	})
}

func TestTaskUseCase_ListTasks(t *testing.T) {
	ctx := context.Background()

	t.Run("lists tasks successfully", func(t *testing.T) {
		// Setup test data
		incidentID := types.IncidentID(1)
		tasks := []*model.Task{
			{ID: types.NewTaskID(), Title: "Task 1", IncidentID: incidentID},
			{ID: types.NewTaskID(), Title: "Task 2", IncidentID: incidentID},
		}

		// Setup mocks
		repo := &mocks.RepositoryMock{
			GetIncidentFunc: func(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
				return &model.Incident{ID: id}, nil
			},
			ListTasksByIncidentFunc: func(ctx context.Context, id types.IncidentID) ([]*model.Task, error) {
				return tasks, nil
			},
		}
		slackRepo := &mocks.SlackClientMock{}

		// Create use case
		uc := usecase.NewTaskUseCase(repo, slackRepo)

		// Execute
		result, err := uc.ListTasks(ctx, incidentID)

		// Verify
		gt.NoError(t, err)
		gt.Equal(t, len(result), 2)
		gt.Equal(t, result[0].Title, "Task 1")
		gt.Equal(t, result[1].Title, "Task 2")
	})
}

func TestTaskUseCase_UpdateTask(t *testing.T) {
	ctx := context.Background()

	t.Run("updates task successfully", func(t *testing.T) {
		// Setup test data
		taskID := types.NewTaskID()
		existingTask, _ := model.NewTask(1, "Original title", "U123456")
		existingTask.ID = taskID

		// Setup mocks
		repo := &mocks.RepositoryMock{
			GetTaskFunc: func(ctx context.Context, id types.TaskID) (*model.Task, error) {
				// Return a copy to simulate repository behavior
				taskCopy := *existingTask
				return &taskCopy, nil
			},
			UpdateTaskFunc: func(ctx context.Context, task *model.Task) error {
				return nil
			},
		}
		slackRepo := &mocks.SlackClientMock{}

		// Create use case
		uc := usecase.NewTaskUseCase(repo, slackRepo)

		// Execute
		newTitle := "Updated title"
		newDescription := "New description"
		newAssignee := types.SlackUserID("U999999")
		newStatus := model.TaskStatusCompleted

		updates := interfaces.TaskUpdateRequest{
			Title:       &newTitle,
			Description: &newDescription,
			AssigneeID:  &newAssignee,
			Status:      &newStatus,
		}

		result, err := uc.UpdateTask(ctx, taskID, updates)

		// Verify
		gt.NoError(t, err)
		gt.V(t, result).NotNil()
		gt.Equal(t, result.Title, newTitle)
		gt.Equal(t, result.Description, newDescription)
		gt.Equal(t, result.AssigneeID, newAssignee)
		gt.Equal(t, result.Status, newStatus)
		gt.V(t, result.CompletedAt).NotNil()

		// Verify mock calls
		gt.Equal(t, len(repo.GetTaskCalls()), 1)
		gt.Equal(t, len(repo.UpdateTaskCalls()), 1)
	})

	t.Run("fails when task not found", func(t *testing.T) {
		// Setup mocks
		repo := &mocks.RepositoryMock{
			GetTaskFunc: func(ctx context.Context, id types.TaskID) (*model.Task, error) {
				return nil, model.ErrTaskNotFound
			},
		}
		slackRepo := &mocks.SlackClientMock{}

		// Create use case
		uc := usecase.NewTaskUseCase(repo, slackRepo)

		// Execute
		newTitle := "Updated title"
		updates := interfaces.TaskUpdateRequest{
			Title: &newTitle,
		}

		_, err := uc.UpdateTask(ctx, types.NewTaskID(), updates)

		// Verify
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to get task")
	})
}

func TestTaskUseCase_CompleteTask(t *testing.T) {
	ctx := context.Background()

	t.Run("completes task successfully", func(t *testing.T) {
		// Setup test data
		taskID := types.NewTaskID()
		existingTask, _ := model.NewTask(1, "Test task", "U123456")
		existingTask.ID = taskID

		// Setup mocks
		repo := &mocks.RepositoryMock{
			GetTaskFunc: func(ctx context.Context, id types.TaskID) (*model.Task, error) {
				// Return a copy to simulate repository behavior
				taskCopy := *existingTask
				return &taskCopy, nil
			},
			UpdateTaskFunc: func(ctx context.Context, task *model.Task) error {
				return nil
			},
		}
		slackRepo := &mocks.SlackClientMock{}

		// Create use case
		uc := usecase.NewTaskUseCase(repo, slackRepo)

		// Execute
		result, err := uc.CompleteTask(ctx, taskID)

		// Verify
		gt.NoError(t, err)
		gt.V(t, result).NotNil()
		gt.Equal(t, result.Status, model.TaskStatusCompleted)
		gt.V(t, result.CompletedAt).NotNil()

		// Verify mock calls
		gt.Equal(t, len(repo.GetTaskCalls()), 1)
		gt.Equal(t, len(repo.UpdateTaskCalls()), 1)
	})

	t.Run("fails when task already completed", func(t *testing.T) {
		// Setup test data with already completed task
		taskID := types.NewTaskID()
		existingTask, _ := model.NewTask(1, "Test task", "U123456")
		existingTask.ID = taskID
		existingTask.Complete() // Already completed

		// Setup mocks
		repo := &mocks.RepositoryMock{
			GetTaskFunc: func(ctx context.Context, id types.TaskID) (*model.Task, error) {
				// Return a copy to simulate repository behavior
				taskCopy := *existingTask
				return &taskCopy, nil
			},
		}
		slackRepo := &mocks.SlackClientMock{}

		// Create use case
		uc := usecase.NewTaskUseCase(repo, slackRepo)

		// Execute
		_, err := uc.CompleteTask(ctx, taskID)

		// Verify
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("task is already completed")
	})
}

func TestTaskUseCase_GetTask(t *testing.T) {
	ctx := context.Background()

	t.Run("gets task successfully", func(t *testing.T) {
		// Setup test data
		taskID := types.NewTaskID()
		expectedTask := &model.Task{
			ID:         taskID,
			Title:      "Test task",
			IncidentID: 1,
			CreatedBy:  "U123456",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			Status:     model.TaskStatusIncompleted,
		}

		// Setup mocks
		repo := &mocks.RepositoryMock{
			GetTaskFunc: func(ctx context.Context, id types.TaskID) (*model.Task, error) {
				return expectedTask, nil
			},
		}
		slackRepo := &mocks.SlackClientMock{}

		// Create use case
		uc := usecase.NewTaskUseCase(repo, slackRepo)

		// Execute
		result, err := uc.GetTask(ctx, taskID)

		// Verify
		gt.NoError(t, err)
		gt.V(t, result).NotNil()
		gt.Equal(t, result.ID, taskID)
		gt.Equal(t, result.Title, "Test task")

		// Verify mock calls
		gt.Equal(t, len(repo.GetTaskCalls()), 1)
	})

	t.Run("fails when task not found", func(t *testing.T) {
		// Setup mocks
		repo := &mocks.RepositoryMock{
			GetTaskFunc: func(ctx context.Context, id types.TaskID) (*model.Task, error) {
				return nil, model.ErrTaskNotFound
			},
		}
		slackRepo := &mocks.SlackClientMock{}

		// Create use case
		uc := usecase.NewTaskUseCase(repo, slackRepo)

		// Execute
		_, err := uc.GetTask(ctx, types.NewTaskID())

		// Verify
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to get task")
	})
}
