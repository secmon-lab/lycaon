package model

import (
	"time"

	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// IncidentRequest represents a temporary incident creation request
type IncidentRequest struct {
	ID          types.IncidentRequestID // UUID
	ChannelID   types.ChannelID         // Origin channel ID
	MessageTS   types.MessageTS         // Message timestamp
	Title       string                  // Incident title
	Description string                  // Incident description (optional)
	CategoryID  string                  // Incident category ID (selected by LLM)
	RequestedBy types.SlackUserID       // User ID who requested
	CreatedAt   time.Time               // When the request was created
}

// NewIncidentRequest creates a new incident request
func NewIncidentRequest(channelID types.ChannelID, messageTS types.MessageTS, title, description, categoryID string, requestedBy types.SlackUserID) *IncidentRequest {
	return &IncidentRequest{
		ID:          types.NewIncidentRequestID(),
		ChannelID:   channelID,
		MessageTS:   messageTS,
		Title:       title,
		Description: description,
		CategoryID:  categoryID,
		RequestedBy: requestedBy,
		CreatedAt:   time.Now(),
	}
}
