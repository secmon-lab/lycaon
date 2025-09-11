package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
)

func TestTaskRepository(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()

	// Use random IDs to avoid conflicts in parallel tests
	timestamp := time.Now().UnixNano()
	incidentID := types.IncidentID(timestamp % 1000000)

	t.Run("CreateTask", func(t *testing.T) {
		task, err := model.NewTask(incidentID, "Test task", "U123456")
		gt.NoError(t, err)
		task.SetMessageTS("1234567890.123456")

		err = repo.CreateTask(ctx, task)
		gt.NoError(t, err)

		// Verify task was created
		retrieved, err := repo.GetTask(ctx, task.ID)
		gt.NoError(t, err)
		gt.Equal(t, retrieved.ID, task.ID)
		gt.Equal(t, retrieved.Title, task.Title)
		gt.Equal(t, retrieved.IncidentID, task.IncidentID)
		gt.Equal(t, retrieved.CreatedBy, task.CreatedBy)
		gt.Equal(t, retrieved.MessageTS, task.MessageTS)
		gt.Equal(t, retrieved.Status, model.TaskStatusIncompleted)
	})

	t.Run("GetTask", func(t *testing.T) {
		task, err := model.NewTask(incidentID, "Another task", "U789012")
		gt.NoError(t, err)

		err = repo.CreateTask(ctx, task)
		gt.NoError(t, err)

		retrieved, err := repo.GetTask(ctx, task.ID)
		gt.NoError(t, err)
		gt.Equal(t, retrieved.ID, task.ID)
		gt.Equal(t, retrieved.Title, task.Title)
	})

	t.Run("GetTask_NotFound", func(t *testing.T) {
		nonExistentID := types.TaskID(fmt.Sprintf("task-%d", time.Now().UnixNano()))
		_, err := repo.GetTask(ctx, nonExistentID)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("task not found")
	})

	t.Run("UpdateTask", func(t *testing.T) {
		task, err := model.NewTask(incidentID, "Task to update", "U345678")
		gt.NoError(t, err)

		err = repo.CreateTask(ctx, task)
		gt.NoError(t, err)

		// Update task
		task.UpdateTitle("Updated title")
		task.UpdateDescription("New description")
		task.Assign("U999999")
		task.Complete()

		err = repo.UpdateTask(ctx, task)
		gt.NoError(t, err)

		// Verify updates
		retrieved, err := repo.GetTask(ctx, task.ID)
		gt.NoError(t, err)
		gt.Equal(t, retrieved.Title, "Updated title")
		gt.Equal(t, retrieved.Description, "New description")
		gt.Equal(t, retrieved.AssigneeID, types.SlackUserID("U999999"))
		gt.Equal(t, retrieved.Status, model.TaskStatusCompleted)
		gt.V(t, retrieved.CompletedAt).NotNil()
	})

	t.Run("UpdateTask_NotFound", func(t *testing.T) {
		task, err := model.NewTask(incidentID, "Non-existent task", "U111111")
		gt.NoError(t, err)

		err = repo.UpdateTask(ctx, task)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("task not found")
	})

	t.Run("ListTasksByIncident", func(t *testing.T) {
		// Create a new incident ID for this test
		testIncidentID := types.IncidentID(time.Now().UnixNano() % 1000000)

		// Create multiple tasks
		task1, err := model.NewTask(testIncidentID, "Task 1", "U111111")
		gt.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps

		task2, err := model.NewTask(testIncidentID, "Task 2", "U222222")
		gt.NoError(t, err)
		time.Sleep(1 * time.Millisecond)

		task3, err := model.NewTask(testIncidentID, "Task 3", "U333333")
		gt.NoError(t, err)

		// Create tasks in random order
		gt.NoError(t, repo.CreateTask(ctx, task2))
		gt.NoError(t, repo.CreateTask(ctx, task1))
		gt.NoError(t, repo.CreateTask(ctx, task3))

		// List tasks
		tasks, err := repo.ListTasksByIncident(ctx, testIncidentID)
		gt.NoError(t, err)
		gt.Equal(t, len(tasks), 3)

		// Verify order (should be sorted by creation time)
		gt.Equal(t, tasks[0].ID, task1.ID)
		gt.Equal(t, tasks[1].ID, task2.ID)
		gt.Equal(t, tasks[2].ID, task3.ID)
	})

	t.Run("ListTasksByIncident_Empty", func(t *testing.T) {
		emptyIncidentID := types.IncidentID(time.Now().UnixNano() % 1000000)
		tasks, err := repo.ListTasksByIncident(ctx, emptyIncidentID)
		gt.NoError(t, err)
		gt.Equal(t, len(tasks), 0)
	})

	t.Run("ListTasksByIncident_MultipleIncidents", func(t *testing.T) {
		incident1ID := types.IncidentID(time.Now().UnixNano() % 1000000)
		incident2ID := types.IncidentID((time.Now().UnixNano() % 1000000) + 1)

		// Create tasks for different incidents
		task1, err := model.NewTask(incident1ID, "Inc1 Task", "U444444")
		gt.NoError(t, err)
		gt.NoError(t, repo.CreateTask(ctx, task1))

		task2, err := model.NewTask(incident2ID, "Inc2 Task", "U555555")
		gt.NoError(t, err)
		gt.NoError(t, repo.CreateTask(ctx, task2))

		// List tasks for incident 1
		tasks1, err := repo.ListTasksByIncident(ctx, incident1ID)
		gt.NoError(t, err)
		gt.Equal(t, len(tasks1), 1)
		gt.Equal(t, tasks1[0].ID, task1.ID)

		// List tasks for incident 2
		tasks2, err := repo.ListTasksByIncident(ctx, incident2ID)
		gt.NoError(t, err)
		gt.Equal(t, len(tasks2), 1)
		gt.Equal(t, tasks2[0].ID, task2.ID)
	})
}
