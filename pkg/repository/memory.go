package repository

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// Memory implements Repository interface with in-memory storage
type Memory struct {
	mu               sync.RWMutex
	messages         map[types.MessageID]*model.Message
	users            map[types.UserID]*model.User
	sessions         map[types.SessionID]*model.Session
	incidents        map[types.IncidentID]*model.Incident
	incidentRequests map[types.IncidentRequestID]*model.IncidentRequest
	tasks            map[types.IncidentID]map[types.TaskID]*model.Task
	statusHistories  map[types.IncidentID][]*model.StatusHistory
	incidentCounter  types.IncidentID
}

// NewMemory creates a new memory repository
func NewMemory() interfaces.Repository {
	return &Memory{
		messages:         make(map[types.MessageID]*model.Message),
		users:            make(map[types.UserID]*model.User),
		sessions:         make(map[types.SessionID]*model.Session),
		incidents:        make(map[types.IncidentID]*model.Incident),
		incidentRequests: make(map[types.IncidentRequestID]*model.IncidentRequest),
		tasks:            make(map[types.IncidentID]map[types.TaskID]*model.Task),
		statusHistories:  make(map[types.IncidentID][]*model.StatusHistory),
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
func (m *Memory) GetMessage(ctx context.Context, id types.MessageID) (*model.Message, error) {
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
func (m *Memory) ListMessages(ctx context.Context, channelID types.ChannelID, limit int) ([]*model.Message, error) {
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
func (m *Memory) GetUser(ctx context.Context, id types.UserID) (*model.User, error) {
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
// Since User.ID is now the Slack User ID, this just calls GetUser
func (m *Memory) GetUserBySlackID(ctx context.Context, slackUserID types.SlackUserID) (*model.User, error) {
	if slackUserID == "" {
		return nil, goerr.New("slack user ID is empty")
	}

	// Convert SlackUserID to UserID and call GetUser
	return m.GetUser(ctx, types.UserID(slackUserID))
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
func (m *Memory) GetSession(ctx context.Context, id types.SessionID) (*model.Session, error) {
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
func (m *Memory) DeleteSession(ctx context.Context, id types.SessionID) error {
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
func (m *Memory) GetIncident(ctx context.Context, id types.IncidentID) (*model.Incident, error) {
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

// GetIncidentByChannelID gets an incident by channel ID from memory
func (m *Memory) GetIncidentByChannelID(ctx context.Context, channelID types.ChannelID) (*model.Incident, error) {
	if channelID == "" {
		return nil, goerr.New("channel ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search through all incidents to find one with matching channel ID
	for _, incident := range m.incidents {
		if incident.ChannelID == channelID {
			// Return a copy to prevent external modifications
			incidentCopy := *incident
			return &incidentCopy, nil
		}
	}

	return nil, goerr.Wrap(model.ErrIncidentNotFound, "failed to get incident by channel ID")
}

// ListIncidents retrieves all incidents from memory
func (m *Memory) ListIncidents(ctx context.Context) ([]*model.Incident, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	incidents := make([]*model.Incident, 0, len(m.incidents))
	for _, incident := range m.incidents {
		// Return a copy to prevent external modifications
		incidentCopy := *incident
		incidents = append(incidents, &incidentCopy)
	}

	// Sort by creation time (newest first)
	sort.Slice(incidents, func(i, j int) bool {
		return incidents[i].CreatedAt.After(incidents[j].CreatedAt)
	})

	return incidents, nil
}

// ListIncidentsSince retrieves incidents created since the specified time
func (m *Memory) ListIncidentsSince(ctx context.Context, since time.Time) ([]*model.Incident, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	incidents := make([]*model.Incident, 0)
	for _, incident := range m.incidents {
		if incident.CreatedAt.After(since) || incident.CreatedAt.Equal(since) {
			// Return a copy to prevent external modifications
			incidentCopy := *incident
			incidents = append(incidents, &incidentCopy)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(incidents, func(i, j int) bool {
		return incidents[i].CreatedAt.After(incidents[j].CreatedAt)
	})

	return incidents, nil
}

// ListIncidentsPaginated retrieves incidents from memory with pagination
func (m *Memory) ListIncidentsPaginated(ctx context.Context, opts types.PaginationOptions) ([]*model.Incident, *types.PaginationResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Default limit if not specified
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Cap at 100
	}

	// Convert map to slice and sort by ID descending (newest first)
	allIncidents := make([]*model.Incident, 0, len(m.incidents))
	for _, incident := range m.incidents {
		// Create a copy to prevent external modifications
		incidentCopy := *incident
		allIncidents = append(allIncidents, &incidentCopy)
	}

	// Sort by ID descending (newest first)
	sort.Slice(allIncidents, func(i, j int) bool {
		return allIncidents[i].ID > allIncidents[j].ID
	})

	// Apply cursor filtering if provided
	startIndex := 0
	if opts.After != nil {
		// Find the index after the cursor
		for i, incident := range allIncidents {
			if incident.ID < *opts.After {
				startIndex = i
				break
			}
		}
		// If cursor not found or at the end, return empty
		if startIndex == 0 && (len(allIncidents) == 0 || allIncidents[0].ID >= *opts.After) {
			return []*model.Incident{}, &types.PaginationResult{
				HasNextPage:     false,
				HasPreviousPage: true,
				TotalCount:      len(allIncidents),
			}, nil
		}
	}

	// Calculate end index
	endIndex := startIndex + limit
	hasNextPage := false
	if endIndex < len(allIncidents) {
		hasNextPage = true
	} else {
		endIndex = len(allIncidents)
	}

	// Get the page of incidents
	pageIncidents := allIncidents[startIndex:endIndex]

	result := &types.PaginationResult{
		HasNextPage:     hasNextPage,
		HasPreviousPage: opts.After != nil,
		TotalCount:      len(allIncidents),
	}

	return pageIncidents, result, nil
}

// GetNextIncidentNumber returns the next available incident number
func (m *Memory) GetNextIncidentNumber(ctx context.Context) (types.IncidentID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.incidentCounter++
	return m.incidentCounter, nil
}

// Clear clears all data (useful for testing)
func (m *Memory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = make(map[types.MessageID]*model.Message)
	m.users = make(map[types.UserID]*model.User)
	m.sessions = make(map[types.SessionID]*model.Session)
	m.incidents = make(map[types.IncidentID]*model.Incident)
	m.incidentRequests = make(map[types.IncidentRequestID]*model.IncidentRequest)
	m.tasks = make(map[types.IncidentID]map[types.TaskID]*model.Task)
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
func (m *Memory) GetIncidentRequest(ctx context.Context, id types.IncidentRequestID) (*model.IncidentRequest, error) {
	if id == "" {
		return nil, goerr.New("incident request ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	request, exists := m.incidentRequests[id]
	if !exists {
		return nil, goerr.Wrap(model.ErrIncidentRequestNotFound, "failed to get incident request")
	}

	return request, nil
}

// DeleteIncidentRequest deletes an incident request from memory
func (m *Memory) DeleteIncidentRequest(ctx context.Context, id types.IncidentRequestID) error {
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

// CreateTask creates a new task
func (m *Memory) CreateTask(ctx context.Context, task *model.Task) error {
	if task == nil {
		return goerr.New("task is nil")
	}
	if task.ID == "" {
		return goerr.New("task ID is empty")
	}
	if task.IncidentID <= 0 {
		return goerr.New("incident ID must be positive", goerr.V("incidentID", task.IncidentID))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize map for incident if not exists
	if m.tasks[task.IncidentID] == nil {
		m.tasks[task.IncidentID] = make(map[types.TaskID]*model.Task)
	}

	// Deep copy to prevent external modifications
	taskCopy := *task
	m.tasks[task.IncidentID][task.ID] = &taskCopy

	return nil
}

// GetTask retrieves a task by ID
func (m *Memory) GetTask(ctx context.Context, taskID types.TaskID) (*model.Task, error) {
	if taskID == "" {
		return nil, goerr.New("task ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Search through all incidents
	for _, tasks := range m.tasks {
		if task, exists := tasks[taskID]; exists {
			// Return a copy to prevent external modifications
			taskCopy := *task
			return &taskCopy, nil
		}
	}

	return nil, goerr.Wrap(model.ErrTaskNotFound, "failed to get task", goerr.V("taskID", taskID))
}

// GetTaskByIncident retrieves a task by incident ID and task ID efficiently
func (m *Memory) GetTaskByIncident(ctx context.Context, incidentID types.IncidentID, taskID types.TaskID) (*model.Task, error) {
	if incidentID <= 0 {
		return nil, goerr.New("incident ID is invalid", goerr.V("incidentID", incidentID))
	}
	if taskID == "" {
		return nil, goerr.New("task ID is empty")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Direct access to the specific incident's tasks
	incidentTasks, exists := m.tasks[incidentID]
	if !exists {
		return nil, goerr.Wrap(model.ErrTaskNotFound, "incident not found",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	task, exists := incidentTasks[taskID]
	if !exists {
		return nil, goerr.Wrap(model.ErrTaskNotFound, "task not found",
			goerr.V("incidentID", incidentID),
			goerr.V("taskID", taskID))
	}

	// Return a copy to prevent external modifications
	taskCopy := *task
	return &taskCopy, nil
}

// UpdateTask updates an existing task
func (m *Memory) UpdateTask(ctx context.Context, task *model.Task) error {
	if task == nil {
		return goerr.New("task is nil")
	}
	if task.ID == "" {
		return goerr.New("task ID is empty")
	}
	if task.IncidentID <= 0 {
		return goerr.New("incident ID must be positive", goerr.V("incidentID", task.IncidentID))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if task exists
	if m.tasks[task.IncidentID] == nil || m.tasks[task.IncidentID][task.ID] == nil {
		return goerr.Wrap(model.ErrTaskNotFound, "failed to update task", goerr.V("taskID", task.ID))
	}

	// Deep copy to prevent external modifications
	taskCopy := *task
	m.tasks[task.IncidentID][task.ID] = &taskCopy

	return nil
}

// DeleteTask deletes a task from memory using direct access with incidentID
func (m *Memory) DeleteTask(ctx context.Context, incidentID types.IncidentID, taskID types.TaskID) error {
	if incidentID <= 0 {
		return goerr.New("incident ID must be positive", goerr.V("incidentID", incidentID))
	}
	if taskID == "" {
		return goerr.New("task ID is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Direct access using incidentID
	tasks, exists := m.tasks[incidentID]
	if !exists {
		return goerr.Wrap(model.ErrTaskNotFound, "incident not found",
			goerr.V("incidentID", incidentID), goerr.V("taskID", taskID))
	}

	if _, exists := tasks[taskID]; !exists {
		return goerr.Wrap(model.ErrTaskNotFound, "task not found",
			goerr.V("incidentID", incidentID), goerr.V("taskID", taskID))
	}

	delete(tasks, taskID)
	return nil
}

// ListTasksByIncident retrieves all tasks for an incident
func (m *Memory) ListTasksByIncident(ctx context.Context, incidentID types.IncidentID) ([]*model.Task, error) {
	if incidentID <= 0 {
		return nil, goerr.New("incident ID must be positive", goerr.V("incidentID", incidentID))
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	incidentTasks, exists := m.tasks[incidentID]
	if !exists {
		// Return empty list if no tasks exist for the incident
		return []*model.Task{}, nil
	}

	// Convert map to slice and create copies
	tasks := make([]*model.Task, 0, len(incidentTasks))
	for _, task := range incidentTasks {
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}

	// Sort by creation time (oldest first)
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})

	return tasks, nil
}

// AddStatusHistory adds a status history entry to memory
func (m *Memory) AddStatusHistory(ctx context.Context, history *model.StatusHistory) error {
	if history == nil {
		return goerr.New("status history is nil")
	}
	if err := history.Validate(); err != nil {
		return goerr.Wrap(err, "invalid status history")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Create a copy to prevent external modification
	historyCopy := *history

	// Initialize slice if not exists
	if m.statusHistories[history.IncidentID] == nil {
		m.statusHistories[history.IncidentID] = []*model.StatusHistory{}
	}

	// Add to slice
	m.statusHistories[history.IncidentID] = append(m.statusHistories[history.IncidentID], &historyCopy)

	return nil
}

// GetStatusHistories retrieves all status histories for an incident
func (m *Memory) GetStatusHistories(ctx context.Context, incidentID types.IncidentID) ([]*model.StatusHistory, error) {
	if err := incidentID.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid incident ID")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	histories, exists := m.statusHistories[incidentID]
	if !exists {
		return []*model.StatusHistory{}, nil // Return empty slice if no histories
	}

	// Create copies to prevent external modification
	result := make([]*model.StatusHistory, 0, len(histories))
	for _, history := range histories {
		historyCopy := *history
		result = append(result, &historyCopy)
	}

	// Sort by timestamp (oldest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].ChangedAt.Before(result[j].ChangedAt)
	})

	return result, nil
}

// UpdateIncidentStatus updates the current status of an incident
func (m *Memory) UpdateIncidentStatus(ctx context.Context, incidentID types.IncidentID, incidentStatus types.IncidentStatus) error {
	if err := incidentID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid incident ID")
	}
	if !incidentStatus.IsValid() {
		return goerr.New("invalid status", goerr.V("status", incidentStatus))
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	incident, exists := m.incidents[incidentID]
	if !exists {
		return goerr.Wrap(model.ErrIncidentNotFound, "incident not found")
	}

	// Update the status
	incident.Status = incidentStatus

	return nil
}

var _ interfaces.Repository = (*Memory)(nil) // Compile-time interface check
