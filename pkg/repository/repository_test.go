package repository_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sort"
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
		gt.NoError(t, err).Required()

		// Verify the message was saved correctly
		retrieved, err := repo.GetMessage(ctx, msg.ID)
		gt.NoError(t, err).Required()
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
		gt.NoError(t, err).Required()

		// Then get and verify all fields
		retrieved, err := repo.GetMessage(ctx, msg.ID)
		gt.NoError(t, err).Required()
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
			gt.NoError(t, err).Required()
			savedMessages = append(savedMessages, msg)

			// Verify message was saved with all fields correct
			retrieved, err := repo.GetMessage(ctx, msgID)
			gt.NoError(t, err).Required() // Failed to retrieve message after save
			gt.Equal(t, msg.ID, retrieved.ID)
			gt.Equal(t, msg.UserID, retrieved.UserID)
			gt.Equal(t, msg.UserName, retrieved.UserName)
			gt.Equal(t, msg.ChannelID, retrieved.ChannelID)
			gt.Equal(t, msg.Text, retrieved.Text)
		}
		t.Logf("Saved %d messages for channel %s", len(savedMessages), channelID)

		// List messages with limit
		messages, err := repo.ListMessages(ctx, channelID, 3)
		gt.NoError(t, err).Required()
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
		gt.NoError(t, err).Required()
		gt.Equal(t, 0, len(messages))
	})

	t.Run("SaveAndGetUser", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		now := time.Now()
		slackUserID := types.SlackUserID(fmt.Sprintf("U%d", now.UnixNano()))
		user := &model.User{
			ID:        types.UserID(slackUserID), // ID is now the Slack User ID
			Name:      "Test User",
			Email:     fmt.Sprintf("test-%d@example.com", now.UnixNano()),
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Save user
		err := repo.SaveUser(ctx, user)
		gt.NoError(t, err).Required()

		// Get user by ID and verify all fields
		retrieved, err := repo.GetUser(ctx, user.ID)
		gt.NoError(t, err).Required()
		gt.Equal(t, user.ID, retrieved.ID)
		gt.Equal(t, user.Name, retrieved.Name)
		gt.Equal(t, user.Email, retrieved.Email)
		// Check timestamps with tolerance
		gt.True(t, user.CreatedAt.Sub(retrieved.CreatedAt).Abs() < time.Second)
		gt.True(t, user.UpdatedAt.Sub(retrieved.UpdatedAt).Abs() < time.Second)

		// Get user by Slack ID and verify all fields
		retrievedBySlack, err := repo.GetUserBySlackID(ctx, slackUserID)
		gt.NoError(t, err).Required()
		gt.Equal(t, user.ID, retrievedBySlack.ID)
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
		gt.NoError(t, err).Required()

		// Get session and verify all fields
		retrieved, err := repo.GetSession(ctx, session.ID)
		gt.NoError(t, err).Required()
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
		gt.NoError(t, err).Required()

		// Verify it exists before deletion
		retrieved, err := repo.GetSession(ctx, session.ID)
		gt.NoError(t, err).Required()
		gt.Equal(t, session.ID, retrieved.ID)

		// Delete session
		err = repo.DeleteSession(ctx, session.ID)
		gt.NoError(t, err).Required()

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
		gt.NoError(t, err).Required()
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
		gt.NoError(t, err).Required()

		// Get incident and verify all fields
		retrieved, err := repo.GetIncident(ctx, incident.ID)
		gt.NoError(t, err).Required()
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
		gt.NoError(t, err).Required()
		gt.True(t, num1 > 0)

		// Get second number - should be incremented
		num2, err := repo.GetNextIncidentNumber(ctx)
		gt.NoError(t, err).Required()
		gt.Equal(t, types.IncidentID(int(num1)+1), num2)

		// Get third number - should be incremented again
		num3, err := repo.GetNextIncidentNumber(ctx)
		gt.NoError(t, err).Required()
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

	t.Run("GetIncidentByChannelID", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()

		// Use random IDs to avoid conflicts in parallel tests
		timestamp := time.Now().UnixNano()
		channelID := types.ChannelID(fmt.Sprintf("C%d", timestamp))

		// Create and save an incident
		incident, err := model.NewIncident(
			1,
			"Test Incident for Channel",
			"Description",
			"category-1",
			channelID,
			types.ChannelName("test-channel"),
			types.SlackUserID("U123456"),
			false, // initialTriage
		)
		gt.NoError(t, err).Required()

		// Set the channel ID to match what we're searching for
		incident.ChannelID = channelID

		err = repo.PutIncident(ctx, incident)
		gt.NoError(t, err).Required()

		// Test finding incident by channel ID
		foundIncident, err := repo.GetIncidentByChannelID(ctx, channelID)
		gt.NoError(t, err).Required()
		gt.V(t, foundIncident).NotNil()
		gt.Equal(t, incident.ID, foundIncident.ID)
		gt.Equal(t, incident.ChannelID, foundIncident.ChannelID)
		gt.Equal(t, incident.Title, foundIncident.Title)
	})

	t.Run("GetIncidentByChannelIDNotFound", func(t *testing.T) {
		repo := newRepo(t)
		defer repo.Close()

		ctx := context.Background()
		_, err := repo.GetIncidentByChannelID(ctx, types.ChannelID("C999999"))
		gt.Error(t, err)
	})

	// Pagination tests
	t.Run("ListIncidentsPaginated", func(t *testing.T) {
		t.Run("BasicPagination", func(t *testing.T) {
			repo := newRepo(t)
			defer repo.Close()
			ctx := context.Background()

			// Create test incidents with unique IDs
			now := time.Now()
			baseID := now.UnixNano() / 1000000 // millisecond precision
			var incidents []*model.Incident
			incidentCount := 25
			for i := 1; i <= incidentCount; i++ {
				incident := &model.Incident{
					ID:          types.IncidentID(baseID + int64(i)),
					ChannelID:   types.ChannelID(fmt.Sprintf("test-channel-%d", baseID)),
					ChannelName: types.ChannelName("test-channel"),
					Title:       fmt.Sprintf("Test Incident %d", i),
					Description: "Test Description",
					CategoryID:  "test",
					CreatedBy:   types.SlackUserID("test-user"),
					CreatedAt:   now.Add(time.Duration(-i) * time.Hour),
				}
				gt.NoError(t, repo.PutIncident(ctx, incident))
				incidents = append(incidents, incident)
			}

			// Sort incidents by ID descending (as the pagination does)
			sort.Slice(incidents, func(i, j int) bool {
				return incidents[i].ID > incidents[j].ID
			})

			// Test pagination with our created incidents
			opts := types.PaginationOptions{
				Limit: 10,
				After: nil,
			}

			// First page
			result, pageInfo, err := repo.ListIncidentsPaginated(ctx, opts)
			gt.NoError(t, err).Required()
			gt.Equal(t, 10, len(result))
			gt.True(t, pageInfo.HasNextPage)
			gt.False(t, pageInfo.HasPreviousPage)

			// Verify that our highest ID incident appears in results
			// (it should be one of the first since we just created it)
			foundHighest := false
			for _, inc := range result {
				if inc.ID == incidents[0].ID {
					foundHighest = true
					break
				}
			}
			gt.True(t, foundHighest) // Should find our highest ID incident in first page

			// Verify ordering - each ID should be less than the previous
			for i := 1; i < len(result); i++ {
				gt.True(t, result[i].ID < result[i-1].ID) // IDs should be in descending order
			}

			// Test pagination with cursor
			cursor := result[9].ID
			opts = types.PaginationOptions{
				Limit: 10,
				After: &cursor,
			}
			result2, pageInfo2, err := repo.ListIncidentsPaginated(ctx, opts)
			gt.NoError(t, err).Required()
			gt.True(t, len(result2) <= 10)
			gt.True(t, pageInfo2.HasPreviousPage)

			// Verify that all IDs in second page are less than cursor
			for _, inc := range result2 {
				gt.True(t, inc.ID < cursor) // All IDs in second page should be less than cursor
			}

			// Verify ordering in second page
			for i := 1; i < len(result2); i++ {
				gt.True(t, result2[i].ID < result2[i-1].ID) // IDs should be in descending order
			}

			// Test that we can find all our created incidents somewhere in pagination
			allFoundIDs := make(map[types.IncidentID]bool)

			// Keep paginating until we've seen all our incidents or run out of pages
			var lastCursor *types.IncidentID
			for pagesChecked := 0; pagesChecked < 10; pagesChecked++ {
				opts := types.PaginationOptions{
					Limit: 50,
					After: lastCursor,
				}
				pageResult, pageInfo, err := repo.ListIncidentsPaginated(ctx, opts)
				gt.NoError(t, err).Required()

				for _, inc := range pageResult {
					for _, created := range incidents {
						if inc.ID == created.ID {
							allFoundIDs[created.ID] = true
						}
					}
				}

				if !pageInfo.HasNextPage || len(pageResult) == 0 {
					break
				}
				cursor := pageResult[len(pageResult)-1].ID
				lastCursor = &cursor
			}

			// We should find all our created incidents
			gt.Equal(t, incidentCount, len(allFoundIDs)) // Should find all created incidents
		})

		t.Run("EmptyResult", func(t *testing.T) {
			repo := newRepo(t)
			defer repo.Close()
			ctx := context.Background()

			// Use a cursor that's beyond all existing incidents
			veryLargeCursor := types.IncidentID(1)
			opts := types.PaginationOptions{
				Limit: 10,
				After: &veryLargeCursor,
			}
			result, pageInfo, err := repo.ListIncidentsPaginated(ctx, opts)
			gt.NoError(t, err).Required()
			gt.Equal(t, 0, len(result))
			gt.False(t, pageInfo.HasNextPage)
		})

		t.Run("LimitEnforcement", func(t *testing.T) {
			repo := newRepo(t)
			defer repo.Close()
			ctx := context.Background()

			// Create 5 incidents with unique IDs
			now := time.Now()
			baseID := now.UnixNano() / 1000000
			for i := 1; i <= 5; i++ {
				incident := &model.Incident{
					ID:          types.IncidentID(baseID + int64(i)),
					ChannelID:   types.ChannelID(fmt.Sprintf("test-channel-%d", baseID)),
					ChannelName: types.ChannelName("test-channel"),
					Title:       fmt.Sprintf("Test Incident %d", i),
					Description: "Test Description",
					CategoryID:  "test",
					CreatedBy:   types.SlackUserID("test-user"),
					CreatedAt:   now,
				}
				gt.NoError(t, repo.PutIncident(ctx, incident))
			}

			// Test with limit larger than available items
			opts := types.PaginationOptions{
				Limit: 10,
				After: nil,
			}
			result, _, err := repo.ListIncidentsPaginated(ctx, opts)
			gt.NoError(t, err).Required()
			// Should get at least our 5 incidents
			gt.True(t, len(result) >= 5)

			// Test with zero limit (should use default)
			opts = types.PaginationOptions{
				Limit: 0,
				After: nil,
			}
			result, _, err = repo.ListIncidentsPaginated(ctx, opts)
			gt.NoError(t, err).Required()
			gt.True(t, len(result) > 0)
		})
	})

	t.Run("StatusHistory", func(t *testing.T) {
		t.Run("AddStatusHistory", func(t *testing.T) {
			repo := newRepo(t)
			defer repo.Close()
			ctx := context.Background()

			// Create test incident first
			now := time.Now()
			incidentID := types.IncidentID(now.UnixNano() / 1000000)
			incident := &model.Incident{
				ID:          incidentID,
				ChannelID:   types.ChannelID(fmt.Sprintf("channel-%d", now.UnixNano())),
				ChannelName: types.ChannelName("test-channel"),
				Title:       "Test Incident",
				Description: "Test Description",
				CategoryID:  "test",
				CreatedBy:   types.SlackUserID("test-user"),
				CreatedAt:   now,
				Status:      types.IncidentStatusHandling,
				Lead:        types.SlackUserID("test-user"),
			}
			gt.NoError(t, repo.PutIncident(ctx, incident))

			// Create status history
			history := &model.StatusHistory{
				ID:         types.NewStatusHistoryID(),
				IncidentID: incidentID,
				Status:     types.IncidentStatusMonitoring,
				ChangedBy:  types.SlackUserID("test-user"),
				ChangedAt:  now,
				Note:       "Test status change",
			}

			// Add status history
			err := repo.AddStatusHistory(ctx, history)
			gt.NoError(t, err).Required()

			// Retrieve and verify
			histories, err := repo.GetStatusHistories(ctx, incidentID)
			gt.NoError(t, err).Required()
			gt.A(t, histories).Length(1)
			gt.Equal(t, history.ID, histories[0].ID)
			gt.Equal(t, history.IncidentID, histories[0].IncidentID)
			gt.Equal(t, history.Status, histories[0].Status)
			gt.Equal(t, history.ChangedBy, histories[0].ChangedBy)
			gt.Equal(t, history.Note, histories[0].Note)
			gt.True(t, history.ChangedAt.Sub(histories[0].ChangedAt).Abs() < time.Second)
		})

		t.Run("GetStatusHistory", func(t *testing.T) {
			repo := newRepo(t)
			defer repo.Close()
			ctx := context.Background()

			// Create test incident
			now := time.Now()
			incidentID := types.IncidentID(now.UnixNano() / 1000000)
			incident := &model.Incident{
				ID:          incidentID,
				ChannelID:   types.ChannelID(fmt.Sprintf("channel-%d", now.UnixNano())),
				ChannelName: types.ChannelName("test-channel"),
				Title:       "Test Incident",
				Description: "Test Description",
				CategoryID:  "test",
				CreatedBy:   types.SlackUserID("test-user"),
				CreatedAt:   now,
				Status:      types.IncidentStatusHandling,
				Lead:        types.SlackUserID("test-user"),
			}
			gt.NoError(t, repo.PutIncident(ctx, incident))

			// Add multiple status history entries
			histories := []*model.StatusHistory{
				{
					ID:         types.NewStatusHistoryID(),
					IncidentID: incidentID,
					Status:     types.IncidentStatusHandling,
					ChangedBy:  types.SlackUserID("user1"),
					ChangedAt:  now,
					Note:       "Initial status",
				},
				{
					ID:         types.NewStatusHistoryID(),
					IncidentID: incidentID,
					Status:     types.IncidentStatusMonitoring,
					ChangedBy:  types.SlackUserID("user2"),
					ChangedAt:  now.Add(time.Hour),
					Note:       "Changed to monitoring",
				},
				{
					ID:         types.NewStatusHistoryID(),
					IncidentID: incidentID,
					Status:     types.IncidentStatusClosed,
					ChangedBy:  types.SlackUserID("user3"),
					ChangedAt:  now.Add(2 * time.Hour),
					Note:       "Incident resolved",
				},
			}

			// Add all histories
			for _, h := range histories {
				gt.NoError(t, repo.AddStatusHistory(ctx, h))
			}

			// Retrieve and verify all
			retrieved, err := repo.GetStatusHistories(ctx, incidentID)
			gt.NoError(t, err).Required()
			gt.Equal(t, 3, len(retrieved))

			// Verify they are sorted by timestamp (oldest first)
			for i := 1; i < len(retrieved); i++ {
				gt.True(t, retrieved[i].ChangedAt.After(retrieved[i-1].ChangedAt) || retrieved[i].ChangedAt.Equal(retrieved[i-1].ChangedAt))
			}

			// Verify specific entries exist
			foundStatuses := make(map[types.IncidentStatus]bool)
			for _, h := range retrieved {
				foundStatuses[h.Status] = true
			}
			gt.True(t, foundStatuses[types.IncidentStatusHandling])
			gt.True(t, foundStatuses[types.IncidentStatusMonitoring])
			gt.True(t, foundStatuses[types.IncidentStatusClosed])
		})

		t.Run("GetStatusHistory_NotFound", func(t *testing.T) {
			repo := newRepo(t)
			defer repo.Close()
			ctx := context.Background()

			// Try to get status history for non-existent incident
			nonExistentID := types.IncidentID(time.Now().UnixNano() / 1000000)
			histories, err := repo.GetStatusHistories(ctx, nonExistentID)
			gt.NoError(t, err).Required()
			gt.Equal(t, 0, len(histories)) // Should return empty slice, not error
		})

		t.Run("AddStatusHistory_InvalidIncident", func(t *testing.T) {
			repo := newRepo(t)
			defer repo.Close()
			ctx := context.Background()

			// Try to add status history for non-existent incident
			nonExistentID := types.IncidentID(time.Now().UnixNano() / 1000000)
			history := &model.StatusHistory{
				ID:         types.NewStatusHistoryID(),
				IncidentID: nonExistentID,
				Status:     types.IncidentStatusHandling,
				ChangedBy:  types.SlackUserID("test-user"),
				ChangedAt:  time.Now(),
				Note:       "Test note",
			}

			// This should work even if incident doesn't exist yet
			// (status history can be added before incident is fully created)
			err := repo.AddStatusHistory(ctx, history)
			gt.NoError(t, err)
		})
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
		gt.NoError(t, err).Required()
		return repo
	})
}
