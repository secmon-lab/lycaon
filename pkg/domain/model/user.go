package model

import "time"

// User represents a Slack user
type User struct {
	ID          string    `json:"id"`
	SlackUserID string    `json:"slack_user_id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewUser creates a new User instance
func NewUser(slackUserID, name, email string) *User {
	now := time.Now()
	return &User{
		SlackUserID: slackUserID,
		Name:        name,
		Email:       email,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
