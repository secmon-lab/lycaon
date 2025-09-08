package slack

import (
	"context"
	"encoding/json"
	"strings"

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
	slackService *slackSvc.Service
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
				// Call the single usecase method that handles everything
				incident, err := h.incidentUC.HandleCreateIncidentAction(
					asyncCtx,
					requestID,
					interaction.User.ID, // Use the actual user who clicked the button
				)
				if err != nil {
					ctxlog.From(asyncCtx).Error("Failed to handle incident creation",
						"error", err,
						"user", interaction.User.ID,
						"requestID", requestID,
					)

					// Send error message
					errorMessage := "Failed to create incident. Please try again."
					// Check if error is due to expired or not found request
					if strings.Contains(err.Error(), "incident request not found") || strings.Contains(err.Error(), "incident request has expired") {
						errorMessage = "Failed to create incident. The request may have expired."
					}
					errorBlocks := h.blockBuilder.BuildErrorBlocks(errorMessage)
					h.slackService.PostEphemeral(
						asyncCtx,
						interaction.Channel.ID,
						interaction.User.ID,
						slack.MsgOptionBlocks(errorBlocks...),
					)
					return goerr.Wrap(err, "failed to handle incident creation")
				}

				ctxlog.From(asyncCtx).Info("Incident created successfully",
					"incidentID", incident.ID,
					"channelName", incident.ChannelName,
					"createdBy", interaction.User.ID,
				)
				return nil
			})

			// Return immediately to acknowledge the interaction
			// The actual processing happens in the background

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
		ctxlog.From(ctx).Info("Incident creation modal submitted")
		// TODO: Process incident creation form data

	default:
		ctxlog.From(ctx).Debug("Unknown view submission",
			"callbackID", interaction.View.CallbackID,
		)
	}

	return nil
}
