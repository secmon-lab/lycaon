package model

import (
	"time"

	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// User represents a Slack user
type User struct {
	ID          types.UserID `json:"id"` // This is the Slack User ID
	Name        string       `json:"name"`
	RealName    string       `json:"real_name"`
	DisplayName string       `json:"display_name"`
	Email       string       `json:"email"`
	AvatarURL   string       `json:"avatar_url"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// NewUser creates a new User instance
func NewUser(slackUserID types.SlackUserID, name, email string) *User {
	now := time.Now()
	return &User{
		ID:        types.UserID(slackUserID), // Use Slack User ID as the primary ID
		Name:      name,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// GetDisplayName returns the best available display name
func (u *User) GetDisplayName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.RealName != "" {
		return u.RealName
	}
	if u.Name != "" {
		return u.Name
	}
	return string(u.ID)
}

// IsExpired checks if the user data is expired (older than the given duration)
func (u *User) IsExpired(ttl time.Duration) bool {
	return time.Since(u.UpdatedAt) > ttl
}
