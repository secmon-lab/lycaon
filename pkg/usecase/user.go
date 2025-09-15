package usecase

import (
	"context"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/slack-go/slack"
)

// UserUseCase provides user information management
type UserUseCase struct {
	repo        interfaces.Repository
	slackClient interfaces.SlackClient
	cacheTTL    time.Duration
}

// NewUserUseCase creates a new user usecase
func NewUserUseCase(repo interfaces.Repository, slackClient interfaces.SlackClient) *UserUseCase {
	return &UserUseCase{
		repo:        repo,
		slackClient: slackClient,
		cacheTTL:    24 * time.Hour, // Default cache TTL is 24 hours
	}
}

// GetOrFetchUser retrieves user information with caching
func (u *UserUseCase) GetOrFetchUser(ctx context.Context, slackUserID types.SlackUserID) (*model.User, error) {
	logger := ctxlog.From(ctx)

	// Try to get from repository first
	user, err := u.repo.GetUserBySlackID(ctx, slackUserID)
	if err == nil && user != nil {
		// Check if cache is still valid
		if !user.IsExpired(u.cacheTTL) {
			logger.Debug("User cache hit",
				"slackUserID", slackUserID,
				"name", user.GetDisplayName())
			return user, nil
		}
		// User exists but is expired, try to refresh
		logger.Debug("User data expired, refreshing", "slackUserID", slackUserID)
	}

	// Fetch from Slack API
	freshUser, err := u.fetchUserFromSlack(ctx, slackUserID)
	if err != nil {
		// If we have expired data and failed to fetch, return the expired data
		if user != nil {
			logger.Warn("Failed to fetch user from Slack, using expired data",
				"slackUserID", slackUserID,
				"error", err)
			return user, nil
		}
		return nil, goerr.Wrap(err, "failed to fetch user from Slack",
			goerr.V("slackUserID", slackUserID))
	}

	// If we had an existing user, preserve the ID
	if user != nil {
		freshUser.ID = user.ID
		freshUser.CreatedAt = user.CreatedAt
	}

	// Save updated user
	if err := u.repo.SaveUser(ctx, freshUser); err != nil {
		// Log error but don't fail - we have the user data
		logger.Warn("Failed to save user data",
			"slackUserID", slackUserID,
			"error", err)
	}

	return freshUser, nil
}

// fetchUserFromSlack fetches user information from Slack API
func (u *UserUseCase) fetchUserFromSlack(ctx context.Context, slackUserID types.SlackUserID) (*model.User, error) {
	logger := ctxlog.From(ctx)

	// Get user info from Slack
	slackUser, err := u.slackClient.GetUserInfoContext(ctx, string(slackUserID))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get user info from Slack",
			goerr.V("slackUserID", slackUserID))
	}

	logger.Debug("Fetched user from Slack",
		"slackUserID", slackUserID,
		"name", slackUser.Name,
		"realName", slackUser.RealName)

	// Create or update user
	// Use Slack User ID as the primary User ID (not UUID)
	user := &model.User{
		ID:          types.UserID(slackUserID),
		SlackUserID: slackUserID,
		Name:        slackUser.Name,
		RealName:    slackUser.RealName,
		DisplayName: slackUser.Profile.DisplayName,
		Email:       slackUser.Profile.Email,
		AvatarURL:   u.getBestAvatarURL(slackUser),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return user, nil
}

// getBestAvatarURL returns the best available avatar URL from Slack user profile
func (u *UserUseCase) getBestAvatarURL(user *slack.User) string {
	// Try to get the highest quality avatar available
	if user.Profile.Image512 != "" {
		return user.Profile.Image512
	}
	if user.Profile.Image192 != "" {
		return user.Profile.Image192
	}
	if user.Profile.Image72 != "" {
		return user.Profile.Image72
	}
	if user.Profile.Image48 != "" {
		return user.Profile.Image48
	}
	if user.Profile.Image32 != "" {
		return user.Profile.Image32
	}
	if user.Profile.Image24 != "" {
		return user.Profile.Image24
	}
	return user.Profile.ImageOriginal
}
