package repository

import (
	"context"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// Collection names
	messagesCollection         = "messages"
	usersCollection            = "users"
	sessionsCollection         = "sessions"
	incidentsCollection        = "incidents"
	incidentRequestsCollection = "incident_requests"
	countersCollection         = "counters"

	// Document IDs
	incidentCounterDocID = "incident"

	// Field names
	fieldCurrentNumber = "current_number"
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
			_ = client.Close()
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

	_, err := f.client.Collection(messagesCollection).Doc(message.ID.String()).Set(ctx, message)
	if err != nil {
		return goerr.Wrap(err, "failed to save message to firestore")
	}

	return nil
}

// GetMessage retrieves a message by ID
func (f *Firestore) GetMessage(ctx context.Context, id types.MessageID) (*model.Message, error) {
	if id == "" {
		return nil, goerr.New("message ID is empty")
	}

	doc, err := f.client.Collection(messagesCollection).Doc(id.String()).Get(ctx)
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
func (f *Firestore) ListMessages(ctx context.Context, channelID types.ChannelID, limit int) ([]*model.Message, error) {
	if channelID == "" {
		return nil, goerr.New("channel ID is empty")
	}

	// Simple query without OrderBy to avoid requiring composite index
	// We'll sort in memory instead
	// Note: Field names in Firestore match Go struct field names (e.g., ChannelID not channel_id)
	query := f.client.Collection(messagesCollection).
		Where("ChannelID", "==", channelID.String())

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

	_, err := f.client.Collection(usersCollection).Doc(user.ID.String()).Set(ctx, user)
	if err != nil {
		return goerr.Wrap(err, "failed to save user to firestore")
	}

	return nil
}

// GetUser retrieves a user by ID
func (f *Firestore) GetUser(ctx context.Context, id types.UserID) (*model.User, error) {
	if id == "" {
		return nil, goerr.New("user ID is empty")
	}

	doc, err := f.client.Collection(usersCollection).Doc(id.String()).Get(ctx)
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
func (f *Firestore) GetUserBySlackID(ctx context.Context, slackUserID types.SlackUserID) (*model.User, error) {
	if slackUserID == "" {
		return nil, goerr.New("slack user ID is empty")
	}

	iter := f.client.Collection(usersCollection).
		Where("SlackUserID", "==", slackUserID.String()).
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

	_, err := f.client.Collection(sessionsCollection).Doc(session.ID.String()).Set(ctx, session)
	if err != nil {
		return goerr.Wrap(err, "failed to save session to firestore")
	}

	return nil
}

// GetSession retrieves a session by ID
func (f *Firestore) GetSession(ctx context.Context, id types.SessionID) (*model.Session, error) {
	if id == "" {
		return nil, goerr.New("session ID is empty")
	}

	doc, err := f.client.Collection(sessionsCollection).Doc(id.String()).Get(ctx)
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
func (f *Firestore) DeleteSession(ctx context.Context, id types.SessionID) error {
	if id == "" {
		return goerr.New("session ID is empty")
	}

	// Check if session exists before deletion
	doc := f.client.Collection(sessionsCollection).Doc(id.String())
	_, err := doc.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return goerr.New("session not found")
		}
		return goerr.Wrap(err, "failed to check session existence")
	}

	// Delete the session
	_, err = doc.Delete(ctx)
	if err != nil {
		return goerr.Wrap(err, "failed to delete session from firestore")
	}

	return nil
}

// PutIncident saves an incident to Firestore
func (f *Firestore) PutIncident(ctx context.Context, incident *model.Incident) error {
	if incident == nil {
		return goerr.New("incident is nil")
	}
	if incident.ID <= 0 {
		return goerr.New("incident ID must be positive")
	}

	// Convert ID to string for document ID
	docID := incident.ID.String()
	_, err := f.client.Collection(incidentsCollection).Doc(docID).Set(ctx, incident)
	if err != nil {
		return goerr.Wrap(err, "failed to save incident to firestore")
	}

	return nil
}

// GetIncident retrieves an incident by ID
func (f *Firestore) GetIncident(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
	if id <= 0 {
		return nil, goerr.New("incident ID must be positive")
	}

	// Convert ID to string for document ID
	docID := id.String()
	doc, err := f.client.Collection(incidentsCollection).Doc(docID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(model.ErrIncidentNotFound, "failed to get incident")
		}
		return nil, goerr.Wrap(err, "failed to get incident from firestore")
	}

	var incident model.Incident
	if err := doc.DataTo(&incident); err != nil {
		return nil, goerr.Wrap(err, "failed to decode incident")
	}

	return &incident, nil
}

// GetNextIncidentNumber returns the next available incident number using atomic increment
func (f *Firestore) GetNextIncidentNumber(ctx context.Context) (types.IncidentID, error) {
	counterDoc := f.client.Collection(countersCollection).Doc(incidentCounterDocID)

	var nextNumber types.IncidentID
	err := f.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		doc, err := tx.Get(counterDoc)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				// Initialize counter if it doesn't exist
				nextNumber = 1
				return tx.Set(counterDoc, map[string]any{
					fieldCurrentNumber: int(nextNumber),
				})
			}
			return goerr.Wrap(err, "failed to get counter document")
		}

		// Get current number and increment
		currentNumber, err := doc.DataAt(fieldCurrentNumber)
		if err != nil {
			return goerr.Wrap(err, "failed to get current_number field")
		}

		// Handle both int and int64 types
		switch v := currentNumber.(type) {
		case int64:
			nextNumber = types.IncidentID(v) + 1
		case int:
			nextNumber = types.IncidentID(v) + 1
		default:
			return goerr.New("unexpected type for current_number")
		}

		// Update counter
		return tx.Update(counterDoc, []firestore.Update{
			{Path: fieldCurrentNumber, Value: int(nextNumber)},
		})
	})

	if err != nil {
		return 0, goerr.Wrap(err, "failed to get next incident number")
	}

	return nextNumber, nil
}

// SaveIncidentRequest saves an incident request to Firestore
func (f *Firestore) SaveIncidentRequest(ctx context.Context, request *model.IncidentRequest) error {
	if request == nil {
		return goerr.New("incident request is nil")
	}
	if request.ID == "" {
		return goerr.New("incident request ID is empty")
	}

	_, err := f.client.Collection(incidentRequestsCollection).Doc(request.ID.String()).Set(ctx, request)
	if err != nil {
		return goerr.Wrap(err, "failed to save incident request")
	}

	return nil
}

// GetIncidentRequest retrieves an incident request from Firestore
func (f *Firestore) GetIncidentRequest(ctx context.Context, id types.IncidentRequestID) (*model.IncidentRequest, error) {
	if id == "" {
		return nil, goerr.New("incident request ID is empty")
	}

	doc, err := f.client.Collection(incidentRequestsCollection).Doc(id.String()).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, goerr.Wrap(model.ErrIncidentRequestNotFound, "failed to get incident request")
		}
		return nil, goerr.Wrap(err, "failed to get incident request")
	}

	var request model.IncidentRequest
	if err := doc.DataTo(&request); err != nil {
		return nil, goerr.Wrap(err, "failed to decode incident request")
	}

	return &request, nil
}

// DeleteIncidentRequest deletes an incident request from Firestore
func (f *Firestore) DeleteIncidentRequest(ctx context.Context, id types.IncidentRequestID) error {
	if id == "" {
		return goerr.New("incident request ID is empty")
	}

	_, err := f.client.Collection(incidentRequestsCollection).Doc(id.String()).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return goerr.Wrap(model.ErrIncidentRequestNotFound, "failed to delete incident request")
		}
		return goerr.Wrap(err, "failed to delete incident request")
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
