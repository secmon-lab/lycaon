package usecase

import (
	"context"
	"errors"

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

// HandleEditIncidentAction handles the edit incident button click action
func (u *Incident) HandleEditIncidentAction(ctx context.Context, requestID, userID, triggerID string) error {
	// Get the incident request to retrieve the title
	request, err := u.GetIncidentRequest(ctx, requestID)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to get incident request for edit",
			"error", err,
			"requestID", requestID,
			"userID", userID,
		)
		
		// Send error message to user (we don't have channel info on error, so just return the error)
		if errors.Is(err, model.ErrIncidentRequestNotFound) || errors.Is(err, model.ErrIncidentRequestExpired) {
			return goerr.Wrap(err, "failed to open edit dialog - request may have expired")
		}
		return goerr.Wrap(err, "failed to retrieve incident request for editing")
	}

	// Build the edit modal with the existing title pre-filled
	modal := u.blockBuilder.BuildIncidentEditModal(requestID, request.Title)
	
	// Open the modal
	_, err = u.slackClient.OpenView(ctx, triggerID, modal)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to open edit modal",
			"error", err,
			"requestID", requestID,
			"userID", userID,
		)
		return goerr.Wrap(err, "failed to open incident edit modal")
	}

	ctxlog.From(ctx).Info("Edit modal opened successfully",
		"requestID", requestID,
		"userID", userID,
		"title", request.Title,
	)
	
	return nil
}

// HandleCreateIncidentActionAsync handles the create incident button click with async processing and error messaging
func (u *Incident) HandleCreateIncidentActionAsync(ctx context.Context, requestID, userID, channelID string) {
	// Process incident creation
	incident, err := u.HandleCreateIncidentAction(ctx, requestID, userID)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to handle incident creation",
			"error", err,
			"user", userID,
			"requestID", requestID,
		)

		// Send error message to user
		errorMessage := "Failed to create incident. Please try again."
		// Check if error is due to expired or not found request
		if errors.Is(err, model.ErrIncidentRequestNotFound) || errors.Is(err, model.ErrIncidentRequestExpired) {
			errorMessage = "Failed to create incident. The request may have expired."
		}
		
		// Build error blocks and send message
		errorBlocks := u.blockBuilder.BuildErrorBlocks(errorMessage)
		if _, _, msgErr := u.slackClient.PostMessage(
			ctx,
			channelID,
			slack.MsgOptionBlocks(errorBlocks...),
		); msgErr != nil {
			ctxlog.From(ctx).Error("Failed to post error message", "error", msgErr)
		}
		return
	}

	ctxlog.From(ctx).Info("Incident created successfully",
		"incidentID", incident.ID,
		"channelName", incident.ChannelName,
		"createdBy", userID,
	)
}

