package model

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// TaskStatus represents the status of a task
type TaskStatus string

// Ensure TaskStatus implements graphql.Marshaler and graphql.Unmarshaler
var _ graphql.Marshaler = TaskStatus("")
var _ graphql.Unmarshaler = (*TaskStatus)(nil)

const (
	// TaskStatusTodo represents a task that needs to be done
	TaskStatusTodo TaskStatus = "todo"
	// TaskStatusFollowUp represents a task that needs follow-up
	TaskStatusFollowUp TaskStatus = "follow-up"
	// TaskStatusCompleted represents a completed task
	TaskStatusCompleted TaskStatus = "completed"
)

// IsValid checks if the task status is valid
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusTodo, TaskStatusFollowUp, TaskStatusCompleted:
		return true
	default:
		return false
	}
}

// MarshalGQL implements the graphql.Marshaler interface for GraphQL enum serialization
func (s TaskStatus) MarshalGQL(w io.Writer) {
	switch s {
	case TaskStatusTodo:
		_, _ = io.WriteString(w, strconv.Quote("TODO"))
	case TaskStatusFollowUp:
		_, _ = io.WriteString(w, strconv.Quote("FOLLOW_UP"))
	case TaskStatusCompleted:
		_, _ = io.WriteString(w, strconv.Quote("COMPLETED"))
	default:
		_, _ = io.WriteString(w, strconv.Quote("TODO")) // Default fallback
	}
}

// UnmarshalGQL implements the graphql.Unmarshaler interface for GraphQL enum deserialization
func (s *TaskStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("TaskStatus must be a string, got %T", v)
	}

	switch str {
	case "TODO":
		*s = TaskStatusTodo
	case "FOLLOW_UP":
		*s = TaskStatusFollowUp
	case "COMPLETED":
		*s = TaskStatusCompleted
	default:
		return fmt.Errorf("invalid TaskStatus: %s", str)
	}

	return nil
}

// Task represents a task in an incident
type Task struct {
	ID          types.TaskID      // Unique identifier
	IncidentID  types.IncidentID  // Parent incident ID
	Title       string            // Task title
	Description string            // Task description (optional)
	Status      TaskStatus        // Task status
	AssigneeID  types.SlackUserID // Assignee Slack ID (optional)
	CreatedBy   types.SlackUserID // Creator Slack ID
	ChannelID   types.ChannelID   // Channel where task message was posted
	MessageTS   string            // Slack message timestamp for link generation
	CreatedAt   time.Time         // Creation timestamp
	UpdatedAt   time.Time         // Update timestamp
	CompletedAt *time.Time        // Completion timestamp (optional)
}

// NewTask creates a new Task instance
func NewTask(incidentID types.IncidentID, title string, createdBy types.SlackUserID) (*Task, error) {
	if incidentID <= 0 {
		return nil, goerr.New("incident ID must be positive", goerr.V("incidentID", incidentID))
	}
	if title == "" {
		return nil, goerr.New("task title is required")
	}
	if createdBy == "" {
		return nil, goerr.New("creator user ID is required")
	}

	now := time.Now()
	return &Task{
		ID:          types.NewTaskID(),
		IncidentID:  incidentID,
		Title:       title,
		Description: "",
		Status:      TaskStatusTodo,
		AssigneeID:  "",
		CreatedBy:   createdBy,
		MessageTS:   "",
		CreatedAt:   now,
		UpdatedAt:   now,
		CompletedAt: nil,
	}, nil
}

// Complete marks the task as completed
func (t *Task) Complete() error {
	if t.Status == TaskStatusCompleted {
		return goerr.New("task is already completed", goerr.V("taskID", t.ID))
	}

	now := time.Now()
	t.Status = TaskStatusCompleted
	t.CompletedAt = &now
	t.UpdatedAt = now
	return nil
}

// Uncomplete marks the task as todo (incomplete)
func (t *Task) Uncomplete() error {
	if t.Status == TaskStatusTodo {
		return goerr.New("task is already todo", goerr.V("taskID", t.ID))
	}

	t.Status = TaskStatusTodo
	t.CompletedAt = nil
	t.UpdatedAt = time.Now()
	return nil
}

// Assign assigns the task to a user
func (t *Task) Assign(userID types.SlackUserID) {
	t.AssigneeID = userID
	t.UpdatedAt = time.Now()
}

// UpdateTitle updates the task title
func (t *Task) UpdateTitle(title string) error {
	if title == "" {
		return goerr.New("task title cannot be empty")
	}
	t.Title = title
	t.UpdatedAt = time.Now()
	return nil
}

// UpdateDescription updates the task description
func (t *Task) UpdateDescription(description string) {
	t.Description = description
	t.UpdatedAt = time.Now()
}

// UpdateStatus updates the task status
func (t *Task) UpdateStatus(status TaskStatus) error {
	if !status.IsValid() {
		return goerr.New("invalid task status", goerr.V("status", status))
	}

	t.Status = status
	t.UpdatedAt = time.Now()

	// Update completion timestamp based on status
	if status == TaskStatusCompleted {
		now := time.Now()
		t.CompletedAt = &now
	} else {
		t.CompletedAt = nil
	}

	return nil
}

// SetMessageTS sets the Slack message timestamp
func (t *Task) SetMessageTS(messageTS string) {
	t.MessageTS = messageTS
	t.UpdatedAt = time.Now()
}

// SetChannelID sets the channel ID where the task message was posted
func (t *Task) SetChannelID(channelID types.ChannelID) {
	t.ChannelID = channelID
	t.UpdatedAt = time.Now()
}

// IsCompleted returns true if the task is completed
func (t *Task) IsCompleted() bool {
	return t.Status == TaskStatusCompleted
}

// GetSlackMessageURL generates a Slack message URL
func (t *Task) GetSlackMessageURL(channelID types.ChannelID) string {
	if t.MessageTS == "" || channelID == "" {
		return ""
	}

	// Convert message timestamp to permalink format
	// Remove the dot from the timestamp for the URL
	// e.g., "1234567890.123456" becomes "1234567890123456"
	formattedTS := strings.Replace(t.MessageTS, ".", "", 1)

	return "https://slack.com/archives/" + string(channelID) + "/p" + formattedTS
}
