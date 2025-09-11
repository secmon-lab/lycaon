package usecase

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// TaskUseCase implements the Task interface
type TaskUseCase struct {
	repo      interfaces.Repository
	slackRepo interfaces.SlackClient
}

// NewTaskUseCase creates a new TaskUseCase instance
func NewTaskUseCase(repo interfaces.Repository, slackRepo interfaces.SlackClient) interfaces.Task {
	return &TaskUseCase{
		repo:      repo,
		slackRepo: slackRepo,
	}
}

// CreateTask creates a new task for an incident
func (u *TaskUseCase) CreateTask(ctx context.Context, incidentID types.IncidentID, title string, userID types.SlackUserID, channelID types.ChannelID, messageTS string) (*model.Task, error) {
	// Validate incident exists
	_, err := u.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident",
			goerr.V("incidentID", incidentID))
	}

	// Create new task
	task, err := model.NewTask(incidentID, title, userID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create task")
	}

	// Set channel ID and message timestamp for link generation
	task.SetChannelID(channelID)
	if messageTS != "" {
		task.SetMessageTS(messageTS)
	}

	// Save to repository
	if err := u.repo.CreateTask(ctx, task); err != nil {
		return nil, goerr.Wrap(err, "failed to save task",
			goerr.V("taskID", task.ID),
			goerr.V("incidentID", incidentID))
	}

	return task, nil
}

// ListTasks retrieves all tasks for an incident
func (u *TaskUseCase) ListTasks(ctx context.Context, incidentID types.IncidentID) ([]*model.Task, error) {
	// Validate incident exists
	_, err := u.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident",
			goerr.V("incidentID", incidentID))
	}

	tasks, err := u.repo.ListTasksByIncident(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to list tasks",
			goerr.V("incidentID", incidentID))
	}

	return tasks, nil
}

// UpdateTask updates an existing task
func (u *TaskUseCase) UpdateTask(ctx context.Context, taskID types.TaskID, updates interfaces.TaskUpdateRequest) (*model.Task, error) {
	// Get existing task
	task, err := u.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get task",
			goerr.V("taskID", taskID))
	}

	// Apply updates
	if updates.Title != nil {
		if err := task.UpdateTitle(*updates.Title); err != nil {
			return nil, goerr.Wrap(err, "failed to update title",
				goerr.V("taskID", taskID))
		}
	}

	if updates.Description != nil {
		task.UpdateDescription(*updates.Description)
	}

	if updates.Status != nil {
		if err := task.UpdateStatus(*updates.Status); err != nil {
			return nil, goerr.Wrap(err, "failed to update status",
				goerr.V("taskID", taskID),
				goerr.V("status", *updates.Status))
		}
	}

	if updates.AssigneeID != nil {
		task.Assign(*updates.AssigneeID)
	}

	if updates.MessageTS != nil {
		task.SetMessageTS(*updates.MessageTS)
	}

	if updates.ChannelID != nil {
		task.SetChannelID(*updates.ChannelID)
	}

	// Save updated task
	if err := u.repo.UpdateTask(ctx, task); err != nil {
		return nil, goerr.Wrap(err, "failed to save updated task",
			goerr.V("taskID", taskID))
	}

	return task, nil
}

// UpdateTaskByIncident updates an existing task efficiently using incident ID
func (u *TaskUseCase) UpdateTaskByIncident(ctx context.Context, incidentID types.IncidentID, taskID types.TaskID, updates interfaces.TaskUpdateRequest) (*model.Task, error) {
	// Get existing task efficiently
	task, err := u.repo.GetTaskByIncident(ctx, incidentID, taskID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get task by incident",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	// Apply updates
	if updates.Title != nil {
		if err := task.UpdateTitle(*updates.Title); err != nil {
			return nil, goerr.Wrap(err, "failed to update title",
				goerr.V("incidentID", incidentID),
				goerr.V("taskID", taskID))
		}
	}

	if updates.Description != nil {
		task.UpdateDescription(*updates.Description)
	}

	if updates.Status != nil {
		if err := task.UpdateStatus(*updates.Status); err != nil {
			return nil, goerr.Wrap(err, "failed to update status",
				goerr.V("incidentID", incidentID),
				goerr.V("taskID", taskID),
				goerr.V("status", *updates.Status))
		}
	}

	if updates.AssigneeID != nil {
		task.Assign(*updates.AssigneeID)
	}

	if updates.MessageTS != nil {
		task.SetMessageTS(*updates.MessageTS)
	}

	if updates.ChannelID != nil {
		task.SetChannelID(*updates.ChannelID)
	}

	// Save updated task
	if err := u.repo.UpdateTask(ctx, task); err != nil {
		return nil, goerr.Wrap(err, "failed to save updated task",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	return task, nil
}

// CompleteTask marks a task as completed
func (u *TaskUseCase) CompleteTask(ctx context.Context, taskID types.TaskID) (*model.Task, error) {
	// Get existing task
	task, err := u.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get task",
			goerr.V("taskID", taskID))
	}

	// Mark as completed
	if err := task.Complete(); err != nil {
		return nil, goerr.Wrap(err, "failed to complete task",
			goerr.V("taskID", taskID))
	}

	// Save updated task
	if err := u.repo.UpdateTask(ctx, task); err != nil {
		return nil, goerr.Wrap(err, "failed to save completed task",
			goerr.V("taskID", taskID))
	}

	return task, nil
}

// CompleteTaskByIncident marks a task as completed efficiently using incident ID
func (u *TaskUseCase) CompleteTaskByIncident(ctx context.Context, incidentID types.IncidentID, taskID types.TaskID) (*model.Task, error) {
	// Get existing task efficiently
	task, err := u.repo.GetTaskByIncident(ctx, incidentID, taskID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get task by incident",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	// Mark as completed
	if err := task.Complete(); err != nil {
		return nil, goerr.Wrap(err, "failed to complete task",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	// Save updated task
	if err := u.repo.UpdateTask(ctx, task); err != nil {
		return nil, goerr.Wrap(err, "failed to save completed task",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	return task, nil
}

// UncompleteTask marks a task as incomplete
func (u *TaskUseCase) UncompleteTask(ctx context.Context, taskID types.TaskID) (*model.Task, error) {
	// Get existing task
	task, err := u.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get task",
			goerr.V("taskID", taskID))
	}

	// Mark as incomplete
	if err := task.Uncomplete(); err != nil {
		return nil, goerr.Wrap(err, "failed to uncomplete task",
			goerr.V("taskID", taskID))
	}

	// Save updated task
	if err := u.repo.UpdateTask(ctx, task); err != nil {
		return nil, goerr.Wrap(err, "failed to save uncompleted task",
			goerr.V("taskID", taskID))
	}

	return task, nil
}

// UncompleteTaskByIncident marks a task as incomplete efficiently using incident ID
func (u *TaskUseCase) UncompleteTaskByIncident(ctx context.Context, incidentID types.IncidentID, taskID types.TaskID) (*model.Task, error) {
	// Get existing task efficiently
	task, err := u.repo.GetTaskByIncident(ctx, incidentID, taskID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get task by incident",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	// Mark as incomplete
	if err := task.Uncomplete(); err != nil {
		return nil, goerr.Wrap(err, "failed to uncomplete task",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	// Save updated task
	if err := u.repo.UpdateTask(ctx, task); err != nil {
		return nil, goerr.Wrap(err, "failed to save uncompleted task",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	return task, nil
}

// GetTask retrieves a task by ID
func (u *TaskUseCase) GetTask(ctx context.Context, taskID types.TaskID) (*model.Task, error) {
	task, err := u.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get task",
			goerr.V("taskID", taskID))
	}

	return task, nil
}

// GetTaskByIncident retrieves a task by incident ID and task ID efficiently
func (u *TaskUseCase) GetTaskByIncident(ctx context.Context, incidentID types.IncidentID, taskID types.TaskID) (*model.Task, error) {
	task, err := u.repo.GetTaskByIncident(ctx, incidentID, taskID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get task by incident",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	return task, nil
}
