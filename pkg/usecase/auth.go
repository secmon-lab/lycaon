package usecase

import (
	"context"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
)

// Auth implements AuthUseCase with repository-based storage
type Auth struct {
	repo interfaces.Repository
}

// NewAuth creates a new Auth use case
func NewAuth(ctx context.Context, repo interfaces.Repository) AuthUseCase {
	return &Auth{
		repo: repo,
	}
}

// CreateSession creates a new session for a user
func (a *Auth) CreateSession(ctx context.Context, slackUserID, userName, userEmail string) (*model.Session, error) {
	logger := ctxlog.From(ctx)

	if slackUserID == "" {
		return nil, goerr.New("slack user ID is required")
	}

	// Find or create user
	user, err := a.repo.GetUserBySlackID(ctx, slackUserID)
	if err != nil {
		// User doesn't exist, create new one
		user = model.NewUser(slackUserID, userName, userEmail)
		user.ID = slackUserID // Use Slack ID directly as user ID

		if err := a.repo.SaveUser(ctx, user); err != nil {
			return nil, goerr.Wrap(err, "failed to save user")
		}

		logger.Info("Created new user",
			"userID", user.ID,
			"slackUserID", slackUserID,
			"userName", userName,
		)
	}

	// Create new session (24 hours validity)
	session, err := model.NewSession(user.ID, 24*time.Hour)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create session")
	}

	// Store session
	if err := a.repo.SaveSession(ctx, session); err != nil {
		return nil, goerr.Wrap(err, "failed to save session")
	}

	logger.Info("Created new session",
		"sessionID", session.ID,
		"userID", user.ID,
		"expiresAt", session.ExpiresAt,
	)

	return session, nil
}

// ValidateSession validates a session by ID and secret
func (a *Auth) ValidateSession(ctx context.Context, sessionID, sessionSecret string) (*model.Session, error) {
	if sessionID == "" || sessionSecret == "" {
		return nil, goerr.New("session ID and secret are required")
	}

	session, err := a.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, goerr.Wrap(err, "session not found")
	}

	// Validate secret
	if session.Secret != sessionSecret {
		return nil, goerr.New("invalid session secret")
	}

	// Check expiration
	if session.IsExpired() {
		return nil, goerr.New("session expired")
	}

	return session, nil
}

// DeleteSession deletes a session
func (a *Auth) DeleteSession(ctx context.Context, sessionID string) error {
	logger := ctxlog.From(ctx)

	if sessionID == "" {
		return goerr.New("session ID is required")
	}

	if err := a.repo.DeleteSession(ctx, sessionID); err != nil {
		return goerr.Wrap(err, "failed to delete session")
	}

	logger.Info("Deleted session",
		"sessionID", sessionID,
	)

	return nil
}

// GetUserFromSession gets user information from a session
func (a *Auth) GetUserFromSession(ctx context.Context, sessionID string) (*model.User, error) {
	if sessionID == "" {
		return nil, goerr.New("session ID is required")
	}

	session, err := a.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, goerr.Wrap(err, "session not found")
	}

	if session.IsExpired() {
		return nil, goerr.New("session expired")
	}

	user, err := a.repo.GetUser(ctx, session.UserID)
	if err != nil {
		return nil, goerr.Wrap(err, "user not found")
	}

	return user, nil
}
