package slack

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/slack-go/slack"
)

// InteractionHandler handles Slack interactions
type InteractionHandler struct {
}

// NewInteractionHandler creates a new interaction handler
func NewInteractionHandler(ctx context.Context) *InteractionHandler {
	return &InteractionHandler{}
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
			ctxlog.From(ctx).Info("Create incident action triggered")
			// TODO: Implement incident creation logic

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
