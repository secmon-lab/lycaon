package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/secmon-lab/lycaon/pkg/utils/apperr"
	"github.com/slack-go/slack"
)

// IncidentConfig holds configuration for Incident use case
type IncidentConfig struct {
	channelPrefix string
	frontendURL   string
}

// IncidentOption is a functional option for configuring Incident
type IncidentOption func(*IncidentConfig)

// WithChannelPrefix sets the channel prefix for incident channels
func WithChannelPrefix(prefix string) IncidentOption {
	return func(c *IncidentConfig) {
		c.channelPrefix = prefix
	}
}

// WithFrontendURL sets the frontend URL for bookmark generation
func WithFrontendURL(url string) IncidentOption {
	return func(c *IncidentConfig) {
		c.frontendURL = url
	}
}

// NewIncidentConfig creates a new IncidentConfig with default values and optional settings
func NewIncidentConfig(opts ...IncidentOption) *IncidentConfig {
	config := &IncidentConfig{
		channelPrefix: "inc", // Default value
	}

	// Apply optional configurations
	for _, opt := range opts {
		opt(config)
	}

	return config
}

// Incident implements Incident interface
type Incident struct {
	repo        interfaces.Repository
	slackClient interfaces.SlackClient
	slackSvc    *slackSvc.UIService
	modelConfig *model.Config
	invite      interfaces.Invite
	config      *IncidentConfig
}

// NewIncident creates a new Incident instance with configuration
func NewIncident(repo interfaces.Repository, slackClient interfaces.SlackClient, slackService *slackSvc.UIService, modelConfig *model.Config, invite interfaces.Invite, config *IncidentConfig) *Incident {
	return &Incident{
		repo:        repo,
		slackClient: slackClient,
		slackSvc:    slackService,
		modelConfig: modelConfig,
		invite:      invite,
		config:      config,
	}
}

// CreateIncident creates a new incident
func (u *Incident) CreateIncident(ctx context.Context, req *model.CreateIncidentRequest) (*model.Incident, error) {
	// Validate severity ID if severities config is available and severity ID is provided
	if u.modelConfig != nil && req.SeverityID != "" {
		severity := u.modelConfig.FindSeverityByID(req.SeverityID)
		if severity == nil {
			return nil, goerr.New("invalid severity ID",
				goerr.V("severityID", req.SeverityID))
		}
	}

	// Get next incident number
	incidentNumber, err := u.repo.GetNextIncidentNumber(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get next incident number")
	}

	// Get Team ID from Slack API
	var teamID types.TeamID
	authResp, err := u.slackClient.AuthTestContext(ctx)
	if err != nil {
		// Log warning but continue without TeamID
		ctxlog.From(ctx).Warn("Failed to get Team ID from Slack API",
			"error", err,
			"description", "Slack channel links will not work without Team ID")
	} else {
		teamID = types.TeamID(authResp.TeamID)
	}

	// Create incident model
	incident, err := model.NewIncident(u.config.channelPrefix, incidentNumber, req.Title, req.Description, req.CategoryID, types.SeverityID(req.SeverityID), types.ChannelID(req.OriginChannelID), types.ChannelName(req.OriginChannelName), teamID, types.SlackUserID(req.CreatedBy), req.InitialTriage)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create incident model")
	}

	// Create Slack channel with the generated channel name (includes title if provided)
	channelName := incident.ChannelName

	channel, err := u.slackClient.CreateConversation(ctx, slack.CreateConversationParams{
		ChannelName: channelName.String(),
		IsPrivate:   false,
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Slack channel")
	}

	// Set channel purpose/description if title is provided
	if req.Title != "" {
		_, err = u.slackClient.SetPurposeOfConversationContext(ctx, channel.ID, req.Title)
		if err != nil {
			// Log error but don't fail - setting purpose is nice to have but not critical
			apperr.Handle(ctx, err)
		}
	}

	// Set channel ID
	incident.ChannelID = types.ChannelID(channel.ID)

	// Invite creator to the channel
	_, err = u.slackClient.InviteUsersToConversation(ctx, channel.ID, req.CreatedBy)
	if err != nil {
		// Log error but don't fail - user might already be in channel or invitation might fail for other reasons
		apperr.Handle(ctx, err)
	}

	// Add bookmark to Web UI if frontend URL is configured
	if u.config.frontendURL != "" {
		bookmarkTitle := fmt.Sprintf("Incident #%d - Web UI", incidentNumber)
		bookmarkURL := fmt.Sprintf("%s/incidents/%d", u.config.frontendURL, incident.ID)

		err = u.slackClient.AddBookmark(ctx, channel.ID, bookmarkTitle, bookmarkURL)
		if err != nil {
			// Log error but don't fail - bookmark is nice to have but not critical
			apperr.Handle(ctx, err)
		} else {
			ctxlog.From(ctx).Info("Added incident bookmark to channel",
				"channelID", channel.ID,
				"incidentID", incident.ID,
				"bookmarkURL", bookmarkURL)
		}
	}

	// Post welcome message to the incident channel
	leadName := string(incident.Lead)
	if leadName == "" {
		leadName = "Unassigned"
	}
	welcomeTS, err := u.slackSvc.PostWelcomeMessage(
		ctx,
		incident.ChannelID,
		incident,
		req.OriginChannelName,
		leadName,
	)
	if err != nil {
		// Log error but don't fail - welcome message is nice to have but not critical
		apperr.Handle(ctx, err)
	} else {
		// Save the welcome message timestamp for later updates
		incident.WelcomeMessageTS = welcomeTS
	}

	// Save incident to repository
	if err := u.repo.PutIncident(ctx, incident); err != nil {
		return nil, goerr.Wrap(err, "failed to save incident")
	}

	// Save initial status history
	initialHistory := &model.StatusHistory{
		ID:         types.NewStatusHistoryID(),
		IncidentID: incident.ID,
		Status:     incident.Status,
		ChangedBy:  incident.CreatedBy,
		ChangedAt:  incident.CreatedAt,
		Note:       "Incident created",
	}
	if err := u.repo.AddStatusHistory(ctx, initialHistory); err != nil {
		// Log error and fail the operation to prevent inconsistent data
		apperr.Handle(ctx, err)
		return nil, goerr.Wrap(err, "failed to save initial status history")
	}

	// Category-based invitation process (serial execution)
	// Note: This function assumes it's already dispatched asynchronously in the Controller layer
	if incident.CategoryID != "" && u.modelConfig != nil && u.invite != nil {
		// Get invitation targets from category configuration
		category := u.modelConfig.FindCategoryByID(incident.CategoryID)
		if category == nil {
			// No category is normal - log and continue
			ctxlog.From(ctx).Info("Category not found",
				"categoryID", incident.CategoryID)
		} else if len(category.InviteUsers) == 0 && len(category.InviteGroups) == 0 {
			// No invitation settings is normal - log and continue
			ctxlog.From(ctx).Info("No invitation settings for category",
				"categoryID", incident.CategoryID)
		} else {
			// Execute invitation synchronously
			_, err := u.invite.InviteUsersByList(
				ctx,
				category.InviteUsers,
				category.InviteGroups,
				incident.ChannelID,
			)
			if err != nil {
				// Log error but continue with incident creation
				apperr.Handle(ctx, err)
			}
		}
	}

	return incident, nil
}

// GetIncident retrieves an incident by ID
func (u *Incident) GetIncident(ctx context.Context, id int) (*model.Incident, error) {
	incident, err := u.repo.GetIncident(ctx, types.IncidentID(id))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident")
	}
	return incident, nil
}

// GetIncidentByChannelID gets an incident by channel ID
func (u *Incident) GetIncidentByChannelID(ctx context.Context, channelID types.ChannelID) (*model.Incident, error) {
	incident, err := u.repo.GetIncidentByChannelID(ctx, channelID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident by channel ID")
	}
	return incident, nil
}

// HandleCreateIncidentAction handles the complete flow when a user clicks the create incident button
func (u *Incident) HandleCreateIncidentAction(ctx context.Context, requestID, userID string) (*model.Incident, error) {
	return u.handleCreateIncidentFromRequest(ctx, requestID, userID, nil)
}

// UpdateIncidentDetails updates incident title, description, and lead
func (u *Incident) UpdateIncidentDetails(ctx context.Context, incidentID types.IncidentID, title, description string, lead types.SlackUserID, severityID string, updatedBy types.SlackUserID) (*model.Incident, error) {
	// Validate incident ID
	if err := incidentID.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid incident ID")
	}

	// Validate severity ID if provided and severities config is available
	if severityID != "" && u.modelConfig != nil {
		severity := u.modelConfig.FindSeverityByID(severityID)
		if severity == nil {
			return nil, goerr.New("invalid severity ID", goerr.V("severityID", severityID))
		}
	}

	// Get existing incident
	incident, err := u.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident")
	}

	// Track if any changes were made
	hasChanges := false

	// Update fields if they have changed
	if title != incident.Title {
		incident.Title = title
		hasChanges = true
	}
	if description != incident.Description {
		incident.Description = description
		hasChanges = true
	}
	if lead != incident.Lead {
		incident.Lead = lead
		hasChanges = true
	}
	if severityID != "" && types.SeverityID(severityID) != incident.SeverityID {
		incident.SeverityID = types.SeverityID(severityID)
		hasChanges = true
	}

	// Only update if there are changes
	if !hasChanges {
		return incident, nil
	}

	// Save updated incident using PutIncident
	if err := u.repo.PutIncident(ctx, incident); err != nil {
		return nil, goerr.Wrap(err, "failed to update incident")
	}

	return incident, nil
}

// UpdateIncident updates incident with UpdateIncidentRequest
func (u *Incident) UpdateIncident(ctx context.Context, incidentID types.IncidentID, req model.UpdateIncidentRequest) (*model.Incident, error) {
	// Validate incident ID
	if err := incidentID.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid incident ID")
	}

	// Validate severity ID if provided
	if req.SeverityID != nil && u.modelConfig != nil {
		severityID := string(*req.SeverityID)
		if severityID != "" {
			severity := u.modelConfig.FindSeverityByID(severityID)
			if severity == nil {
				return nil, goerr.New("invalid severity ID",
					goerr.V("severityID", severityID))
			}
		}
	}

	// Get existing incident
	incident, err := u.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident")
	}

	// Track if any changes were made
	hasChanges := false
	var changes []string

	// Update fields if they have changed
	if req.Title != nil && *req.Title != incident.Title {
		incident.Title = *req.Title
		hasChanges = true
		changes = append(changes, "title")
	}
	if req.Description != nil && *req.Description != incident.Description {
		incident.Description = *req.Description
		hasChanges = true
		changes = append(changes, "description")
	}
	if req.Lead != nil && *req.Lead != incident.Lead {
		incident.Lead = *req.Lead
		hasChanges = true
		changes = append(changes, "lead")
	}
	if req.Status != nil && *req.Status != incident.Status {
		incident.Status = *req.Status
		hasChanges = true
		changes = append(changes, "status")
	}
	if req.SeverityID != nil && *req.SeverityID != incident.SeverityID {
		incident.SeverityID = *req.SeverityID
		hasChanges = true
		changes = append(changes, "severity")
	}

	// Only update if there are changes
	if !hasChanges {
		return incident, nil
	}

	// Save updated incident using PutIncident
	if err := u.repo.PutIncident(ctx, incident); err != nil {
		return nil, goerr.Wrap(err, "failed to update incident")
	}

	// Post update notification to incident channel
	if incident.ChannelID != "" {
		// Build a simple notification message
		message := fmt.Sprintf("📝 Incident details updated: %s", strings.Join(changes, ", "))
		_, _, err = u.slackClient.PostMessage(
			ctx,
			string(incident.ChannelID),
			slack.MsgOptionText(message, false),
		)
		if err != nil {
			// Log error but don't fail - notification is nice to have but not critical
			apperr.Handle(ctx, err)
		}
	}

	return incident, nil
}
