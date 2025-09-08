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
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/secmon-lab/lycaon/pkg/repository"
)

func testRepository(t *testing.T, newRepo func(t *testing.T) interfaces.Repository) {
	t.Run("SaveMessage", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		now := time.Now()
		msg := &model.Message{
			ID:        types.MessageID(fmt.Sprintf("msg-%d", now.UnixNano())),
			UserID:    types.SlackUserID(fmt.Sprintf("user-%d", now.UnixNano())),
			UserName:  "Test User",
			ChannelID: types.ChannelID(fmt.Sprintf("channel-%d", now.UnixNano())),
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
			ID:        types.MessageID(fmt.Sprintf("msg-%d", now.UnixNano())),
			UserID:    types.SlackUserID(fmt.Sprintf("user-%d", now.UnixNano())),
			UserName:  "Test User 2",
			ChannelID: types.ChannelID(fmt.Sprintf("channel-%d", now.UnixNano())),
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
		nonExistentID := types.MessageID(fmt.Sprintf("msg-non-existent-%d", time.Now().UnixNano()))
		_, err := repo.GetMessage(ctx, nonExistentID)
		gt.Error(t, err)
	})

	t.Run("ListMessages", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		// Use unique channel ID with timestamp to avoid conflicts
		channelID := types.ChannelID(fmt.Sprintf("channel-list-%d", time.Now().UnixNano()))

		// Save multiple messages with unique IDs
		savedMessages := []*model.Message{}
		baseTime := time.Now()
		userID := types.SlackUserID(fmt.Sprintf("user-%d", baseTime.UnixNano()))

		for i := 0; i < 5; i++ {
			msgID := types.MessageID(fmt.Sprintf("msg-list-%d-%d", baseTime.UnixNano(), i))
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
		emptyChannelID := types.ChannelID(fmt.Sprintf("empty-channel-%d", time.Now().UnixNano()))
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
			ID:          types.UserID(fmt.Sprintf("user-%d", now.UnixNano())),
			SlackUserID: types.SlackUserID(fmt.Sprintf("U%d", now.UnixNano())),
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
		nonExistentID := types.UserID(fmt.Sprintf("user-non-existent-%d", time.Now().UnixNano()))
		nonExistentSlackID := types.SlackUserID(fmt.Sprintf("U-non-existent-%d", time.Now().UnixNano()))

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
			ID:        types.SessionID(fmt.Sprintf("session-%d", now.UnixNano())),
			UserID:    types.UserID(fmt.Sprintf("user-%d", now.UnixNano())),
			Secret:    types.SessionSecret(fmt.Sprintf("secret-hash-%d", now.UnixNano())),
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
			ID:        types.SessionID(fmt.Sprintf("session-del-%d", now.UnixNano())),
			UserID:    types.UserID(fmt.Sprintf("user-del-%d", now.UnixNano())),
			Secret:    types.SessionSecret(fmt.Sprintf("secret-hash-del-%d", now.UnixNano())),
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
		nonExistentID := types.SessionID(fmt.Sprintf("session-non-existent-%d", time.Now().UnixNano()))
		err = repo.DeleteSession(ctx, nonExistentID)
		gt.Error(t, err)
	})

	t.Run("SaveAndGetIncident", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		now := time.Now()

		// Get next incident number
		incidentNum, err := repo.GetNextIncidentNumber(ctx)
		gt.NoError(t, err)
		gt.True(t, incidentNum > 0)

		// Create incident
		incident := &model.Incident{
			ID:                incidentNum,
			ChannelID:         types.ChannelID(fmt.Sprintf("C-INC-%d", now.UnixNano())),
			ChannelName:       types.ChannelName(fmt.Sprintf("inc-%03d", incidentNum)),
			OriginChannelID:   types.ChannelID(fmt.Sprintf("C-ORIGIN-%d", now.UnixNano())),
			OriginChannelName: types.ChannelName("general"),
			CreatedBy:         types.SlackUserID(fmt.Sprintf("U-CREATOR-%d", now.UnixNano())),
			CreatedAt:         now,
		}

		// Save incident
		err = repo.PutIncident(ctx, incident)
		gt.NoError(t, err)

		// Get incident and verify all fields
		retrieved, err := repo.GetIncident(ctx, incident.ID)
		gt.NoError(t, err)
		gt.Equal(t, incident.ID, retrieved.ID)
		gt.Equal(t, incident.ChannelID, retrieved.ChannelID)
		gt.Equal(t, incident.ChannelName, retrieved.ChannelName)
		gt.Equal(t, incident.OriginChannelID, retrieved.OriginChannelID)
		gt.Equal(t, incident.OriginChannelName, retrieved.OriginChannelName)
		gt.Equal(t, incident.CreatedBy, retrieved.CreatedBy)
		// Check timestamp with tolerance
		gt.True(t, incident.CreatedAt.Sub(retrieved.CreatedAt).Abs() < time.Second)
	})

	t.Run("GetNextIncidentNumber", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()

		// Get first number
		num1, err := repo.GetNextIncidentNumber(ctx)
		gt.NoError(t, err)
		gt.True(t, num1 > 0)

		// Get second number - should be incremented
		num2, err := repo.GetNextIncidentNumber(ctx)
		gt.NoError(t, err)
		gt.Equal(t, types.IncidentID(int(num1)+1), num2)

		// Get third number - should be incremented again
		num3, err := repo.GetNextIncidentNumber(ctx)
		gt.NoError(t, err)
		gt.Equal(t, types.IncidentID(int(num2)+1), num3)
	})

	t.Run("ConcurrentIncidentNumberGeneration", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		numGoroutines := 10
		results := make(chan types.IncidentID, numGoroutines)
		errors := make(chan error, numGoroutines)

		// Launch multiple goroutines to get incident numbers concurrently
		for i := 0; i < numGoroutines; i++ {
			go func() {
				num, err := repo.GetNextIncidentNumber(ctx)
				if err != nil {
					errors <- err
				} else {
					results <- num
				}
			}()
		}

		// Collect results
		numbers := make(map[types.IncidentID]bool)
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-errors:
				t.Fatalf("Error getting incident number: %v", err)
			case num := <-results:
				if numbers[num] {
					t.Fatalf("Duplicate incident number generated: %d", num)
				}
				numbers[num] = true
			}
		}

		// Verify we got unique sequential numbers
		gt.Equal(t, numGoroutines, len(numbers))
	})

	t.Run("GetIncidentNotFound", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()

		// Try to get non-existent incident
		_, err := repo.GetIncident(ctx, types.IncidentID(999999))
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("not found")
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
