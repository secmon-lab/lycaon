package model

import (
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// StatusHistory represents a status change history entry
type StatusHistory struct {
	ID         types.StatusHistoryID `json:"id"`
	IncidentID types.IncidentID      `json:"incidentId"`
	Status     types.IncidentStatus  `json:"status"`
	ChangedBy  types.SlackUserID     `json:"changedBy"`
	ChangedAt  time.Time             `json:"changedAt"`
	Note       string                `json:"note,omitempty"`
}

// StatusHistoryWithUser represents a status history entry with user information
type StatusHistoryWithUser struct {
	StatusHistory
	User *User `json:"user"`
}

// NewStatusHistory creates a new status history entry
func NewStatusHistory(incidentID types.IncidentID, status types.IncidentStatus, changedBy types.SlackUserID, note string) (*StatusHistory, error) {
	if err := incidentID.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid incident ID")
	}

	if !status.IsValid() {
		return nil, goerr.New("invalid status", goerr.V("status", status))
	}

	if changedBy == "" {
		return nil, goerr.New("changed by user ID is required")
	}

	return &StatusHistory{
		ID:         types.NewStatusHistoryID(),
		IncidentID: incidentID,
		Status:     status,
		ChangedBy:  changedBy,
		ChangedAt:  time.Now(),
		Note:       note,
	}, nil
}

// Validate validates the status history entry
func (sh *StatusHistory) Validate() error {
	if err := sh.ID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid status history ID")
	}

	if err := sh.IncidentID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid incident ID")
	}

	if !sh.Status.IsValid() {
		return goerr.New("invalid status", goerr.V("status", sh.Status))
	}

	if sh.ChangedBy == "" {
		return goerr.New("changed by user ID is required")
	}

	if sh.ChangedAt.IsZero() {
		return goerr.New("changed at timestamp is required")
	}

	return nil
}

// NewStatusHistoryWithUser creates a new status history with user information
func NewStatusHistoryWithUser(history *StatusHistory, user *User) *StatusHistoryWithUser {
	return &StatusHistoryWithUser{
		StatusHistory: *history,
		User:          user,
	}
}
