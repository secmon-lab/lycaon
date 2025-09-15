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

// Incident implements Incident interface
type Incident struct {
	repo         interfaces.Repository
	slackClient  interfaces.SlackClient
	blockBuilder *slackSvc.BlockBuilder
	categories   *model.CategoriesConfig
	invite       interfaces.Invite
}

// NewIncident creates a new Incident instance with a custom SlackClient
func NewIncident(repo interfaces.Repository, slackClient interfaces.SlackClient, categories *model.CategoriesConfig, invite interfaces.Invite) *Incident {
	return &Incident{
		repo:         repo,
		slackClient:  slackClient,
		blockBuilder: slackSvc.NewBlockBuilder(),
		categories:   categories,
		invite:       invite,
	}
}

// CreateIncident creates a new incident
func (u *Incident) CreateIncident(ctx context.Context, req *model.CreateIncidentRequest) (*model.Incident, error) {
	// Get next incident number
	incidentNumber, err := u.repo.GetNextIncidentNumber(ctx)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get next incident number")
	}

	// Create incident model
	incident, err := model.NewIncident(incidentNumber, req.Title, req.Description, req.CategoryID, types.ChannelID(req.OriginChannelID), types.ChannelName(req.OriginChannelName), types.SlackUserID(req.CreatedBy), req.InitialTriage)
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

	// Post welcome message to the incident channel
	welcomeBlocks := u.blockBuilder.BuildIncidentChannelWelcomeBlocks(int(incidentNumber), req.OriginChannelName, req.CreatedBy, incident.Description, req.CategoryID, u.categories)
	_, _, err = u.slackClient.PostMessage(
		ctx,
		channel.ID,
		slack.MsgOptionBlocks(welcomeBlocks...),
	)
	if err != nil {
		// Log error but don't fail - welcome message is nice to have but not critical
		apperr.Handle(ctx, err)
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
	if incident.CategoryID != "" && u.categories != nil && u.invite != nil {
		// Get invitation targets from category configuration
		category := u.categories.FindCategoryByID(incident.CategoryID)
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

// CreateIncidentFromInteraction handles the complete incident creation flow from a Slack interaction
func (u *Incident) CreateIncidentFromInteraction(ctx context.Context, originChannelID, title, userID string) (*model.Incident, error) {
	// Get channel info from Slack
	channelInfo, err := u.slackClient.GetConversationInfo(ctx, originChannelID, false)
	if err != nil {
		// If we can't get channel info, use channel ID as name
		apperr.Handle(ctx, err)
		channelInfo = &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Name: originChannelID,
			},
		}
	}

	// Create the incident (with empty description for backward compatibility)
	incident, err := u.CreateIncident(ctx, &model.CreateIncidentRequest{
		Title:             title,
		Description:       "",
		CategoryID:        "unknown",
		OriginChannelID:   originChannelID,
		OriginChannelName: channelInfo.Name,
		CreatedBy:         userID,
	})
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create incident")
	}

	// Send success message to the original channel
	successBlocks := u.blockBuilder.BuildIncidentCreatedBlocks(
		incident.ChannelName.String(),
		incident.ChannelID.String(),
		incident.Title,
		incident.CategoryID,
		u.categories,
	)
	_, _, err = u.slackClient.PostMessage(
		ctx,
		originChannelID,
		slack.MsgOptionBlocks(successBlocks...),
	)
	if err != nil {
		// Log error but don't fail - message is nice to have but not critical
		apperr.Handle(ctx, err)
	}

	return incident, nil
}

// HandleCreateIncidentAction handles the complete flow when a user clicks the create incident button
func (u *Incident) HandleCreateIncidentAction(ctx context.Context, requestID, userID string) (*model.Incident, error) {
	return u.handleCreateIncidentFromRequest(ctx, requestID, userID, nil)
}

// UpdateIncidentDetails updates incident title, description, and lead
func (u *Incident) UpdateIncidentDetails(ctx context.Context, incidentID types.IncidentID, title, description string, lead types.SlackUserID) (*model.Incident, error) {
	// Validate incident ID
	if err := incidentID.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid incident ID")
	}

	// Get existing incident
	incident, err := u.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident")
	}

	// Track if any changes were made
	hasChanges := false
	var changes []string

	// Update fields if provided
	if title != "" && title != incident.Title {
		incident.Title = title
		hasChanges = true
		changes = append(changes, "title")
	}
	if description != "" && description != incident.Description {
		incident.Description = description
		hasChanges = true
		changes = append(changes, "description")
	}
	if lead != "" && lead != incident.Lead {
		incident.Lead = lead
		hasChanges = true
		changes = append(changes, "lead")
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
