package repository_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/repository"
)

func testRepository(t *testing.T, newRepo func(t *testing.T) interfaces.Repository) {
	t.Run("SaveMessage", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		now := time.Now()
		msg := &model.Message{
			ID:        fmt.Sprintf("msg-%d", now.UnixNano()),
			UserID:    fmt.Sprintf("user-%d", now.UnixNano()),
			UserName:  "Test User",
			ChannelID: fmt.Sprintf("channel-%d", now.UnixNano()),
			Text:      "Test message",
			Timestamp: now,
		}

		err := repo.SaveMessage(ctx, msg)
		gt.NoError(t, err)

		// Verify the message was saved correctly
		retrieved, err := repo.GetMessage(ctx, msg.ID)
		gt.NoError(t, err)
		gt.Equal(t, msg.ID, retrieved.ID)
		gt.Equal(t, msg.UserID, retrieved.UserID)
		gt.Equal(t, msg.UserName, retrieved.UserName)
		gt.Equal(t, msg.ChannelID, retrieved.ChannelID)
		gt.Equal(t, msg.Text, retrieved.Text)
	})

	t.Run("GetMessage", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		now := time.Now()
		msg := &model.Message{
			ID:        fmt.Sprintf("msg-%d", now.UnixNano()),
			UserID:    fmt.Sprintf("user-%d", now.UnixNano()),
			UserName:  "Test User 2",
			ChannelID: fmt.Sprintf("channel-%d", now.UnixNano()),
			Text:      "Another test message",
			Timestamp: now,
		}

		// Save first
		err := repo.SaveMessage(ctx, msg)
		gt.NoError(t, err)

		// Then get and verify all fields
		retrieved, err := repo.GetMessage(ctx, msg.ID)
		gt.NoError(t, err)
		gt.Equal(t, msg.ID, retrieved.ID)
		gt.Equal(t, msg.UserID, retrieved.UserID)
		gt.Equal(t, msg.UserName, retrieved.UserName)
		gt.Equal(t, msg.ChannelID, retrieved.ChannelID)
		gt.Equal(t, msg.Text, retrieved.Text)
		// Timestamp comparison with tolerance for storage precision
		gt.True(t, msg.Timestamp.Sub(retrieved.Timestamp).Abs() < time.Second)
	})

	t.Run("GetMessage_NotFound", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		// Use a random ID that doesn't exist
		nonExistentID := fmt.Sprintf("msg-non-existent-%d", time.Now().UnixNano())
		_, err := repo.GetMessage(ctx, nonExistentID)
		gt.Error(t, err)
	})

	t.Run("ListMessages", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		// Use unique channel ID with timestamp to avoid conflicts
		channelID := fmt.Sprintf("channel-list-%d", time.Now().UnixNano())

		// Save multiple messages with unique IDs
		savedMessages := []*model.Message{}
		baseTime := time.Now()
		userID := fmt.Sprintf("user-%d", baseTime.UnixNano())

		for i := 0; i < 5; i++ {
			msgID := fmt.Sprintf("msg-list-%d-%d", baseTime.UnixNano(), i)
			msg := &model.Message{
				ID:        msgID,
				UserID:    userID,
				UserName:  "List User",
				ChannelID: channelID,
				Text:      fmt.Sprintf("Message %d", i),
				Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			}
			err := repo.SaveMessage(ctx, msg)
			gt.NoError(t, err)
			savedMessages = append(savedMessages, msg)

			// Verify message was saved with all fields correct
			retrieved, err := repo.GetMessage(ctx, msgID)
			gt.NoError(t, err).Must() // Failed to retrieve message after save
			gt.Equal(t, msg.ID, retrieved.ID)
			gt.Equal(t, msg.UserID, retrieved.UserID)
			gt.Equal(t, msg.UserName, retrieved.UserName)
			gt.Equal(t, msg.ChannelID, retrieved.ChannelID)
			gt.Equal(t, msg.Text, retrieved.Text)
		}
		t.Logf("Saved %d messages for channel %s", len(savedMessages), channelID)

		// List messages with limit
		messages, err := repo.ListMessages(ctx, channelID, 3)
		gt.NoError(t, err)
		t.Logf("Retrieved %d messages for channel %s", len(messages), channelID)
		gt.Equal(t, 3, len(messages))

		// Check ordering (newest first)
		for i := 0; i < len(messages)-1; i++ {
			gt.True(t, messages[i].Timestamp.After(messages[i+1].Timestamp))
		}
	})

	t.Run("ListMessages_EmptyChannel", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		// Use a unique channel ID that has no messages
		emptyChannelID := fmt.Sprintf("empty-channel-%d", time.Now().UnixNano())
		messages, err := repo.ListMessages(ctx, emptyChannelID, 10)
		gt.NoError(t, err)
		gt.Equal(t, 0, len(messages))
	})

	t.Run("SaveAndGetUser", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		now := time.Now()
		user := &model.User{
			ID:          fmt.Sprintf("user-%d", now.UnixNano()),
			SlackUserID: fmt.Sprintf("U%d", now.UnixNano()),
			Name:        "Test User",
			Email:       fmt.Sprintf("test-%d@example.com", now.UnixNano()),
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Save user
		err := repo.SaveUser(ctx, user)
		gt.NoError(t, err)

		// Get user by ID and verify all fields
		retrieved, err := repo.GetUser(ctx, user.ID)
		gt.NoError(t, err)
		gt.Equal(t, user.ID, retrieved.ID)
		gt.Equal(t, user.SlackUserID, retrieved.SlackUserID)
		gt.Equal(t, user.Name, retrieved.Name)
		gt.Equal(t, user.Email, retrieved.Email)
		// Check timestamps with tolerance
		gt.True(t, user.CreatedAt.Sub(retrieved.CreatedAt).Abs() < time.Second)
		gt.True(t, user.UpdatedAt.Sub(retrieved.UpdatedAt).Abs() < time.Second)

		// Get user by Slack ID and verify all fields
		retrievedBySlack, err := repo.GetUserBySlackID(ctx, user.SlackUserID)
		gt.NoError(t, err)
		gt.Equal(t, user.ID, retrievedBySlack.ID)
		gt.Equal(t, user.SlackUserID, retrievedBySlack.SlackUserID)
		gt.Equal(t, user.Name, retrievedBySlack.Name)
		gt.Equal(t, user.Email, retrievedBySlack.Email)
	})

	t.Run("GetUser_NotFound", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		// Use random IDs that don't exist
		nonExistentID := fmt.Sprintf("user-non-existent-%d", time.Now().UnixNano())
		nonExistentSlackID := fmt.Sprintf("U-non-existent-%d", time.Now().UnixNano())

		_, err := repo.GetUser(ctx, nonExistentID)
		gt.Error(t, err)

		_, err = repo.GetUserBySlackID(ctx, nonExistentSlackID)
		gt.Error(t, err)
	})

	t.Run("SaveAndGetSession", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		now := time.Now()
		session := &model.Session{
			ID:        fmt.Sprintf("session-%d", now.UnixNano()),
			UserID:    fmt.Sprintf("user-%d", now.UnixNano()),
			Secret:    fmt.Sprintf("secret-hash-%d", now.UnixNano()),
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now,
		}

		// Save session
		err := repo.SaveSession(ctx, session)
		gt.NoError(t, err)

		// Get session and verify all fields
		retrieved, err := repo.GetSession(ctx, session.ID)
		gt.NoError(t, err)
		gt.Equal(t, session.ID, retrieved.ID)
		gt.Equal(t, session.UserID, retrieved.UserID)
		gt.Equal(t, session.Secret, retrieved.Secret)
		// Check timestamps with tolerance
		gt.True(t, session.ExpiresAt.Sub(retrieved.ExpiresAt).Abs() < time.Second)
		gt.True(t, session.CreatedAt.Sub(retrieved.CreatedAt).Abs() < time.Second)
	})

	t.Run("DeleteSession", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		now := time.Now()
		session := &model.Session{
			ID:        fmt.Sprintf("session-del-%d", now.UnixNano()),
			UserID:    fmt.Sprintf("user-del-%d", now.UnixNano()),
			Secret:    fmt.Sprintf("secret-hash-del-%d", now.UnixNano()),
			ExpiresAt: now.Add(24 * time.Hour),
			CreatedAt: now,
		}

		// Save session
		err := repo.SaveSession(ctx, session)
		gt.NoError(t, err)

		// Verify it exists before deletion
		retrieved, err := repo.GetSession(ctx, session.ID)
		gt.NoError(t, err)
		gt.Equal(t, session.ID, retrieved.ID)

		// Delete session
		err = repo.DeleteSession(ctx, session.ID)
		gt.NoError(t, err)

		// Try to get deleted session - should fail
		_, err = repo.GetSession(ctx, session.ID)
		gt.Error(t, err)

		// Try to delete non-existent session with random ID
		nonExistentID := fmt.Sprintf("session-non-existent-%d", time.Now().UnixNano())
		err = repo.DeleteSession(ctx, nonExistentID)
		gt.Error(t, err)
	})
}

func TestMemoryRepository(t *testing.T) {
	testRepository(t, func(t *testing.T) interfaces.Repository {
		return repository.NewMemory()
	})
}

func TestFirestoreRepository(t *testing.T) {
	// Skip test if Firestore test environment variables are not set
	projectID := os.Getenv("TEST_FIRESTORE_PROJECT")
	databaseID := os.Getenv("TEST_FIRESTORE_DATABASE")

	if projectID == "" || databaseID == "" {
		t.Skip("Skipping Firestore test: TEST_FIRESTORE_PROJECT and TEST_FIRESTORE_DATABASE must be set")
	}

	testRepository(t, func(t *testing.T) interfaces.Repository {
		ctx := context.Background()
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		ctx = ctxlog.With(ctx, logger)

		repo, err := repository.NewFirestore(ctx, projectID, databaseID)
		gt.NoError(t, err)
		return repo
	})
}
