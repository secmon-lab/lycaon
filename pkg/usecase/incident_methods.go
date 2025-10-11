package usecase

import (
	"context"
	"errors"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/slack-go/slack"
)

// HandleCreateIncidentWithDetails handles the create incident with edited details from modal
func (u *Incident) HandleCreateIncidentWithDetails(ctx context.Context, requestID, title, description, categoryID, severityID, userID string) (*model.Incident, error) {
	return u.handleCreateIncidentFromRequest(ctx, requestID, userID, &incidentDetails{
		title:       title,
		description: description,
		categoryID:  categoryID,
		severityID:  severityID,
		assetIDs:    []types.AssetID{},
	})
}

// HandleCreateIncidentWithDetailsAndAssets handles the create incident with edited details and assets from modal
func (u *Incident) HandleCreateIncidentWithDetailsAndAssets(ctx context.Context, requestID, title, description, categoryID, severityID string, assetIDs []types.AssetID, isPrivate bool, userID string) (*model.Incident, error) {
	return u.handleCreateIncidentFromRequest(ctx, requestID, userID, &incidentDetails{
		title:       title,
		description: description,
		categoryID:  categoryID,
		severityID:  severityID,
		assetIDs:    assetIDs,
		private:     isPrivate,
	})
}

// incidentDetails holds the details for incident creation
type incidentDetails struct {
	title       string
	description string
	categoryID  string
	severityID  string
	assetIDs    []types.AssetID
	private     bool
}

// handleCreateIncidentFromRequest is the common implementation for creating incidents from requests
func (u *Incident) handleCreateIncidentFromRequest(ctx context.Context, requestID, userID string, details *incidentDetails) (*model.Incident, error) {
	if requestID == "" {
		return nil, goerr.New("request ID is empty")
	}
	if userID == "" {
		return nil, goerr.New("user ID is empty")
	}

	// Retrieve the incident request
	request, err := u.repo.GetIncidentRequest(ctx, types.IncidentRequestID(requestID))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident request")
	}

	var incident *model.Incident
	var title string

	if details != nil {
		// Create incident with detailed information (from modal)
		title = details.title

		// Get channel info from Slack
		channelInfo, err := u.slackClient.GetConversationInfo(ctx, request.ChannelID.String(), false)
		if err != nil {
			// If we can't get channel info, use channel ID as name
			ctxlog.From(ctx).Warn("Failed to get conversation info, using channel ID as name",
				"error", err,
				"channelID", request.ChannelID,
			)
			channelInfo = &slack.Channel{
				GroupConversation: slack.GroupConversation{
					Name: request.ChannelID.String(),
				},
			}
		}

		// Create the incident with the provided details
		incident, err = u.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             details.title,
			Description:       details.description,
			CategoryID:        details.categoryID,
			SeverityID:        details.severityID,
			AssetIDs:          details.assetIDs,
			OriginChannelID:   request.ChannelID.String(),
			OriginChannelName: channelInfo.Name,
			CreatedBy:         userID,
			InitialTriage:     false, // TODO: Get from modal checkbox
			Private:           details.private,
		})
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create incident")
		}

		// Send notification to the original channel thread with also_send_to_channel
		if err := u.slackSvc.PostIncidentCreatedNotification(
			ctx,
			request.ChannelID,
			request.MessageTS,
			channelInfo.Name,
			incident.ChannelID,
			title,
			details.categoryID,
			details.severityID,
		); err != nil {
			// Log error but don't fail - the incident was created successfully
			ctxlog.From(ctx).Warn("Failed to post incident creation notification",
				"error", err,
				"channelID", request.ChannelID,
				"incidentID", incident.ID,
			)
		}
	} else {
		// Create incident with basic information (from button)
		title = request.Title

		// Get channel info from Slack
		channelInfo, err := u.slackClient.GetConversationInfo(ctx, request.ChannelID.String(), false)
		if err != nil {
			// If we can't get channel info, use channel ID as name
			ctxlog.From(ctx).Warn("Failed to get conversation info, using channel ID as name",
				"error", err,
				"channelID", request.ChannelID,
			)
			channelInfo = &slack.Channel{
				GroupConversation: slack.GroupConversation{
					Name: request.ChannelID.String(),
				},
			}
		}

		// Create the incident
		incident, err = u.CreateIncident(ctx, &model.CreateIncidentRequest{
			Title:             request.Title,
			Description:       request.Description,
			CategoryID:        request.CategoryID,
			SeverityID:        request.SeverityID,
			OriginChannelID:   request.ChannelID.String(),
			OriginChannelName: channelInfo.Name,
			CreatedBy:         userID,
		})
		if err != nil {
			return nil, goerr.Wrap(err, "failed to create incident")
		}

		// Send notification to the original channel thread with also_send_to_channel
		if err := u.slackSvc.PostIncidentCreatedNotification(
			ctx,
			request.ChannelID,
			request.MessageTS,
			channelInfo.Name,
			incident.ChannelID,
			incident.Title,
			incident.CategoryID,
			incident.SeverityID.String(),
		); err != nil {
			// Log error but don't fail - the incident was created successfully
			ctxlog.From(ctx).Warn("Failed to post incident creation notification",
				"error", err,
				"channelID", request.ChannelID,
				"incidentID", incident.ID,
			)
		}
	}

	// Update the original message to show incident was declared
	u.updateOriginalMessageToDeclared(ctx, request, title)

	// Clean up the request after successful creation
	if err := u.repo.DeleteIncidentRequest(ctx, types.IncidentRequestID(requestID)); err != nil {
		// Log error but don't fail - the incident was created successfully
		ctxlog.From(ctx).Warn("Failed to delete incident request after creation",
			"error", err,
			"requestID", requestID,
		)
	}

	return incident, nil
}

// updateOriginalMessageToDeclared updates the bot's prompt message to show incident was declared
func (u *Incident) updateOriginalMessageToDeclared(ctx context.Context, request *model.IncidentRequest, title string) {
	// Update the bot's message, not the original user message
	messageToUpdate := request.BotMessageTS
	if messageToUpdate == "" {
		// Fallback to original message timestamp if bot message timestamp is not available
		messageToUpdate = request.MessageTS
		ctxlog.From(ctx).Warn("Bot message timestamp not available, falling back to original message",
			"channelID", request.ChannelID,
			"originalMessageTS", request.MessageTS,
		)
	}

	ctxlog.From(ctx).Info("Updating bot message to show incident declared",
		"channelID", request.ChannelID,
		"botMessageTS", messageToUpdate,
		"originalMessageTS", request.MessageTS,
		"title", title,
	)

	if err := u.slackSvc.UpdateOriginalPromptToUsed(ctx, request.ChannelID, messageToUpdate, title); err != nil {
		ctxlog.From(ctx).Warn("Failed to update bot message",
			"error", err,
			"channelID", request.ChannelID,
			"messageTS", messageToUpdate,
			"originalMessageTS", request.MessageTS,
		)
	} else {
		ctxlog.From(ctx).Info("Successfully updated bot message",
			"channelID", request.ChannelID,
			"messageTS", messageToUpdate,
		)
	}
}

// GetIncidentRequest retrieves an incident request by ID
func (u *Incident) GetIncidentRequest(ctx context.Context, requestID string) (*model.IncidentRequest, error) {
	if requestID == "" {
		return nil, goerr.New("request ID is empty")
	}

	request, err := u.repo.GetIncidentRequest(ctx, types.IncidentRequestID(requestID))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get incident request from repository")
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
		if errors.Is(err, model.ErrIncidentRequestNotFound) {
			return goerr.Wrap(err, "failed to open edit dialog - request not found")
		}
		return goerr.Wrap(err, "failed to retrieve incident request for editing")
	}

	// Build the edit modal with the existing title and description pre-filled
	ctxlog.From(ctx).Debug("Building edit modal",
		"requestID", requestID,
		"categoriesCount", len(u.modelConfig.Categories),
		"currentCategoryID", request.CategoryID,
		"currentSeverityID", request.SeverityID,
		"assetIDsCount", len(request.AssetIDs),
	)

	// Open the modal using slack service
	err = u.slackSvc.OpenIncidentEditModal(ctx, triggerID, requestID, request.Title, request.Description, request.CategoryID, request.SeverityID, request.AssetIDs)
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
		// Check if error is due to not found request
		if errors.Is(err, model.ErrIncidentRequestNotFound) {
			errorMessage = "Failed to create incident. The request was not found."
		}

		// Send error message using slack service
		if msgErr := u.slackSvc.PostErrorMessage(ctx, types.ChannelID(channelID), errorMessage); msgErr != nil {
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
