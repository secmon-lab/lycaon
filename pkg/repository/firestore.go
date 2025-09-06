package repository

import (
	"context"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	messagesCollection = "messages"
	usersCollection    = "users"
	sessionsCollection = "sessions"
)

// Firestore implements Repository interface with Firestore
type Firestore struct {
	client *firestore.Client
}

// NewFirestore creates a new Firestore repository
func NewFirestore(ctx context.Context, projectID, databaseID string) (interfaces.Repository, error) {
	logger := ctxlog.From(ctx)

	// Create client with database ID
	client, err := firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create firestore client")
	}

	// Test connection by attempting to read from a collection
	// This will fail fast if the project ID is invalid or if there are permission issues
	_, err = client.Collection(messagesCollection).Limit(1).Documents(ctx).Next()
	if err != nil && err != iterator.Done {
		// Only fail if it's a real error (not just empty collection)
		if status.Code(err) == codes.PermissionDenied || status.Code(err) == codes.Unauthenticated {
			client.Close()
			return nil, goerr.Wrap(err, "failed to connect to firestore project",
				goerr.V("firestore error code", status.Code(err).String()),
			)
		}
		// For other errors (like NotFound for new projects), log but continue
		logger.Debug("Firestore connection test returned error (may be empty collection)",
			"error", err,
			"errorCode", status.Code(err).String(),
		)
	}

	logger.Info("Firestore repository initialized successfully",
		"projectID", projectID,
		"databaseID", databaseID,
	)

	return &Firestore{
		client: client,
	}, nil
}

// SaveMessage saves a message to Firestore
func (f *Firestore) SaveMessage(ctx context.Context, message *model.Message) error {
	if message == nil {
		return goerr.New("message is nil")
	}
	if message.ID == "" {
		return goerr.New("message ID is empty")
	}

	_, err := f.client.Collection(messagesCollection).Doc(message.ID).Set(ctx, message)
	if err != nil {
		return goerr.Wrap(err, "failed to save message to firestore")
	}

	return nil
}

// GetMessage retrieves a message by ID
func (f *Firestore) GetMessage(ctx context.Context, id string) (*model.Message, error) {
	if id == "" {
		return nil, goerr.New("message ID is empty")
	}

	doc, err := f.client.Collection(messagesCollection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("message not found")
		}
		return nil, goerr.Wrap(err, "failed to get message from firestore")
	}

	var message model.Message
	if err := doc.DataTo(&message); err != nil {
		return nil, goerr.Wrap(err, "failed to decode message")
	}

	return &message, nil
}

// ListMessages lists messages for a channel
func (f *Firestore) ListMessages(ctx context.Context, channelID string, limit int) ([]*model.Message, error) {
	if channelID == "" {
		return nil, goerr.New("channel ID is empty")
	}

	// Simple query without OrderBy to avoid requiring composite index
	// We'll sort in memory instead
	// Note: Field names in Firestore match Go struct field names (e.g., ChannelID not channel_id)
	query := f.client.Collection(messagesCollection).
		Where("ChannelID", "==", channelID)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var messages []*model.Message
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, goerr.Wrap(err, "failed to iterate messages")
		}

		var message model.Message
		if err := doc.DataTo(&message); err != nil {
			return nil, goerr.Wrap(err, "failed to decode message")
		}

		messages = append(messages, &message)
	}

	// Sort by timestamp in descending order (newest first)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.After(messages[j].Timestamp)
	})

	// Apply limit after sorting
	if limit > 0 && len(messages) > limit {
		messages = messages[:limit]
	}

	return messages, nil
}

// SaveUser saves a user to Firestore
func (f *Firestore) SaveUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return goerr.New("user is nil")
	}
	if user.ID == "" {
		return goerr.New("user ID is empty")
	}

	_, err := f.client.Collection(usersCollection).Doc(user.ID).Set(ctx, user)
	if err != nil {
		return goerr.Wrap(err, "failed to save user to firestore")
	}

	return nil
}

// GetUser retrieves a user by ID
func (f *Firestore) GetUser(ctx context.Context, id string) (*model.User, error) {
	if id == "" {
		return nil, goerr.New("user ID is empty")
	}

	doc, err := f.client.Collection(usersCollection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("user not found")
		}
		return nil, goerr.Wrap(err, "failed to get user from firestore")
	}

	var user model.User
	if err := doc.DataTo(&user); err != nil {
		return nil, goerr.Wrap(err, "failed to decode user")
	}

	return &user, nil
}

// GetUserBySlackID retrieves a user by Slack ID
func (f *Firestore) GetUserBySlackID(ctx context.Context, slackUserID string) (*model.User, error) {
	if slackUserID == "" {
		return nil, goerr.New("slack user ID is empty")
	}

	iter := f.client.Collection(usersCollection).
		Where("SlackUserID", "==", slackUserID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, goerr.New("user not found")
	}
	if err != nil {
		return nil, goerr.Wrap(err, "failed to query user by slack ID")
	}

	var user model.User
	if err := doc.DataTo(&user); err != nil {
		return nil, goerr.Wrap(err, "failed to decode user")
	}

	return &user, nil
}

// SaveSession saves a session to Firestore
func (f *Firestore) SaveSession(ctx context.Context, session *model.Session) error {
	if session == nil {
		return goerr.New("session is nil")
	}
	if session.ID == "" {
		return goerr.New("session ID is empty")
	}

	_, err := f.client.Collection(sessionsCollection).Doc(session.ID).Set(ctx, session)
	if err != nil {
		return goerr.Wrap(err, "failed to save session to firestore")
	}

	return nil
}

// GetSession retrieves a session by ID
func (f *Firestore) GetSession(ctx context.Context, id string) (*model.Session, error) {
	if id == "" {
		return nil, goerr.New("session ID is empty")
	}

	doc, err := f.client.Collection(sessionsCollection).Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.New("session not found")
		}
		return nil, goerr.Wrap(err, "failed to get session from firestore")
	}

	var session model.Session
	if err := doc.DataTo(&session); err != nil {
		return nil, goerr.Wrap(err, "failed to decode session")
	}

	return &session, nil
}

// DeleteSession deletes a session from Firestore
func (f *Firestore) DeleteSession(ctx context.Context, id string) error {
	if id == "" {
		return goerr.New("session ID is empty")
	}

	_, err := f.client.Collection(sessionsCollection).Doc(id).Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete session from firestore")
	}

	return nil
}

// Close closes the Firestore client
func (f *Firestore) Close() error {
	if f.client != nil {
		return f.client.Close()
	}
	return nil
}

var _ interfaces.Repository = (*Firestore)(nil) // Compile-time interface check
