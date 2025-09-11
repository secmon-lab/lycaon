package slack

import (
	"context"
	"encoding/json"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/utils/async"
	"github.com/slack-go/slack"
)

// InteractionHandler handles Slack interactions
// Responsibility: Parse Slack messages, extract necessary information, 
// prepare data for usecase, and control async dispatch
type InteractionHandler struct {
	slackUC interfaces.SlackInteraction
}

// NewInteractionHandler creates a new interaction handler
func NewInteractionHandler(ctx context.Context, slackUC interfaces.SlackInteraction) *InteractionHandler {
	return &InteractionHandler{
		slackUC: slackUC,
	}
}

// HandleInteraction handles a Slack interaction
// Controller responsibility: Parse request, extract data, prepare usecase call, handle async dispatch
func (h *InteractionHandler) HandleInteraction(ctx context.Context, payload []byte) error {
	logger := ctxlog.From(ctx)

	// Parse Slack interaction payload
	var interaction slack.InteractionCallback
	if err := json.Unmarshal(payload, &interaction); err != nil {
		return goerr.Wrap(err, "failed to unmarshal interaction payload")
	}

	logger.Info("Handling Slack interaction",
		"type", string(interaction.Type),
		"user", interaction.User.ID,
		"team", interaction.Team.ID,
	)

	// Prepare interaction data for usecase
	interactionData := &interfaces.SlackInteractionData{
		Type:      string(interaction.Type),
		UserID:    interaction.User.ID,
		TeamID:    interaction.Team.ID,
		ChannelID: interaction.Channel.ID,
		TriggerID: interaction.TriggerID,
		RawPayload: payload,
	}

	// All usecase calls must be async dispatched to return 200 immediately
	switch interaction.Type {
	case slack.InteractionTypeBlockActions:
		// Validate critical fields before async dispatch
		for _, action := range interaction.ActionCallback.BlockActions {
			if action.ActionID == "create_incident" && action.Value == "" {
				return goerr.New("empty request ID")
			}
		}
		return h.handleAsync(ctx, interactionData, h.slackUC.HandleBlockActions)

	case slack.InteractionTypeViewSubmission:
		// Validate critical fields for incident creation modal
		if interaction.View.CallbackID == "incident_creation_modal" {
			// Check if title field exists and is not empty
			if titleBlock, ok := interaction.View.State.Values["title_block"]; ok {
				// Check for either "title" or "title_input" keys
				titleInput, hasTitle := titleBlock["title"]
				if !hasTitle {
					titleInput, hasTitle = titleBlock["title_input"]
				}
				if !hasTitle || titleInput.Value == "" {
					return goerr.New("incident title is required")
				}
			} else {
				return goerr.New("incident title is required") 
			}
		}
		return h.handleAsync(ctx, interactionData, h.slackUC.HandleViewSubmission)

	case slack.InteractionTypeShortcut:
		return h.handleAsync(ctx, interactionData, h.slackUC.HandleShortcut)

	case slack.InteractionTypeViewClosed:
		// No processing needed for view closed
		logger.Debug("View closed", "viewID", interaction.View.ID)
		return nil

	default:
		logger.Debug("Unhandled interaction type", "type", string(interaction.Type))
		return nil
	}
}

// handleAsync handles all usecase calls asynchronously to return 200 immediately
// Controller responsibility: Dispatch usecase processing and return immediate response
func (h *InteractionHandler) handleAsync(ctx context.Context, data *interfaces.SlackInteractionData, usecaseHandler func(context.Context, *interfaces.SlackInteractionData) error) error {
	logger := ctxlog.From(ctx)
	
	// Create background context for async processing
	backgroundCtx := async.NewBackgroundContext(ctx)
	
	// Dispatch usecase processing asynchronously
	async.Dispatch(backgroundCtx, func(asyncCtx context.Context) error {
		if err := usecaseHandler(asyncCtx, data); err != nil {
			logger.Error("UseCase processing failed",
				"error", err,
				"interactionType", data.Type,
				"user", data.UserID,
			)
			// Log error but don't propagate - async processing
			return nil
		}
		
		logger.Debug("UseCase processing completed",
			"interactionType", data.Type,
			"user", data.UserID,
		)
		return nil
	})
	
	// Return immediately to send 200 response to Slack
	logger.Debug("Interaction dispatched for async processing",
		"interactionType", data.Type,
		"user", data.UserID,
	)
	return nil
}

