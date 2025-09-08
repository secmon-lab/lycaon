package model

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// Session represents an authenticated user session
type Session struct {
	ID        types.SessionID     `json:"id"`      // session_id
	Secret    types.SessionSecret `json:"-"`       // session_secret (hidden from JSON)
	UserID    types.UserID        `json:"user_id"` // Associated user ID
	CreatedAt time.Time           `json:"created_at"`
	ExpiresAt time.Time           `json:"expires_at"`
}

// NewSession creates a new Session with UUID v7 ID and random Secret
func NewSession(userID types.UserID, duration time.Duration) (*Session, error) {
	// Generate session ID using types constructor
	sessionID, err := types.NewSessionID()
	if err != nil {
		return nil, err
	}

	// Generate 32-character random secret (24 bytes = 32 chars in base64)
	sessionSecret, err := generateRandomSecret(24)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &Session{
		ID:        sessionID,
		Secret:    types.SessionSecret(sessionSecret),
		UserID:    userID,
		CreatedAt: now,
		ExpiresAt: now.Add(duration),
	}, nil
}

// IsExpired checks if the session has expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if the session is valid (not expired and has proper fields)
func (s *Session) IsValid() bool {
	return s.ID != "" && s.Secret != "" && s.UserID != "" && !s.IsExpired()
}

// generateRandomSecret generates a random base64-encoded string
// byteLength is the number of random bytes to generate (will be ~1.33x longer in base64)
func generateRandomSecret(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use URL-safe base64 encoding without padding for cleaner URLs
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
