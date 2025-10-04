package graphql

import (
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	slackservice "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/secmon-lab/lycaon/pkg/usecase"
)

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

// Resolver serves as dependency injection point for the application
type Resolver struct {
	repo        interfaces.Repository
	slackSvc    interfaces.SlackClient
	incidentUC  interfaces.Incident
	taskUC      interfaces.Task
	authUC      interfaces.Auth
	statusUC    *usecase.StatusUseCase
	modelConfig *model.Config
	userUC      *usecase.UserUseCase
}

// UseCases contains all usecase interfaces
type UseCases struct {
	IncidentUC interfaces.Incident
	TaskUC     interfaces.Task
	AuthUC     interfaces.Auth
}

// NewResolver creates a new resolver instance
func NewResolver(repo interfaces.Repository, slackSvc interfaces.SlackClient, uc *UseCases, modelConfig *model.Config) *Resolver {
	slackUIService := slackservice.NewUIService(slackSvc, modelConfig)
	return &Resolver{
		repo:        repo,
		slackSvc:    slackSvc,
		incidentUC:  uc.IncidentUC,
		taskUC:      uc.TaskUC,
		authUC:      uc.AuthUC,
		statusUC:    usecase.NewStatusUseCase(repo, slackUIService, modelConfig),
		modelConfig: modelConfig,
		userUC:      usecase.NewUserUseCase(repo, slackSvc),
	}
}
