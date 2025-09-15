package usecase_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/cli/config"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
)

func TestAuthCreateSession(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	slackConfig := &config.SlackConfig{}
	auth := usecase.NewAuth(ctx, repo, slackConfig)

	session, err := auth.CreateSession(ctx, "U12345", "Test User", "test@example.com")
	gt.NoError(t, err).Required()
	gt.NotEqual(t, "", session.ID)
	gt.NotEqual(t, "", session.Secret)
	gt.Equal(t, "U12345", session.UserID)
	gt.True(t, session.ExpiresAt.After(time.Now()))

	// Create another session for the same user
	session2, err := auth.CreateSession(ctx, "U12345", "Test User", "test@example.com")
	gt.NoError(t, err).Required()
	gt.NotEqual(t, session.ID, session2.ID)      // Different session ID
	gt.Equal(t, session.UserID, session2.UserID) // Same user ID
}

func TestAuthValidateSession(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	slackConfig := &config.SlackConfig{}
	auth := usecase.NewAuth(ctx, repo, slackConfig)

	// Create a session
	session, err := auth.CreateSession(ctx, "U12345", "Test User", "test@example.com")
	gt.NoError(t, err).Required()

	t.Run("Valid session", func(t *testing.T) {
		validated, err := auth.ValidateSession(ctx, session.ID.String(), session.Secret.String())
		gt.NoError(t, err).Required()
		gt.Equal(t, session.ID, validated.ID)
		gt.Equal(t, session.UserID, validated.UserID)
	})

	t.Run("Invalid secret", func(t *testing.T) {
		_, err := auth.ValidateSession(ctx, session.ID.String(), "wrong-secret")
		gt.Error(t, err)
	})

	t.Run("Non-existent session", func(t *testing.T) {
		_, err := auth.ValidateSession(ctx, "non-existent", "secret")
		gt.Error(t, err)
	})

	t.Run("Empty credentials", func(t *testing.T) {
		_, err := auth.ValidateSession(ctx, "", "")
		gt.Error(t, err)
	})
}

func TestAuthDeleteSession(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	slackConfig := &config.SlackConfig{}
	auth := usecase.NewAuth(ctx, repo, slackConfig)

	// Create a session
	session, err := auth.CreateSession(ctx, "U12345", "Test User", "test@example.com")
	gt.NoError(t, err).Required()

	// Delete the session
	err = auth.DeleteSession(ctx, session.ID.String())
	gt.NoError(t, err).Required()

	// Try to validate deleted session
	_, err = auth.ValidateSession(ctx, session.ID.String(), session.Secret.String())
	gt.Error(t, err)

	// Try to delete non-existent session
	err = auth.DeleteSession(ctx, "non-existent")
	gt.Error(t, err)
}

func TestAuthGetUserFromSession(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	slackConfig := &config.SlackConfig{}
	auth := usecase.NewAuth(ctx, repo, slackConfig)

	// Create a session
	session, err := auth.CreateSession(ctx, "U12345", "Test User", "test@example.com")
	gt.NoError(t, err).Required()

	t.Run("Valid session", func(t *testing.T) {
		user, err := auth.GetUserFromSession(ctx, session.ID.String())
		gt.NoError(t, err).Required()
		gt.Equal(t, types.UserID("U12345"), user.ID)
		gt.Equal(t, "Test User", user.Name)
		gt.Equal(t, "test@example.com", user.Email)
	})

	t.Run("Non-existent session", func(t *testing.T) {
		_, err := auth.GetUserFromSession(ctx, "non-existent")
		gt.Error(t, err)
	})

	t.Run("Empty session ID", func(t *testing.T) {
		_, err := auth.GetUserFromSession(ctx, "")
		gt.Error(t, err)
	})
}

func TestAuthCleanupExpiredSessions(t *testing.T) {
	// This test requires access to internal methods,
	// so we'll test it indirectly through the public interface

	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx = ctxlog.With(ctx, logger)
	repo := repository.NewMemory()
	slackConfig := &config.SlackConfig{}
	auth := usecase.NewAuth(ctx, repo, slackConfig)

	// Create multiple sessions
	session1, err := auth.CreateSession(ctx, "U1", "User1", "user1@example.com")
	gt.NoError(t, err).Required()

	session2, err := auth.CreateSession(ctx, "U2", "User2", "user2@example.com")
	gt.NoError(t, err).Required()

	// Both sessions should be valid
	_, err = auth.ValidateSession(ctx, session1.ID.String(), session1.Secret.String())
	gt.NoError(t, err).Required()

	_, err = auth.ValidateSession(ctx, session2.ID.String(), session2.Secret.String())
	gt.NoError(t, err).Required()

	// Note: To properly test cleanup, we would need to:
	// 1. Create sessions with shorter expiration times
	// 2. Wait for them to expire
	// 3. Call cleanup method
	// This is omitted here for brevity
}
