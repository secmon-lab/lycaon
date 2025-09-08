package interfaces

//go:generate moq -out mocks/repository_mock.go -pkg mocks . Repository

import (
	"context"

	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// Repository defines the interface for data persistence
type Repository interface {
	// Message operations
	SaveMessage(ctx context.Context, message *model.Message) error
	GetMessage(ctx context.Context, id types.MessageID) (*model.Message, error)
	ListMessages(ctx context.Context, channelID types.ChannelID, limit int) ([]*model.Message, error)

	// User operations
	SaveUser(ctx context.Context, user *model.User) error
	GetUser(ctx context.Context, id types.UserID) (*model.User, error)
	GetUserBySlackID(ctx context.Context, slackUserID types.SlackUserID) (*model.User, error)

	// Session operations
	SaveSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, id types.SessionID) (*model.Session, error)
	DeleteSession(ctx context.Context, id types.SessionID) error

	// Incident operations
	PutIncident(ctx context.Context, incident *model.Incident) error
	GetIncident(ctx context.Context, id types.IncidentID) (*model.Incident, error)
	GetNextIncidentNumber(ctx context.Context) (types.IncidentID, error)

	// Incident request operations
	SaveIncidentRequest(ctx context.Context, request *model.IncidentRequest) error
	GetIncidentRequest(ctx context.Context, id types.IncidentRequestID) (*model.IncidentRequest, error)
	DeleteIncidentRequest(ctx context.Context, id types.IncidentRequestID) error

	// Close closes the repository connection
	Close() error
}
