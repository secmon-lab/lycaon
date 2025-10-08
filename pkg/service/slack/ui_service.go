package slack

import (
	"context"

	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// UIService provides high-level Slack operations by delegating to private message and modal services
type UIService struct {
	msg    *messageService
	modal  *modalService
	client interfaces.SlackClient
	config *model.Config
}

// NewUIService creates a new UIService with the given SlackClient and config
// For testing, inject a mock SlackClient
func NewUIService(client interfaces.SlackClient, config *model.Config) *UIService {
	builder := NewBlockBuilder()
	return &UIService{
		msg:    newMessageService(client, builder, config),
		modal:  newModalService(client, builder, config),
		client: client,
		config: config,
	}
}

// Message operations - delegate to messageService

// PostStatusMessage sends a status message to the incident channel
func (s *UIService) PostStatusMessage(ctx context.Context, channelID types.ChannelID, incident *model.Incident, leadName string) error {
	return s.msg.postStatusMessage(ctx, channelID, incident, leadName)
}

// UpdateStatusMessage updates an existing status message
func (s *UIService) UpdateStatusMessage(ctx context.Context, channelID types.ChannelID, messageTS string, incident *model.Incident, leadName string) error {
	return s.msg.updateStatusMessage(ctx, channelID, messageTS, incident, leadName)
}

// PostWelcomeMessage sends a welcome message to the incident channel
func (s *UIService) PostWelcomeMessage(ctx context.Context, channelID types.ChannelID, incident *model.Incident, originChannelName, leadName string) (string, error) {
	return s.msg.postWelcomeMessage(ctx, channelID, incident, originChannelName, leadName)
}

// PostIncidentCreatedNotification sends an incident creation notification
func (s *UIService) PostIncidentCreatedNotification(ctx context.Context, channelID types.ChannelID, messageTS types.MessageTS, originChannelName string, incidentChannelID types.ChannelID, title, categoryID, severityID string) error {
	return s.msg.postIncidentCreatedNotification(ctx, channelID, messageTS, originChannelName, incidentChannelID, title, categoryID, severityID)
}

// UpdateOriginalPromptToUsed updates the original incident prompt to show it was used
func (s *UIService) UpdateOriginalPromptToUsed(ctx context.Context, channelID types.ChannelID, messageTS types.MessageTS, title string) error {
	return s.msg.updateOriginalPromptToUsed(ctx, channelID, messageTS, title)
}

// PostErrorMessage sends an error message
func (s *UIService) PostErrorMessage(ctx context.Context, channelID types.ChannelID, errorText string) error {
	return s.msg.postErrorMessage(ctx, channelID, errorText)
}

// PostTaskMessage sends a task message
func (s *UIService) PostTaskMessage(ctx context.Context, channelID types.ChannelID, task *model.Task, assigneeUsername string) (string, error) {
	return s.msg.postTaskMessage(ctx, channelID, task, assigneeUsername)
}

// UpdateTaskMessage updates an existing task message
func (s *UIService) UpdateTaskMessage(ctx context.Context, channelID types.ChannelID, messageTS string, task *model.Task, assigneeUsername string) error {
	return s.msg.updateTaskMessage(ctx, channelID, messageTS, task, assigneeUsername)
}

// PostIncidentPromptMessage posts an incident prompt message with LLM-generated details
func (s *UIService) PostIncidentPromptMessage(ctx context.Context, channelID string, messageTS, requestID, title, description, categoryID, severityID string) (string, error) {
	return s.msg.postIncidentPromptMessage(ctx, channelID, messageTS, requestID, title, description, categoryID, severityID)
}

// Modal operations - delegate to modalService

// OpenStatusChangeModal opens a status change modal
func (s *UIService) OpenStatusChangeModal(ctx context.Context, triggerID string, incident *model.Incident, channelID, messageTS string) error {
	return s.modal.openStatusChangeModal(ctx, triggerID, incident, channelID, messageTS)
}

// OpenIncidentEditModal opens an incident edit modal
func (s *UIService) OpenIncidentEditModal(ctx context.Context, triggerID, requestID, title, description, categoryID, severityID string, assetIDs []types.AssetID) error {
	return s.modal.openIncidentEditModal(ctx, triggerID, requestID, title, description, categoryID, severityID, assetIDs)
}

// OpenIncidentDetailsEditModal opens an incident details edit modal
func (s *UIService) OpenIncidentDetailsEditModal(ctx context.Context, triggerID string, incident *model.Incident, channelID, messageTS string) error {
	return s.modal.openIncidentDetailsEditModal(ctx, triggerID, incident, channelID, messageTS)
}

// OpenTaskEditModal opens a task edit modal
func (s *UIService) OpenTaskEditModal(ctx context.Context, triggerID string, task *model.Task, channelMembers []types.SlackUserID) error {
	return s.modal.openTaskEditModal(ctx, triggerID, task, channelMembers)
}
