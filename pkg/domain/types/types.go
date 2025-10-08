package types

import (
	"fmt"

	"github.com/google/uuid"
)

// UserID represents a user identifier
type UserID string

// String returns the string representation
func (id UserID) String() string {
	return string(id)
}

// NewUserID creates a new UserID
func NewUserID() UserID {
	return UserID(uuid.New().String())
}

// SlackUserID represents a Slack user identifier
type SlackUserID string

// String returns the string representation
func (id SlackUserID) String() string {
	return string(id)
}

// ChannelID represents a Slack channel identifier
type ChannelID string

// String returns the string representation
func (id ChannelID) String() string {
	return string(id)
}

// ChannelName represents a Slack channel name
type ChannelName string

// String returns the string representation
func (n ChannelName) String() string {
	return string(n)
}

// TeamID represents a Slack workspace/team ID
type TeamID string

// String returns the string representation
func (id TeamID) String() string {
	return string(id)
}

// MessageID represents a message identifier
type MessageID string

// String returns the string representation
func (id MessageID) String() string {
	return string(id)
}

// NewMessageID creates a new MessageID
func NewMessageID() MessageID {
	return MessageID(fmt.Sprintf("msg-%s", uuid.New().String()))
}

// MessageTS represents a Slack message timestamp
type MessageTS string

// String returns the string representation
func (ts MessageTS) String() string {
	return string(ts)
}

// ThreadTS represents a Slack thread timestamp
type ThreadTS string

// String returns the string representation
func (ts ThreadTS) String() string {
	return string(ts)
}

// EventTS represents a Slack event timestamp
type EventTS string

// String returns the string representation
func (ts EventTS) String() string {
	return string(ts)
}

// SessionID represents a session identifier
type SessionID string

// String returns the string representation
func (id SessionID) String() string {
	return string(id)
}

// NewSessionID creates a new SessionID using UUID v7
func NewSessionID() (SessionID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return SessionID(id.String()), nil
}

// SessionSecret represents a session secret token
type SessionSecret string

// String returns the string representation
func (s SessionSecret) String() string {
	return string(s)
}

// IncidentID represents an incident identifier
type IncidentID int

// String returns the string representation
func (id IncidentID) String() string {
	return fmt.Sprintf("%d", id)
}

// Int returns the int representation
func (id IncidentID) Int() int {
	return int(id)
}

// Validate checks if the incident ID is valid (positive)
func (id IncidentID) Validate() error {
	if id <= 0 {
		return fmt.Errorf("incident ID must be positive, got: %d", id)
	}
	return nil
}

// IncidentRequestID represents an incident request identifier
type IncidentRequestID string

// String returns the string representation
func (id IncidentRequestID) String() string {
	return string(id)
}

// NewIncidentRequestID creates a new IncidentRequestID
func NewIncidentRequestID() IncidentRequestID {
	return IncidentRequestID(uuid.New().String())
}

// TaskID represents a task identifier
type TaskID string

// String returns the string representation
func (id TaskID) String() string {
	return string(id)
}

// NewTaskID creates a new TaskID
func NewTaskID() TaskID {
	return TaskID(uuid.New().String())
}

// SeverityID represents a severity identifier
type SeverityID string

// String returns the string representation
func (id SeverityID) String() string {
	return string(id)
}

// AssetID represents an asset identifier
type AssetID string

// String returns the string representation
func (id AssetID) String() string {
	return string(id)
}

// PaginationOptions represents pagination parameters for list operations
type PaginationOptions struct {
	// Limit is the maximum number of items to return
	Limit int
	// After is the cursor to start after (for forward pagination)
	After *IncidentID
}

// PaginationResult represents pagination information for a result set
type PaginationResult struct {
	// HasNextPage indicates if there are more items after this page
	HasNextPage bool
	// HasPreviousPage indicates if there are items before this page
	HasPreviousPage bool
	// TotalCount is the total number of items (may be estimated)
	TotalCount int
}
