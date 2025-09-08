package usecase

import (
	"context"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/slack-go/slack"
)

// HandleCreateIncidentWithDetails handles the create incident with edited details from modal
func (u *Incident) HandleCreateIncidentWithDetails(ctx context.Context, requestID, title, description, userID string) (*model.Incident, error) {
	if requestID == "" {
		return nil, goerr.New("request ID is empty")
	}
	if userID == "" {
		return nil, goerr.New("user ID is empty")
	}

	// Retrieve the incident request to get the channel ID
	request, err := u.repo.GetIncidentRequest(ctx, requestID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident request")
	}

	// Get channel info from Slack
	channelInfo, err := u.slackClient.GetConversationInfo(ctx, request.ChannelID, false)
	if err != nil {
		// If we can't get channel info, use channel ID as name
		ctxlog.From(ctx).Warn("Failed to get conversation info, using channel ID as name",
			"error", err,
			"channelID", request.ChannelID,
		)
		channelInfo = &slack.Channel{
			GroupConversation: slack.GroupConversation{
				Name: request.ChannelID,
			},
		}
	}

	// Create the incident with the provided title and description
	incident, err := u.CreateIncident(ctx, title, description, request.ChannelID, channelInfo.Name, userID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create incident")
	}

	// Send notification to the original channel
	notificationBlocks := u.blockBuilder.BuildIncidentCreatedBlocks(channelInfo.Name, incident.ChannelID, title)
	if _, _, err := u.slackClient.PostMessage(
		ctx,
		request.ChannelID,
		slack.MsgOptionBlocks(notificationBlocks...),
	); err != nil {
		// Log error but don't fail - the incident was created successfully
		ctxlog.From(ctx).Warn("Failed to post incident creation notification",
			"error", err,
			"channelID", request.ChannelID,
			"incidentID", incident.ID,
		)
	}

	// Clean up the request after successful creation
	if err := u.repo.DeleteIncidentRequest(ctx, requestID); err != nil {
		// Log error but don't fail - the incident was created successfully
		ctxlog.From(ctx).Warn("Failed to delete incident request after creation",
			"error", err,
			"requestID", requestID,
		)
	}

	return incident, nil
}

// GetIncidentRequest retrieves an incident request by ID
func (u *Incident) GetIncidentRequest(ctx context.Context, requestID string) (*model.IncidentRequest, error) {
	if requestID == "" {
		return nil, goerr.New("request ID is empty")
	}

	request, err := u.repo.GetIncidentRequest(ctx, requestID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident request from repository")
	}

	// Check if request has expired
	if request.IsExpired() {
		return nil, goerr.Wrap(model.ErrIncidentRequestExpired, "request already expired")
	}

	return request, nil
}