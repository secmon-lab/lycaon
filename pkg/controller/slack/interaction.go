package slack

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/secmon-lab/lycaon/pkg/utils/async"
	"github.com/slack-go/slack"
)

// InteractionHandler handles Slack interactions
type InteractionHandler struct {
	incidentUC   interfaces.Incident
	slackService interfaces.SlackClient
	blockBuilder *slackSvc.BlockBuilder
}

// NewInteractionHandler creates a new interaction handler
func NewInteractionHandler(ctx context.Context, incidentUC interfaces.Incident, slackToken string) *InteractionHandler {
	return &InteractionHandler{
		incidentUC:   incidentUC,
		slackService: slackSvc.New(slackToken),
		blockBuilder: slackSvc.NewBlockBuilder(),
	}
}

// HandleInteraction handles a Slack interaction
func (h *InteractionHandler) HandleInteraction(ctx context.Context, payload []byte) error {
	var interaction slack.InteractionCallback
	if err := json.Unmarshal(payload, &interaction); err != nil {
		return goerr.Wrap(err, "failed to unmarshal interaction payload")
	}

	ctxlog.From(ctx).Info("Handling Slack interaction",
		"type", string(interaction.Type),
		"user", interaction.User.ID,
		"team", interaction.Team.ID,
	)

	// Handle different interaction types
	switch interaction.Type {
	case slack.InteractionTypeBlockActions:
		return h.handleBlockActions(ctx, &interaction)

	case slack.InteractionTypeShortcut:
		return h.handleShortcut(ctx, &interaction)

	case slack.InteractionTypeViewSubmission:
		return h.handleViewSubmission(ctx, &interaction)

	case slack.InteractionTypeViewClosed:
		ctxlog.From(ctx).Debug("View closed",
			"viewID", interaction.View.ID,
		)
		return nil

	default:
		ctxlog.From(ctx).Debug("Unhandled interaction type",
			"type", string(interaction.Type),
		)
		return nil
	}
}

// handleBlockActions handles block action interactions
func (h *InteractionHandler) handleBlockActions(ctx context.Context, interaction *slack.InteractionCallback) error {
	for _, action := range interaction.ActionCallback.BlockActions {
		ctxlog.From(ctx).Info("Block action triggered",
			"actionID", action.ActionID,
			"blockID", action.BlockID,
			"value", action.Value,
			"type", string(action.Type),
		)

		// Handle specific actions based on ActionID
		switch action.ActionID {
		case "create_incident":
			ctxlog.From(ctx).Info("Create incident action triggered",
				"user", interaction.User.ID,
				"channel", interaction.Channel.ID,
				"requestID", action.Value,
			)

			requestID := action.Value
			if requestID == "" {
				ctxlog.From(ctx).Error("Empty request ID in action value")
				return goerr.New("empty request ID")
			}

			// Send immediate acknowledgment to Slack
			// This prevents the "This interaction failed" error
			ctxlog.From(ctx).Info("Acknowledging incident creation request")

			// Process incident creation asynchronously with preserved context
			backgroundCtx := async.NewBackgroundContext(ctx)
			async.Dispatch(backgroundCtx, func(asyncCtx context.Context) error {
				// Call the single usecase method that handles everything including error messaging
				h.incidentUC.HandleCreateIncidentActionAsync(
					asyncCtx,
					requestID,
					interaction.User.ID,
					interaction.Channel.ID,
				)
				return nil
			})

			// Return immediately to acknowledge the interaction
			// The actual processing happens in the background

		case "edit_incident":
			ctxlog.From(ctx).Info("Edit incident action triggered",
				"user", interaction.User.ID,
				"channel", interaction.Channel.ID,
				"requestID", action.Value,
				"triggerID", interaction.TriggerID,
			)

			requestID := action.Value
			if requestID == "" {
				ctxlog.From(ctx).Error("Empty request ID in action value")
				return goerr.New("empty request ID")
			}

			// Call the single usecase method that handles the entire edit flow
			err := h.incidentUC.HandleEditIncidentAction(ctx, requestID, interaction.User.ID, interaction.TriggerID)
			if err != nil {
				ctxlog.From(ctx).Error("Failed to handle edit incident action",
					"error", err,
					"requestID", requestID,
					"user", interaction.User.ID,
				)
				return goerr.Wrap(err, "failed to handle edit incident action")
			}

		case "acknowledge":
			ctxlog.From(ctx).Info("Acknowledge action triggered")
			// TODO: Implement acknowledge logic

		case "resolve":
			ctxlog.From(ctx).Info("Resolve action triggered")
			// TODO: Implement resolve logic

		default:
			ctxlog.From(ctx).Debug("Unknown action",
				"actionID", action.ActionID,
			)
		}
	}

	return nil
}

// handleShortcut handles shortcut interactions
func (h *InteractionHandler) handleShortcut(ctx context.Context, interaction *slack.InteractionCallback) error {
	ctxlog.From(ctx).Info("Shortcut triggered",
		"callbackID", interaction.CallbackID,
		"triggerID", interaction.TriggerID,
	)

	// Handle specific shortcuts based on CallbackID
	switch interaction.CallbackID {
	case "create_incident_shortcut":
		ctxlog.From(ctx).Info("Create incident shortcut triggered")
		// TODO: Open incident creation modal

	default:
		ctxlog.From(ctx).Debug("Unknown shortcut",
			"callbackID", interaction.CallbackID,
		)
	}

	return nil
}

// handleViewSubmission handles view submission interactions
func (h *InteractionHandler) handleViewSubmission(ctx context.Context, interaction *slack.InteractionCallback) error {
	ctxlog.From(ctx).Info("View submitted",
		"viewID", interaction.View.ID,
		"callbackID", interaction.View.CallbackID,
	)

	// Handle specific view submissions based on CallbackID
	switch interaction.View.CallbackID {
	case "incident_creation_modal":
		ctxlog.From(ctx).Info("Incident creation modal submitted",
			"user", interaction.User.ID,
			"team", interaction.Team.ID,
		)

		// Extract request ID from private metadata
		requestID := interaction.View.PrivateMetadata
		if requestID == "" {
			ctxlog.From(ctx).Error("Empty request ID in private metadata")
			return goerr.New("empty request ID")
		}

		// Extract title from the modal (required)
		var titleValue string
		if titleBlock, ok := interaction.View.State.Values["title_block"]; ok {
			if titleInput, ok := titleBlock["title_input"]; ok {
				titleValue = titleInput.Value
			}
		}

		// Extract description from the modal (optional)
		var descriptionValue string
		if descBlock, ok := interaction.View.State.Values["description_block"]; ok {
			if descInput, ok := descBlock["description_input"]; ok {
				descriptionValue = descInput.Value
			}
		}

		// Validate required title field
		if titleValue == "" {
			ctxlog.From(ctx).Error("Title is required for incident creation")
			return goerr.New("incident title is required")
		}

		ctxlog.From(ctx).Info("Processing incident creation with details",
			"requestID", requestID,
			"title", titleValue,
			"hasDescription", descriptionValue != "",
		)

		// Process incident creation asynchronously
		backgroundCtx := async.NewBackgroundContext(ctx)
		async.Dispatch(backgroundCtx, func(asyncCtx context.Context) error {
			// Call the incident creation with the edited details
			incident, err := h.incidentUC.HandleCreateIncidentWithDetails(
				asyncCtx,
				requestID,
				titleValue,
				descriptionValue,
				interaction.User.ID,
			)
			if err != nil {
				ctxlog.From(asyncCtx).Error("Failed to create incident from modal",
					"error", err,
					"user", interaction.User.ID,
					"requestID", requestID,
				)
				// Note: We can't easily send error messages from view submission
				// The modal will close, but the error is logged
				return goerr.Wrap(err, "failed to create incident from modal")
			}

			ctxlog.From(asyncCtx).Info("Incident created successfully from modal",
				"incidentID", incident.ID,
				"channelName", incident.ChannelName,
				"createdBy", interaction.User.ID,
			)
			return nil
		})

	default:
		ctxlog.From(ctx).Debug("Unknown view submission",
			"callbackID", interaction.View.CallbackID,
		)
	}

	return nil
}
