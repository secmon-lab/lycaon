package interfaces

//go:generate moq -out mocks/repository_mock.go -pkg mocks . Repository

import (
	"context"

	"github.com/secmon-lab/lycaon/pkg/domain/model"
)

// Repository defines the interface for data persistence
type Repository interface {
	// Message operations
	SaveMessage(ctx context.Context, message *model.Message) error
	GetMessage(ctx context.Context, id string) (*model.Message, error)
	ListMessages(ctx context.Context, channelID string, limit int) ([]*model.Message, error)

	// User operations
	SaveUser(ctx context.Context, user *model.User) error
	GetUser(ctx context.Context, id string) (*model.User, error)
	GetUserBySlackID(ctx context.Context, slackUserID string) (*model.User, error)

	// Session operations
	SaveSession(ctx context.Context, session *model.Session) error
	GetSession(ctx context.Context, id string) (*model.Session, error)
	DeleteSession(ctx context.Context, id string) error

	// Close closes the repository connection
	Close() error
}
