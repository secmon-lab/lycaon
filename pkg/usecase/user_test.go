package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces/mocks"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
	"github.com/secmon-lab/lycaon/pkg/usecase"
	"github.com/slack-go/slack"
)

func TestUserUseCaseGetOrFetchUser(t *testing.T) {
	ctx := context.Background()

	t.Run("Fresh user fetch should call Slack API", func(t *testing.T) {
		repo := repository.NewMemory()
		slackClient := &mocks.SlackClientMock{}

		// Mock user data with avatar
		mockSlackUser := &slack.User{
			ID:       "U123456",
			Name:     "test_user",
			RealName: "Test User",
			Profile: slack.UserProfile{
				DisplayName: "Test User",
				Email:       "test@example.com",
				Image512:    "https://example.com/avatar512.jpg",
			},
		}

		slackClient.GetUserInfoContextFunc = func(ctx context.Context, userID string) (*slack.User, error) {
			return mockSlackUser, nil
		}

		userUC := usecase.NewUserUseCase(repo, slackClient)
		slackUserID := types.SlackUserID("U123456")

		user, err := userUC.GetOrFetchUser(ctx, slackUserID)
		gt.NoError(t, err)
		gt.NotNil(t, user)
		gt.Equal(t, user.AvatarURL, "https://example.com/avatar512.jpg")

		// Verify Slack API was called
		calls := slackClient.GetUserInfoContextCalls()
		gt.Equal(t, len(calls), 1)
		gt.Equal(t, calls[0].UserID, "U123456")
	})

	t.Run("Valid cached user should not call Slack API", func(t *testing.T) {
		repo := repository.NewMemory()
		slackClient := &mocks.SlackClientMock{}

		userUC := usecase.NewUserUseCase(repo, slackClient)
		slackUserID := types.SlackUserID("U123456")

		// Create user with valid avatar and recent timestamp
		cachedUser := &model.User{
			ID:          types.UserID(slackUserID),
			Name:        "test_user",
			RealName:    "Test User",
			DisplayName: "Test User",
			Email:       "test@example.com",
			AvatarURL:   "https://example.com/cached_avatar.jpg",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(), // Fresh timestamp
		}

		// Save cached user
		err := repo.SaveUser(ctx, cachedUser)
		gt.NoError(t, err)

		// GetOrFetchUser should use cached data
		user, err := userUC.GetOrFetchUser(ctx, slackUserID)
		gt.NoError(t, err)
		gt.NotNil(t, user)
		gt.Equal(t, user.AvatarURL, "https://example.com/cached_avatar.jpg")

		// Verify Slack API was NOT called
		calls := slackClient.GetUserInfoContextCalls()
		gt.Equal(t, len(calls), 0)
	})

	t.Run("User with empty avatar should call Slack API", func(t *testing.T) {
		repo := repository.NewMemory()
		slackClient := &mocks.SlackClientMock{}

		// Mock user data with avatar
		mockSlackUser := &slack.User{
			ID:       "U123456",
			Name:     "test_user",
			RealName: "Test User",
			Profile: slack.UserProfile{
				DisplayName: "Test User",
				Email:       "test@example.com",
				Image512:    "https://example.com/new_avatar512.jpg",
			},
		}

		slackClient.GetUserInfoContextFunc = func(ctx context.Context, userID string) (*slack.User, error) {
			return mockSlackUser, nil
		}

		userUC := usecase.NewUserUseCase(repo, slackClient)
		slackUserID := types.SlackUserID("U123456")

		// Create user with empty avatar URL but recent timestamp
		userWithoutAvatar := &model.User{
			ID:          types.UserID(slackUserID),
			Name:        "test_user",
			RealName:    "Test User",
			DisplayName: "Test User",
			Email:       "test@example.com",
			AvatarURL:   "", // Empty avatar
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(), // Fresh timestamp but empty avatar
		}

		// Save user with empty avatar
		err := repo.SaveUser(ctx, userWithoutAvatar)
		gt.NoError(t, err)

		// GetOrFetchUser should re-fetch from Slack due to empty avatar
		user, err := userUC.GetOrFetchUser(ctx, slackUserID)
		gt.NoError(t, err)
		gt.NotNil(t, user)
		gt.Equal(t, user.AvatarURL, "https://example.com/new_avatar512.jpg")

		// Verify Slack API was called
		calls := slackClient.GetUserInfoContextCalls()
		gt.Equal(t, len(calls), 1)
		gt.Equal(t, calls[0].UserID, "U123456")
	})

	t.Run("Expired user should call Slack API", func(t *testing.T) {
		repo := repository.NewMemory()
		slackClient := &mocks.SlackClientMock{}

		// Mock user data with avatar
		mockSlackUser := &slack.User{
			ID:       "U123456",
			Name:     "test_user",
			RealName: "Test User",
			Profile: slack.UserProfile{
				DisplayName: "Test User",
				Email:       "test@example.com",
				Image512:    "https://example.com/refreshed_avatar512.jpg",
			},
		}

		slackClient.GetUserInfoContextFunc = func(ctx context.Context, userID string) (*slack.User, error) {
			return mockSlackUser, nil
		}

		userUC := usecase.NewUserUseCase(repo, slackClient)
		slackUserID := types.SlackUserID("U123456")

		// Create expired user (25 hours old, cache TTL is 24 hours)
		expiredUser := &model.User{
			ID:          types.UserID(slackUserID),
			Name:        "test_user",
			RealName:    "Test User",
			DisplayName: "Test User",
			Email:       "test@example.com",
			AvatarURL:   "https://example.com/old_avatar.jpg",
			CreatedAt:   time.Now().Add(-25 * time.Hour),
			UpdatedAt:   time.Now().Add(-25 * time.Hour), // Expired timestamp
		}

		// Save expired user
		err := repo.SaveUser(ctx, expiredUser)
		gt.NoError(t, err)

		// GetOrFetchUser should re-fetch from Slack due to expiration
		user, err := userUC.GetOrFetchUser(ctx, slackUserID)
		gt.NoError(t, err)
		gt.NotNil(t, user)
		gt.Equal(t, user.AvatarURL, "https://example.com/refreshed_avatar512.jpg")

		// Verify Slack API was called
		calls := slackClient.GetUserInfoContextCalls()
		gt.Equal(t, len(calls), 1)
		gt.Equal(t, calls[0].UserID, "U123456")
	})

	t.Run("RefreshUser should always call Slack API", func(t *testing.T) {
		repo := repository.NewMemory()
		slackClient := &mocks.SlackClientMock{}

		// Mock user data with avatar
		mockSlackUser := &slack.User{
			ID:       "U123456",
			Name:     "test_user",
			RealName: "Test User",
			Profile: slack.UserProfile{
				DisplayName: "Test User",
				Email:       "test@example.com",
				Image512:    "https://example.com/force_refreshed_avatar.jpg",
			},
		}

		slackClient.GetUserInfoContextFunc = func(ctx context.Context, userID string) (*slack.User, error) {
			return mockSlackUser, nil
		}

		userUC := usecase.NewUserUseCase(repo, slackClient)
		slackUserID := types.SlackUserID("U123456")

		// Create fresh cached user
		cachedUser := &model.User{
			ID:          types.UserID(slackUserID),
			Name:        "test_user",
			RealName:    "Test User",
			DisplayName: "Test User",
			Email:       "test@example.com",
			AvatarURL:   "https://example.com/cached_avatar.jpg",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(), // Fresh timestamp
		}

		// Save cached user
		err := repo.SaveUser(ctx, cachedUser)
		gt.NoError(t, err)

		// RefreshUser should force refresh even with valid cache
		user, err := userUC.RefreshUser(ctx, slackUserID)
		gt.NoError(t, err)
		gt.NotNil(t, user)
		gt.Equal(t, user.AvatarURL, "https://example.com/force_refreshed_avatar.jpg")

		// Verify Slack API was called
		calls := slackClient.GetUserInfoContextCalls()
		gt.Equal(t, len(calls), 1)
		gt.Equal(t, calls[0].UserID, "U123456")
	})
}

func TestUserUseCaseGetBestAvatarURL(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMemory()
	slackClient := &mocks.SlackClientMock{}
	userUC := usecase.NewUserUseCase(repo, slackClient)

	t.Run("Select highest quality avatar", func(t *testing.T) {
		mockUser := &slack.User{
			Profile: slack.UserProfile{
				Image24:  "https://example.com/24.jpg",
				Image32:  "https://example.com/32.jpg",
				Image48:  "https://example.com/48.jpg",
				Image72:  "https://example.com/72.jpg",
				Image192: "https://example.com/192.jpg",
				Image512: "https://example.com/512.jpg",
			},
		}

		slackClient.GetUserInfoContextFunc = func(ctx context.Context, userID string) (*slack.User, error) {
			return mockUser, nil
		}

		slackUserID := types.SlackUserID("U123456")
		user, err := userUC.GetOrFetchUser(ctx, slackUserID)
		gt.NoError(t, err)
		gt.Equal(t, user.AvatarURL, "https://example.com/512.jpg")
	})

	t.Run("Fallback to lower quality when 512 not available", func(t *testing.T) {
		mockUser := &slack.User{
			Profile: slack.UserProfile{
				Image24:  "https://example.com/24.jpg",
				Image32:  "https://example.com/32.jpg",
				Image192: "https://example.com/192.jpg",
				// No Image512
			},
		}

		slackClient.GetUserInfoContextFunc = func(ctx context.Context, userID string) (*slack.User, error) {
			return mockUser, nil
		}

		slackUserID := types.SlackUserID("U654321")
		user, err := userUC.GetOrFetchUser(ctx, slackUserID)
		gt.NoError(t, err)
		gt.Equal(t, user.AvatarURL, "https://example.com/192.jpg")
	})
}
