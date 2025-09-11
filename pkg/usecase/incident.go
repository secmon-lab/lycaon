package usecase

import (
	"context"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/slack-go/slack"
)

// Incident implements Incident interface
type Incident struct {
	repo         interfaces.Repository
	slackClient  interfaces.SlackClient
	blockBuilder *slackSvc.BlockBuilder
	categories   *model.CategoriesConfig
}

// NewIncident creates a new Incident instance with a custom SlackClient
func NewIncident(repo interfaces.Repository, slackClient interfaces.SlackClient, categories *model.CategoriesConfig) *Incident {
	return &Incident{
		repo:         repo,
		slackClient:  slackClient,
		blockBuilder: slackSvc.NewBlockBuilder(),
		categories:   categories,
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
	incident, err := model.NewIncident(incidentNumber, req.Title, req.Description, req.CategoryID, types.ChannelID(req.OriginChannelID), types.ChannelName(req.OriginChannelName), types.SlackUserID(req.CreatedBy))
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
			ctxlog.From(ctx).Warn("Failed to set channel purpose",
				"error", err,
				"channelID", channel.ID,
				"title", req.Title,
			)
		}
	}

	// Set channel ID
	incident.ChannelID = types.ChannelID(channel.ID)

	// Invite creator to the channel
	_, err = u.slackClient.InviteUsersToConversation(ctx, channel.ID, req.CreatedBy)
	if err != nil {
		// Log error but don't fail - user might already be in channel or invitation might fail for other reasons
		// The incident is still created successfully
		ctxlog.From(ctx).Warn("Failed to invite user to incident channel",
			"error", err,
			"channelID", channel.ID,
			"userID", req.CreatedBy,
		)
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
		ctxlog.From(ctx).Warn("Failed to post welcome message to incident channel",
			"error", err,
			"channelID", channel.ID,
			"incidentNumber", incidentNumber,
		)
	}

	// Save incident to repository
	if err := u.repo.PutIncident(ctx, incident); err != nil {
		return nil, goerr.Wrap(err, "failed to save incident")
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
		ctxlog.From(ctx).Warn("Failed to get conversation info, using channel ID as name",
			"error", err,
			"channelID", originChannelID,
		)
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
		// The incident is already created successfully
		ctxlog.From(ctx).Warn("Failed to post success message to original channel",
			"error", err,
			"channelID", originChannelID,
			"incidentChannelID", incident.ChannelID,
		)
	}

	return incident, nil
}

// HandleCreateIncidentAction handles the complete flow when a user clicks the create incident button
func (u *Incident) HandleCreateIncidentAction(ctx context.Context, requestID, userID string) (*model.Incident, error) {
	return u.handleCreateIncidentFromRequest(ctx, requestID, userID, nil)
}
