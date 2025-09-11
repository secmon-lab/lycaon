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
