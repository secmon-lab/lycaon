package model

import (
	"context"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// AuthContextKey is the key for storing AuthContext in context
	authContextKey contextKey = "authContext"
)

// AuthContext contains authentication information
// that should be preserved across async boundaries
type AuthContext struct {
	// User information
	UserID      string `json:"user_id,omitempty"`
	SlackUserID string `json:"slack_user_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
}

// NewAuthContext creates a new AuthContext
func NewAuthContext() *AuthContext {
	return &AuthContext{}
}

// WithAuthContext adds AuthContext to the context
func WithAuthContext(ctx context.Context, authCtx *AuthContext) context.Context {
	if authCtx == nil {
		return ctx
	}
	return context.WithValue(ctx, authContextKey, authCtx)
}

// GetAuthContext retrieves AuthContext from the context
func GetAuthContext(ctx context.Context) (*AuthContext, bool) {
	authCtx, ok := ctx.Value(authContextKey).(*AuthContext)
	return authCtx, ok
}

// GetOrCreateAuthContext retrieves AuthContext from context or creates a new one if not present
func GetOrCreateAuthContext(ctx context.Context) *AuthContext {
	if authCtx, ok := GetAuthContext(ctx); ok && authCtx != nil {
		return authCtx
	}
	return NewAuthContext()
}

// Clone creates a deep copy of the AuthContext
func (a *AuthContext) Clone() *AuthContext {
	if a == nil {
		return nil
	}
	return &AuthContext{
		UserID:      a.UserID,
		SlackUserID: a.SlackUserID,
		SessionID:   a.SessionID,
	}
}
