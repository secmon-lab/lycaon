package model

import (
	"time"

	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// User represents a Slack user
type User struct {
	ID          types.UserID      `json:"id"`
	SlackUserID types.SlackUserID `json:"slack_user_id"`
	Name        string            `json:"name"`
	Email       string            `json:"email"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// NewUser creates a new User instance
func NewUser(slackUserID types.SlackUserID, name, email string) *User {
	now := time.Now()
	return &User{
		ID:          types.NewUserID(),
		SlackUserID: slackUserID,
		Name:        name,
		Email:       email,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
