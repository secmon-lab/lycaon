package slack

import (
	"context"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// modalService handles Slack modal operations (private service)
type modalService struct {
	client  interfaces.SlackClient
	builder *BlockBuilder
	config  *model.Config
}

// newModalService creates a new modalService instance
func newModalService(client interfaces.SlackClient, builder *BlockBuilder, config *model.Config) *modalService {
	return &modalService{
		client:  client,
		builder: builder,
		config:  config,
	}
}

// openStatusChangeModal opens a status change modal
func (s *modalService) openStatusChangeModal(ctx context.Context, triggerID string, incident *model.Incident, channelID, messageTS string) error {
	if triggerID == "" {
		return goerr.New("trigger ID is required")
	}

	// Build status selection modal
	modalView := s.builder.BuildStatusSelectionModal(incident, channelID, messageTS)

	// Open modal
	_, err := s.client.OpenView(ctx, triggerID, modalView)
	if err != nil {
		return goerr.Wrap(err, "failed to open status selection modal")
	}

	return nil
}

// openIncidentEditModal opens an incident edit modal
func (s *modalService) openIncidentEditModal(ctx context.Context, triggerID, requestID, title, description, categoryID, severityID string, assetIDs []types.AssetID) error {
	if triggerID == "" {
		return goerr.New("trigger ID is required")
	}

	// Get categories, severities, and assets from config
	var categories []model.Category
	var severities []model.Severity
	var assets []model.Asset
	if s.config != nil {
		if s.config.Categories != nil {
			categories = s.config.Categories
		}
		if s.config.Severities != nil {
			severities = s.config.Severities
		}
		if s.config.Assets != nil {
			assets = s.config.Assets
		}
	}

	// Build edit modal
	modal := s.builder.BuildIncidentEditModal(requestID, title, description, categoryID, severityID, assetIDs, categories, severities, assets)

	// Open the modal
	_, err := s.client.OpenView(ctx, triggerID, modal)
	if err != nil {
		return goerr.Wrap(err, "failed to open incident edit modal")
	}

	return nil
}

// openIncidentDetailsEditModal opens an incident details edit modal
func (s *modalService) openIncidentDetailsEditModal(ctx context.Context, triggerID string, incident *model.Incident, channelID, messageTS string) error {
	if triggerID == "" {
		return goerr.New("trigger ID is required")
	}

	// Get severities and assets from config
	var severities []model.Severity
	var assets []model.Asset
	if s.config != nil {
		if s.config.Severities != nil {
			severities = s.config.Severities
		}
		if s.config.Assets != nil {
			assets = s.config.Assets
		}
	}

	// Build edit incident details modal
	modal := s.builder.BuildEditIncidentDetailsModal(incident, channelID, messageTS, severities, assets)

	// Open the modal
	_, err := s.client.OpenView(ctx, triggerID, modal)
	if err != nil {
		return goerr.Wrap(err, "failed to open edit incident details modal")
	}

	return nil
}

// openTaskEditModal opens a task edit modal
func (s *modalService) openTaskEditModal(ctx context.Context, triggerID string, task *model.Task, channelMembers []types.SlackUserID) error {
	if triggerID == "" {
		return goerr.New("trigger ID is required")
	}

	// Build task edit modal
	modal := BuildTaskEditModal(task, channelMembers)

	// Open the modal
	_, err := s.client.OpenView(ctx, triggerID, modal)
	if err != nil {
		return goerr.Wrap(err, "failed to open task edit modal")
	}

	return nil
}
