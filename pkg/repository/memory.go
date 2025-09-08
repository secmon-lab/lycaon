package repository

import (
	"context"
	"sort"
	"sync"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
)

// Memory implements Repository interface with in-memory storage
type Memory struct {
	mu               sync.RWMutex
	messages         map[string]*model.Message
	users            map[string]*model.User
	sessions         map[string]*model.Session
	incidents        map[int]*model.Incident
	incidentRequests map[string]*model.IncidentRequest
	incidentCounter  int
}

// NewMemory creates a new memory repository
func NewMemory() interfaces.Repository {
	return &Memory{
		messages:         make(map[string]*model.Message),
		users:            make(map[string]*model.User),
		sessions:         make(map[string]*model.Session),
		incidents:        make(map[int]*model.Incident),
		incidentRequests: make(map[string]*model.IncidentRequest),
		incidentCounter:  0,
	}
}

// SaveMessage saves a message to memory
func (m *Memory) SaveMessage(ctx context.Context, message *model.Message) error {
	if message == nil {
		return goerr.New("message is nil")
	}
	if message.ID == "" {
		return goerr.New("message ID is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages[message.ID] = message
	return nil
}

// GetMessage retrieves a message by ID
func (m *Memory) GetMessage(ctx context.Context, id string) (*model.Message, error) {
	if id == "" {
		return nil, goerr.New("message ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	msg, exists := m.messages[id]
	if !exists {
		return nil, goerr.New("message not found")
	}

	// Return a copy to prevent external modification
	msgCopy := *msg
	return &msgCopy, nil
}

// ListMessages lists messages for a channel
func (m *Memory) ListMessages(ctx context.Context, channelID string, limit int) ([]*model.Message, error) {
	if channelID == "" {
		return nil, goerr.New("channel ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var messages []*model.Message
	for _, msg := range m.messages {
		if msg.ChannelID == channelID {
			msgCopy := *msg
			messages = append(messages, &msgCopy)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.After(messages[j].Timestamp)
	})

	// Apply limit
	if limit > 0 && len(messages) > limit {
		messages = messages[:limit]
	}

	return messages, nil
}

// SaveUser saves a user to memory
func (m *Memory) SaveUser(ctx context.Context, user *model.User) error {
	if user == nil {
		return goerr.New("user is nil")
	}
	if user.ID == "" {
		return goerr.New("user ID is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Deep copy to prevent external modifications
	userCopy := *user
	m.users[user.ID] = &userCopy

	return nil
}

// GetUser retrieves a user by ID
func (m *Memory) GetUser(ctx context.Context, id string) (*model.User, error) {
	if id == "" {
		return nil, goerr.New("user ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	user, exists := m.users[id]
	if !exists {
		return nil, goerr.New("user not found")
	}

	// Return a copy to prevent external modifications
	userCopy := *user
	return &userCopy, nil
}

// GetUserBySlackID retrieves a user by Slack ID
func (m *Memory) GetUserBySlackID(ctx context.Context, slackUserID string) (*model.User, error) {
	if slackUserID == "" {
		return nil, goerr.New("slack user ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.SlackUserID == slackUserID {
			// Return a copy to prevent external modifications
			userCopy := *user
			return &userCopy, nil
		}
	}

	return nil, goerr.New("user not found")
}

// SaveSession saves a session to memory
func (m *Memory) SaveSession(ctx context.Context, session *model.Session) error {
	if session == nil {
		return goerr.New("session is nil")
	}
	if session.ID == "" {
		return goerr.New("session ID is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Deep copy to prevent external modifications
	sessionCopy := *session
	m.sessions[session.ID] = &sessionCopy

	return nil
}

// GetSession retrieves a session by ID
func (m *Memory) GetSession(ctx context.Context, id string) (*model.Session, error) {
	if id == "" {
		return nil, goerr.New("session ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[id]
	if !exists {
		return nil, goerr.New("session not found")
	}

	// Return a copy to prevent external modifications
	sessionCopy := *session
	return &sessionCopy, nil
}

// DeleteSession deletes a session from memory
func (m *Memory) DeleteSession(ctx context.Context, id string) error {
	if id == "" {
		return goerr.New("session ID is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[id]; !exists {
		return goerr.New("session not found")
	}

	delete(m.sessions, id)
	return nil
}

// Close does nothing for memory repository
func (m *Memory) Close() error {
	return nil
}

// PutIncident saves an incident to memory
func (m *Memory) PutIncident(ctx context.Context, incident *model.Incident) error {
	if incident == nil {
		return goerr.New("incident is nil")
	}
	if incident.ID <= 0 {
		return goerr.New("incident ID must be positive")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Deep copy to prevent external modifications
	incidentCopy := *incident
	m.incidents[incident.ID] = &incidentCopy

	return nil
}

// GetIncident retrieves an incident by ID
func (m *Memory) GetIncident(ctx context.Context, id int) (*model.Incident, error) {
	if id <= 0 {
		return nil, goerr.New("incident ID must be positive")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	incident, exists := m.incidents[id]
	if !exists {
		return nil, goerr.Wrap(model.ErrIncidentNotFound, "failed to get incident")
	}

	// Return a copy to prevent external modifications
	incidentCopy := *incident
	return &incidentCopy, nil
}

// GetNextIncidentNumber returns the next available incident number
func (m *Memory) GetNextIncidentNumber(ctx context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.incidentCounter++
	return m.incidentCounter, nil
}

// Clear clears all data (useful for testing)
func (m *Memory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make(map[string]*model.Message)
	m.users = make(map[string]*model.User)
	m.sessions = make(map[string]*model.Session)
	m.incidents = make(map[int]*model.Incident)
	m.incidentRequests = make(map[string]*model.IncidentRequest)
	m.incidentCounter = 0
}

// SaveIncidentRequest saves an incident request to memory
func (m *Memory) SaveIncidentRequest(ctx context.Context, request *model.IncidentRequest) error {
	if request == nil {
		return goerr.New("incident request is nil")
	}
	if request.ID == "" {
		return goerr.New("incident request ID is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.incidentRequests[request.ID] = request
	return nil
}

// GetIncidentRequest retrieves an incident request from memory
func (m *Memory) GetIncidentRequest(ctx context.Context, id string) (*model.IncidentRequest, error) {
	if id == "" {
		return nil, goerr.New("incident request ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	request, exists := m.incidentRequests[id]
	if !exists {
		return nil, goerr.Wrap(model.ErrIncidentRequestNotFound, "failed to get incident request")
	}

	// Check if expired
	if request.IsExpired() {
		return nil, goerr.Wrap(model.ErrIncidentRequestExpired, "incident request expired")
	}

	return request, nil
}

// DeleteIncidentRequest deletes an incident request from memory
func (m *Memory) DeleteIncidentRequest(ctx context.Context, id string) error {
	if id == "" {
		return goerr.New("incident request ID is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.incidentRequests[id]; !exists {
		return goerr.Wrap(model.ErrIncidentRequestNotFound, "failed to delete incident request")
	}

	delete(m.incidentRequests, id)
	return nil
}

// Count returns the number of messages (useful for testing)
func (m *Memory) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.messages)
}

var _ interfaces.Repository = (*Memory)(nil) // Compile-time interface check
