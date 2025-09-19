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
		// Check if cache is still valid and avatar URL is present
		if !user.IsExpired(u.cacheTTL) && user.AvatarURL != "" {
			logger.Debug("User cache hit",
				"slackUserID", slackUserID,
				"name", user.GetDisplayName())
			return user, nil
		}
		// User exists but is expired or missing avatar, try to refresh
		if user.AvatarURL == "" {
			logger.Debug("User missing avatar URL, refreshing",
				"slackUserID", slackUserID)
		} else {
			logger.Debug("User data expired, refreshing",
				"slackUserID", slackUserID)
		}
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

	// Update and save user with preserved fields
	u.updateAndSaveUser(ctx, freshUser, user, slackUserID)

	return freshUser, nil
}

// RefreshUser forces a refresh of user data from Slack API, bypassing cache
func (u *UserUseCase) RefreshUser(ctx context.Context, slackUserID types.SlackUserID) (*model.User, error) {
	logger := ctxlog.From(ctx)

	logger.Debug("Force refreshing user from Slack", "slackUserID", slackUserID)

	// Get existing user for ID preservation
	existingUser, _ := u.repo.GetUserBySlackID(ctx, slackUserID)

	// Fetch fresh data from Slack API
	freshUser, err := u.fetchUserFromSlack(ctx, slackUserID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to refresh user from Slack",
			goerr.V("slackUserID", slackUserID))
	}

	// Update and save user with preserved fields
	u.updateAndSaveUser(ctx, freshUser, existingUser, slackUserID)

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

// updateAndSaveUser preserves existing user fields and saves the updated user to repository
func (u *UserUseCase) updateAndSaveUser(ctx context.Context, freshUser, existingUser *model.User, slackUserID types.SlackUserID) {
	logger := ctxlog.From(ctx)

	// Preserve existing user ID and creation time if user exists
	if existingUser != nil {
		freshUser.ID = existingUser.ID
		freshUser.CreatedAt = existingUser.CreatedAt
	}

	// Save updated user
	if err := u.repo.SaveUser(ctx, freshUser); err != nil {
		// Log error but don't fail - we have the user data
		logger.Warn("Failed to save user data",
			"slackUserID", slackUserID,
			"error", err)
	}
}

// GetChannelMembers retrieves all members of a Slack channel
func (u *UserUseCase) GetChannelMembers(ctx context.Context, channelID string) ([]*model.User, error) {
	logger := ctxlog.From(ctx)

	if channelID == "" {
		return nil, goerr.New("channel ID is required")
	}

	// Get channel members from Slack API
	params := &slack.GetUsersInConversationParameters{
		ChannelID: channelID,
		Limit:     200, // Get up to 200 members per request
	}

	var allMemberIDs []string
	for {
		memberIDs, nextCursor, err := u.slackClient.GetUsersInConversationContext(ctx, params)
		if err != nil {
			return nil, goerr.Wrap(err, "failed to get channel members from Slack",
				goerr.V("channelID", channelID))
		}

		allMemberIDs = append(allMemberIDs, memberIDs...)

		// If there's no next cursor, we've got all members
		if nextCursor == "" {
			break
		}
		params.Cursor = nextCursor
	}

	logger.Debug("Retrieved channel member IDs from Slack",
		"channelID", channelID,
		"memberCount", len(allMemberIDs))

	// Convert Slack user IDs to User models
	var users []*model.User
	for _, memberID := range allMemberIDs {
		slackUserID := types.SlackUserID(memberID)

		// Get or fetch user information
		user, err := u.GetOrFetchUser(ctx, slackUserID)
		if err != nil {
			// Log error but continue with other users
			logger.Warn("Failed to get user information for channel member",
				"channelID", channelID,
				"slackUserID", slackUserID,
				"error", err)
			continue
		}

		users = append(users, user)
	}

	logger.Info("Retrieved channel members",
		"channelID", channelID,
		"totalMemberIDs", len(allMemberIDs),
		"successfulUsers", len(users))

	return users, nil
}
