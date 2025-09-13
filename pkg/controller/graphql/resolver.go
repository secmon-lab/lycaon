package graphql

import (
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/usecase"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver serves as dependency injection point for the application
type Resolver struct {
	repo       interfaces.Repository
	slackSvc   interfaces.SlackClient
	incidentUC interfaces.Incident
	taskUC     interfaces.Task
	authUC     interfaces.Auth
	statusUC   *usecase.StatusUseCase
	categories *model.CategoriesConfig
	userUC     *usecase.UserUseCase
}

// UseCases contains all usecase interfaces
type UseCases struct {
	IncidentUC interfaces.Incident
	TaskUC     interfaces.Task
	AuthUC     interfaces.Auth
}

// NewResolver creates a new resolver instance
func NewResolver(repo interfaces.Repository, slackSvc interfaces.SlackClient, uc *UseCases, categories *model.CategoriesConfig) *Resolver {
	return &Resolver{
		repo:       repo,
		slackSvc:   slackSvc,
		incidentUC: uc.IncidentUC,
		taskUC:     uc.TaskUC,
		authUC:     uc.AuthUC,
		statusUC:   usecase.NewStatusUseCase(repo, slackSvc),
		categories: categories,
		userUC:     usecase.NewUserUseCase(repo, slackSvc),
	}
}
