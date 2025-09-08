package model

import (
	"time"

	"github.com/google/uuid"
)

// IncidentRequest represents a temporary incident creation request
type IncidentRequest struct {
	ID          string    // UUID
	ChannelID   string    // Origin channel ID
	MessageTS   string    // Message timestamp
	Title       string    // Incident title
	RequestedBy string    // User ID who requested
	CreatedAt   time.Time // When the request was created
	ExpiresAt   time.Time // When the request expires (e.g., 30 minutes)
}

// NewIncidentRequest creates a new incident request
func NewIncidentRequest(channelID, messageTS, title, requestedBy string) *IncidentRequest {
	now := time.Now()
	return &IncidentRequest{
		ID:          uuid.New().String(),
		ChannelID:   channelID,
		MessageTS:   messageTS,
		Title:       title,
		RequestedBy: requestedBy,
		CreatedAt:   now,
		ExpiresAt:   now.Add(30 * time.Minute), // Expire after 30 minutes
	}
}

// IsExpired checks if the request has expired
func (r *IncidentRequest) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}
