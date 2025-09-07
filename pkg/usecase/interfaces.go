package usecase

import (
	"context"

	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/slack-go/slack/slackevents"
)

// SlackMessageUseCase defines the interface for Slack message processing
type SlackMessageUseCase interface {
	// ProcessMessage processes an incoming Slack message
	ProcessMessage(ctx context.Context, event *slackevents.MessageEvent) error

	// GenerateResponse generates an LLM response for a message
	GenerateResponse(ctx context.Context, message *model.Message) (string, error)

	// SaveAndRespond saves a message and generates a response
	SaveAndRespond(ctx context.Context, event *slackevents.MessageEvent) (string, error)
}

// OAuthConfig represents OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	State        string
}

// OAuthURL represents OAuth URL information
type OAuthURL struct {
	URL    string
	State  string
	TeamID string
}

// AuthUseCase defines the interface for authentication operations
type AuthUseCase interface {
	// GenerateOAuthURL generates Slack OAuth URL with team ID from API
	GenerateOAuthURL(ctx context.Context, config OAuthConfig) (*OAuthURL, error)

	// HandleCallback processes the OAuth callback using OpenID Connect
	HandleCallback(ctx context.Context, code, redirectURI string) (*model.User, error)

	// CreateSession creates a new session for a user
	CreateSession(ctx context.Context, slackUserID, userName, userEmail string) (*model.Session, error)

	// ValidateSession validates a session by ID and secret
	ValidateSession(ctx context.Context, sessionID, sessionSecret string) (*model.Session, error)

	// DeleteSession deletes a session
	DeleteSession(ctx context.Context, sessionID string) error

	// GetUserFromSession gets user information from a session
	GetUserFromSession(ctx context.Context, sessionID string) (*model.User, error)
}
