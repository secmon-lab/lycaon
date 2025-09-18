package usecase

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	slackblocks "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/secmon-lab/lycaon/pkg/utils/async"
	"github.com/slack-go/slack"
)

// SlackInteraction implements SlackInteraction interface
type SlackInteraction struct {
	incidentUC   interfaces.Incident
	taskUC       interfaces.Task
	statusUC     interfaces.StatusUseCase
	slackClient  interfaces.SlackClient
	blockBuilder *slackblocks.BlockBuilder
}

// NewSlackInteraction creates a new SlackInteraction instance
func NewSlackInteraction(incidentUC interfaces.Incident, taskUC interfaces.Task, statusUC interfaces.StatusUseCase, slackClient interfaces.SlackClient) *SlackInteraction {
	return &SlackInteraction{
		incidentUC:   incidentUC,
		taskUC:       taskUC,
		statusUC:     statusUC,
		slackClient:  slackClient,
		blockBuilder: slackblocks.NewBlockBuilder(),
	}
}

// HandleBlockActions handles block action interactions (buttons)
// UseCase responsibility: Execute business logic for button clicks
func (s *SlackInteraction) HandleBlockActions(ctx context.Context, data *interfaces.SlackInteractionData) error {
	var interaction slack.InteractionCallback
	if err := json.Unmarshal(data.RawPayload, &interaction); err != nil {
		return goerr.Wrap(err, "failed to unmarshal interaction payload")
	}

	return s.handleBlockActions(ctx, &interaction)
}

// HandleViewSubmission handles modal/view submission interactions
// UseCase responsibility: Execute business logic for modal submissions
func (s *SlackInteraction) HandleViewSubmission(ctx context.Context, data *interfaces.SlackInteractionData) error {
	var interaction slack.InteractionCallback
	if err := json.Unmarshal(data.RawPayload, &interaction); err != nil {
		return goerr.Wrap(err, "failed to unmarshal interaction payload")
	}

	return s.handleViewSubmission(ctx, &interaction)
}

// HandleShortcut handles shortcut interactions
// UseCase responsibility: Execute business logic for shortcuts
func (s *SlackInteraction) HandleShortcut(ctx context.Context, data *interfaces.SlackInteractionData) error {
	var interaction slack.InteractionCallback
	if err := json.Unmarshal(data.RawPayload, &interaction); err != nil {
		return goerr.Wrap(err, "failed to unmarshal interaction payload")
	}

	return s.handleShortcut(ctx, &interaction)
}

// handleBlockActions handles block action interactions
func (s *SlackInteraction) handleBlockActions(ctx context.Context, interaction *slack.InteractionCallback) error {
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
			return s.handleCreateIncidentAction(ctx, interaction, action)

		case "edit_incident":
			return s.handleEditIncidentAction(ctx, interaction, action)

		case "edit_incident_details":
			return s.handleEditIncidentDetailsAction(ctx, interaction, action)

		case "edit_incident_status":
			return s.handleEditIncidentStatusAction(ctx, interaction, action)

		case "acknowledge":
			ctxlog.From(ctx).Info("Acknowledge action triggered")
			// TODO: Implement acknowledge logic

		case "resolve":
			ctxlog.From(ctx).Info("Resolve action triggered")
			// TODO: Implement resolve logic

		default:
			// Check if it's a task action
			if strings.HasPrefix(action.ActionID, "task_") {
				return s.handleTaskAction(ctx, interaction, action)
			}

			ctxlog.From(ctx).Debug("Unknown action",
				"actionID", action.ActionID,
			)
		}
	}

	return nil
}

// handleCreateIncidentAction handles create incident button action
func (s *SlackInteraction) handleCreateIncidentAction(ctx context.Context, interaction *slack.InteractionCallback, action *slack.BlockAction) error {
	ctxlog.From(ctx).Info("Create incident action triggered",
		"user", interaction.User.ID,
		"channel", interaction.Channel.ID,
		"requestID", action.Value,
	)

	requestID := action.Value

	// Send immediate acknowledgment to Slack
	ctxlog.From(ctx).Info("Acknowledging incident creation request")

	// Process incident creation asynchronously with preserved context
	backgroundCtx := async.NewBackgroundContext(ctx)
	async.Dispatch(backgroundCtx, func(asyncCtx context.Context) error {
		// Call the single usecase method that handles everything including error messaging
		s.incidentUC.HandleCreateIncidentActionAsync(
			asyncCtx,
			requestID,
			interaction.User.ID,
			interaction.Channel.ID,
		)
		return nil
	})

	return nil
}

// handleEditIncidentAction handles edit incident button action
func (s *SlackInteraction) handleEditIncidentAction(ctx context.Context, interaction *slack.InteractionCallback, action *slack.BlockAction) error {
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
	err := s.incidentUC.HandleEditIncidentAction(ctx, requestID, interaction.User.ID, interaction.TriggerID)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to handle edit incident action",
			"error", err,
			"requestID", requestID,
			"user", interaction.User.ID,
		)
		return goerr.Wrap(err, "failed to handle edit incident action")
	}

	return nil
}

// handleShortcut handles shortcut interactions
func (s *SlackInteraction) handleShortcut(ctx context.Context, interaction *slack.InteractionCallback) error {
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
func (s *SlackInteraction) handleViewSubmission(ctx context.Context, interaction *slack.InteractionCallback) error {
	ctxlog.From(ctx).Info("View submitted",
		"viewID", interaction.View.ID,
		"callbackID", interaction.View.CallbackID,
	)

	// Handle specific view submissions based on CallbackID
	switch interaction.View.CallbackID {
	case "incident_creation_modal", "incident_edit_modal":
		return s.handleIncidentModalSubmission(ctx, interaction)

	case "status_change_modal":
		return s.handleStatusChangeModalSubmission(ctx, interaction)

	case "edit_incident_details_modal":
		return s.handleEditIncidentDetailsModalSubmission(ctx, interaction)

	default:
		// Check if it's a task edit modal submission
		if strings.HasPrefix(interaction.View.CallbackID, "task_edit_submit:") {
			return s.handleTaskEditSubmission(ctx, interaction)
		}

		ctxlog.From(ctx).Debug("Unknown view submission",
			"callbackID", interaction.View.CallbackID,
		)
	}

	return nil
}

// handleIncidentModalSubmission handles incident modal submission
func (s *SlackInteraction) handleIncidentModalSubmission(ctx context.Context, interaction *slack.InteractionCallback) error {
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

	// Extract category from the modal (required)
	var categoryValue string
	if categoryBlock, ok := interaction.View.State.Values["category_block"]; ok {
		if categorySelect, ok := categoryBlock["category_select"]; ok {
			if categorySelect.SelectedOption.Value != "" {
				categoryValue = categorySelect.SelectedOption.Value
			}
		}
	}

	// Validate required fields
	if titleValue == "" {
		ctxlog.From(ctx).Error("Title is required for incident creation")
		return goerr.New("incident title is required")
	}
	if categoryValue == "" {
		ctxlog.From(ctx).Error("Category is required for incident creation")
		return goerr.New("incident category is required")
	}

	ctxlog.From(ctx).Info("Processing incident creation with details",
		"requestID", requestID,
		"title", titleValue,
		"hasDescription", descriptionValue != "",
		"category", categoryValue,
	)

	// Process incident creation asynchronously
	backgroundCtx := async.NewBackgroundContext(ctx)
	async.Dispatch(backgroundCtx, func(asyncCtx context.Context) error {
		// Call the incident creation with the edited details
		incident, err := s.incidentUC.HandleCreateIncidentWithDetails(
			asyncCtx,
			requestID,
			titleValue,
			descriptionValue,
			categoryValue,
			interaction.User.ID,
		)
		if err != nil {
			ctxlog.From(asyncCtx).Error("Failed to create incident from modal",
				"error", err,
				"user", interaction.User.ID,
				"requestID", requestID,
			)
			return goerr.Wrap(err, "failed to create incident from modal")
		}

		ctxlog.From(asyncCtx).Info("Incident created successfully from modal",
			"incidentID", incident.ID,
			"channelName", incident.ChannelName,
			"createdBy", interaction.User.ID,
		)
		return nil
	})

	return nil
}

// handleTaskAction handles task-related button actions
func (s *SlackInteraction) handleTaskAction(ctx context.Context, interaction *slack.InteractionCallback, action *slack.BlockAction) error {
	logger := ctxlog.From(ctx)

	// Parse action ID to determine task action type and task ID
	if strings.HasPrefix(action.ActionID, "task_complete_") {
		taskID := types.TaskID(strings.TrimPrefix(action.ActionID, "task_complete_"))
		return s.handleTaskComplete(ctx, interaction, taskID)
	} else if strings.HasPrefix(action.ActionID, "task_uncomplete_") {
		taskID := types.TaskID(strings.TrimPrefix(action.ActionID, "task_uncomplete_"))
		return s.handleTaskUncomplete(ctx, interaction, taskID)
	} else if strings.HasPrefix(action.ActionID, "task_edit_") {
		taskID := types.TaskID(strings.TrimPrefix(action.ActionID, "task_edit_"))
		return s.handleTaskEdit(ctx, interaction, taskID)
	}

	logger.Debug("Unknown task action", "actionID", action.ActionID)
	return nil
}

// getIncidentIDByChannel gets incident ID from channel ID for efficient task operations
func (s *SlackInteraction) getIncidentIDByChannel(ctx context.Context, channelID string) (types.IncidentID, error) {
	incident, err := s.incidentUC.GetIncidentByChannelID(ctx, types.ChannelID(channelID))
	if err != nil {
		return 0, goerr.Wrap(err, "failed to get incident by channel",
			goerr.V("channelID", channelID))
	}
	return incident.ID, nil
}

// handleTaskComplete handles task completion
func (s *SlackInteraction) handleTaskComplete(ctx context.Context, interaction *slack.InteractionCallback, taskID types.TaskID) error {
	logger := ctxlog.From(ctx)

	// Get incident ID from channel for efficient task lookup
	incidentID, err := s.getIncidentIDByChannel(ctx, interaction.Channel.ID)
	if err != nil {
		logger.Error("Failed to get incident for task completion", "error", err, "taskID", taskID, "channelID", interaction.Channel.ID)
		return goerr.Wrap(err, "failed to get incident for task completion")
	}

	// Complete the task efficiently
	task, err := s.taskUC.CompleteTaskByIncident(ctx, incidentID, taskID)
	if err != nil {
		logger.Error("Failed to complete task", "error", err, "incidentID", incidentID, "taskID", taskID)
		return goerr.Wrap(err, "failed to complete task")
	}

	// Update the message with completed task
	blocks := slackblocks.BuildTaskMessage(task, "")

	// Update the original message
	_, _, _, err = s.slackClient.UpdateMessage(
		ctx,
		interaction.Channel.ID,
		interaction.Message.Timestamp,
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		logger.Error("Failed to update task message", "error", err, "taskID", taskID)
		return goerr.Wrap(err, "failed to update task message")
	}

	logger.Info("Task completed successfully", "taskID", taskID)
	return nil
}

// handleTaskUncomplete handles task uncompletion
func (s *SlackInteraction) handleTaskUncomplete(ctx context.Context, interaction *slack.InteractionCallback, taskID types.TaskID) error {
	logger := ctxlog.From(ctx)

	// Get incident ID from channel for efficient task lookup
	incidentID, err := s.getIncidentIDByChannel(ctx, interaction.Channel.ID)
	if err != nil {
		logger.Error("Failed to get incident for task uncompletion", "error", err, "taskID", taskID, "channelID", interaction.Channel.ID)
		return goerr.Wrap(err, "failed to get incident for task uncompletion")
	}

	// Uncomplete the task efficiently
	task, err := s.taskUC.UncompleteTaskByIncident(ctx, incidentID, taskID)
	if err != nil {
		logger.Error("Failed to uncomplete task", "error", err, "incidentID", incidentID, "taskID", taskID)
		return goerr.Wrap(err, "failed to uncomplete task")
	}

	// Update the message with uncompleted task
	blocks := slackblocks.BuildTaskMessage(task, "")

	// Update the original message
	_, _, _, err = s.slackClient.UpdateMessage(
		ctx,
		interaction.Channel.ID,
		interaction.Message.Timestamp,
		slack.MsgOptionBlocks(blocks...),
	)
	if err != nil {
		logger.Error("Failed to update task message", "error", err, "taskID", taskID)
		return goerr.Wrap(err, "failed to update task message")
	}

	logger.Info("Task uncompleted successfully", "taskID", taskID)
	return nil
}

// handleTaskEdit handles task edit button click
func (s *SlackInteraction) handleTaskEdit(ctx context.Context, interaction *slack.InteractionCallback, taskID types.TaskID) error {
	logger := ctxlog.From(ctx)

	// Get incident ID from channel for efficient task lookup
	incidentID, err := s.getIncidentIDByChannel(ctx, interaction.Channel.ID)
	if err != nil {
		logger.Error("Failed to get incident for task editing", "error", err, "taskID", taskID, "channelID", interaction.Channel.ID)
		return goerr.Wrap(err, "failed to get incident for task editing")
	}

	// Get the task efficiently using incident ID
	task, err := s.taskUC.GetTaskByIncident(ctx, incidentID, taskID)
	if err != nil {
		logger.Error("Failed to get task for editing", "error", err, "incidentID", incidentID, "taskID", taskID)
		return goerr.Wrap(err, "failed to get task for editing")
	}

	// Get channel members for assignee selection
	// For now, use empty list - this should be implemented to get actual channel members
	var channelMembers []types.SlackUserID

	// Build edit modal
	modal := slackblocks.BuildTaskEditModal(task, channelMembers)

	// Open the modal
	_, err = s.slackClient.OpenView(ctx, interaction.TriggerID, modal)
	if err != nil {
		logger.Error("Failed to open task edit modal", "error", err, "taskID", taskID)
		return goerr.Wrap(err, "failed to open task edit modal")
	}

	logger.Info("Task edit modal opened", "taskID", taskID)
	return nil
}

// handleTaskEditSubmission handles task edit modal submission
func (s *SlackInteraction) handleTaskEditSubmission(ctx context.Context, interaction *slack.InteractionCallback) error {
	logger := ctxlog.From(ctx)

	// Extract incident ID and task ID from callback ID
	callbackData := strings.TrimPrefix(interaction.View.CallbackID, "task_edit_submit:")
	parts := strings.SplitN(callbackData, ":", 2)
	if len(parts) != 2 {
		return goerr.New("invalid callback ID format",
			goerr.V("callbackID", interaction.View.CallbackID))
	}

	incidentIDStr, taskIDStr := parts[0], parts[1]
	incidentIDUint, err := strconv.ParseUint(incidentIDStr, 10, 64)
	if err != nil {
		return goerr.Wrap(err, "invalid incident ID in callback",
			goerr.V("incidentIDStr", incidentIDStr))
	}

	// Check for overflow when converting uint64 to int
	if incidentIDUint > math.MaxInt64 {
		return goerr.New("incident ID too large",
			goerr.V("incidentID", incidentIDUint))
	}

	incidentID := types.IncidentID(incidentIDUint)
	taskID := types.TaskID(taskIDStr)

	// Extract values from modal
	var updates interfaces.TaskUpdateRequest

	// Extract title
	if titleBlock, ok := interaction.View.State.Values["title_block"]; ok {
		if titleInput, ok := titleBlock["title"]; ok && titleInput.Value != "" {
			updates.Title = &titleInput.Value
		}
	}

	// Extract description
	if descBlock, ok := interaction.View.State.Values["description_block"]; ok {
		if descInput, ok := descBlock["description"]; ok {
			updates.Description = &descInput.Value
		}
	}

	// Extract status
	if statusBlock, ok := interaction.View.State.Values["status_block"]; ok {
		if statusSelect, ok := statusBlock["status"]; ok && statusSelect.SelectedOption.Value != "" {
			status := model.TaskStatus(statusSelect.SelectedOption.Value)
			updates.Status = &status
		}
	}

	// Extract assignee
	if assigneeBlock, ok := interaction.View.State.Values["assignee_block"]; ok {
		if assigneeSelect, ok := assigneeBlock["assignee"]; ok && assigneeSelect.SelectedUser != "" {
			assigneeID := types.SlackUserID(assigneeSelect.SelectedUser)
			updates.AssigneeID = &assigneeID
		}
	}

	// Update the task efficiently using incident ID from callback
	updatedTask, err := s.taskUC.UpdateTaskByIncident(ctx, incidentID, taskID, updates)
	if err != nil {
		logger.Error("Failed to update task", "error", err, "incidentID", incidentID, "taskID", taskID)
		return goerr.Wrap(err, "failed to update task")
	}

	// Update the original task message with new information
	logger.Debug("Checking task message update conditions",
		"messageTS", updatedTask.MessageTS,
		"channelID", updatedTask.ChannelID,
		"taskID", taskID)

	if updatedTask.MessageTS != "" {
		// Determine the channel ID to use
		channelID := updatedTask.ChannelID
		if channelID == "" {
			// Fallback to incident channel ID for backward compatibility
			incident, err := s.incidentUC.GetIncident(ctx, int(updatedTask.IncidentID))
			if err != nil {
				logger.Warn("Failed to get incident for channel fallback", "error", err, "incidentID", updatedTask.IncidentID)
				return nil // Task update was successful, just can't update message
			}
			channelID = incident.ChannelID
			logger.Debug("Using incident channel as fallback", "channelID", channelID)
		}

		logger.Info("Updating task message in Slack",
			"taskID", taskID,
			"channelID", channelID,
			"messageTS", updatedTask.MessageTS)

		blocks := slackblocks.BuildTaskMessage(updatedTask, "")

		_, _, _, err = s.slackClient.UpdateMessage(
			ctx,
			string(channelID),
			updatedTask.MessageTS,
			slack.MsgOptionBlocks(blocks...),
		)
		if err != nil {
			logger.Warn("Failed to update task message after edit",
				"error", err,
				"taskID", taskID,
				"channelID", channelID,
				"messageTS", updatedTask.MessageTS)
			// Don't return error as the task update was successful
		} else {
			logger.Info("Task message updated successfully in Slack", "taskID", taskID)
		}
	} else {
		logger.Warn("Cannot update task message - missing messageTS",
			"taskID", taskID,
			"messageTS", updatedTask.MessageTS)
	}

	logger.Info("Task updated successfully", "taskID", taskID)
	return nil
}

// handleEditIncidentStatusAction handles edit incident status button action
func (s *SlackInteraction) handleEditIncidentStatusAction(ctx context.Context, interaction *slack.InteractionCallback, action *slack.BlockAction) error {
	ctxlog.From(ctx).Info("Edit incident status action triggered",
		"user", interaction.User.ID,
		"channel", interaction.Channel.ID,
		"incidentID", action.Value,
		"triggerID", interaction.TriggerID,
	)

	incidentIDStr := action.Value
	if incidentIDStr == "" {
		ctxlog.From(ctx).Error("Empty incident ID in action value")
		return goerr.New("empty incident ID")
	}

	// Call the status usecase to handle the status edit flow
	err := s.statusUC.HandleEditStatusAction(ctx, incidentIDStr, types.SlackUserID(interaction.User.ID), interaction.TriggerID)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to handle edit status action",
			"error", err,
			"incidentID", incidentIDStr,
			"user", interaction.User.ID,
		)
		return goerr.Wrap(err, "failed to handle edit status action")
	}

	ctxlog.From(ctx).Info("Edit status action handled successfully",
		"incidentID", incidentIDStr,
		"user", interaction.User.ID,
	)

	return nil
}

// handleEditIncidentDetailsAction handles edit incident details button action
func (s *SlackInteraction) handleEditIncidentDetailsAction(ctx context.Context, interaction *slack.InteractionCallback, action *slack.BlockAction) error {
	ctxlog.From(ctx).Info("Edit incident details action triggered",
		"user", interaction.User.ID,
		"channel", interaction.Channel.ID,
		"incidentID", action.Value,
		"triggerID", interaction.TriggerID,
	)

	incidentIDStr := action.Value
	if incidentIDStr == "" {
		ctxlog.From(ctx).Error("Empty incident ID in action value")
		return goerr.New("empty incident ID")
	}

	// Parse incident ID
	incidentIDInt, err := strconv.Atoi(incidentIDStr)
	if err != nil {
		ctxlog.From(ctx).Error("Invalid incident ID format",
			"error", err,
			"incidentID", incidentIDStr,
		)
		return goerr.Wrap(err, "invalid incident ID format")
	}
	incidentID := types.IncidentID(incidentIDInt)

	// Get the incident to show current values in the modal
	incident, err := s.incidentUC.GetIncident(ctx, incidentIDInt)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to get incident for editing",
			"error", err,
			"incidentID", incidentID,
		)
		return goerr.Wrap(err, "failed to get incident for editing")
	}

	// Build edit incident details modal
	modal := s.buildEditIncidentDetailsModal(incident)

	// Open the modal
	_, err = s.slackClient.OpenView(ctx, interaction.TriggerID, modal)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to open edit incident details modal",
			"error", err,
			"incidentID", incidentID,
		)
		return goerr.Wrap(err, "failed to open edit incident details modal")
	}

	ctxlog.From(ctx).Info("Edit incident details modal opened successfully",
		"incidentID", incidentID,
		"user", interaction.User.ID,
	)

	return nil
}

// handleStatusChangeModalSubmission handles status change modal submission
func (s *SlackInteraction) handleStatusChangeModalSubmission(ctx context.Context, interaction *slack.InteractionCallback) error {
	ctxlog.From(ctx).Info("Status change modal submitted",
		"user", interaction.User.ID,
		"team", interaction.Team.ID,
	)

	// Extract incident ID from private metadata
	incidentIDStr := interaction.View.PrivateMetadata
	if incidentIDStr == "" {
		return goerr.New("missing incident ID in private metadata")
	}

	incidentIDInt, err := strconv.Atoi(incidentIDStr)
	if err != nil {
		return goerr.Wrap(err, "invalid incident ID in private metadata")
	}
	incidentID := types.IncidentID(incidentIDInt)

	// Extract status from form values
	statusBlock, ok := interaction.View.State.Values["status_block"]
	if !ok {
		return goerr.New("status_block not found in form values")
	}

	statusSelect, ok := statusBlock["status_select"]
	if !ok {
		return goerr.New("status_select not found in status_block")
	}

	if statusSelect.SelectedOption.Value == "" {
		return goerr.New("no status selected")
	}

	newStatus := types.IncidentStatus(statusSelect.SelectedOption.Value)

	// Extract note (optional)
	var note string
	if noteBlock, ok := interaction.View.State.Values["note_block"]; ok {
		if noteInput, ok := noteBlock["note_input"]; ok && noteInput.Value != "" {
			note = noteInput.Value
		}
	}

	// Get user ID
	userID := types.SlackUserID(interaction.User.ID)

	// Call status update usecase
	err = s.statusUC.UpdateStatus(ctx, incidentID, newStatus, userID, note)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to update status",
			"error", err,
			"incidentID", incidentID,
			"newStatus", newStatus,
			"user", userID,
		)
		return goerr.Wrap(err, "failed to update incident status")
	}

	ctxlog.From(ctx).Info("Status updated successfully",
		"incidentID", incidentID,
		"newStatus", newStatus,
		"user", userID,
		"note", note,
	)

	return nil
}

// buildEditIncidentDetailsModal creates a modal for editing incident details
func (s *SlackInteraction) buildEditIncidentDetailsModal(incident *model.Incident) slack.ModalViewRequest {
	blocks := []slack.Block{
		&slack.InputBlock{
			Type:    slack.MBTInput,
			BlockID: "title_block",
			Label: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "Title",
			},
			Element: &slack.PlainTextInputBlockElement{
				Type:         slack.METPlainTextInput,
				ActionID:     "title_input",
				InitialValue: incident.Title,
				Placeholder: &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Enter incident title",
				},
			},
		},
		&slack.InputBlock{
			Type:    slack.MBTInput,
			BlockID: "description_block",
			Label: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "Description",
			},
			Element: &slack.PlainTextInputBlockElement{
				Type:         slack.METPlainTextInput,
				ActionID:     "description_input",
				Multiline:    true,
				InitialValue: incident.Description,
				Placeholder: &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Enter incident description",
				},
			},
			Optional: true,
		},
		&slack.InputBlock{
			Type:    slack.MBTInput,
			BlockID: "lead_block",
			Label: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "Lead",
			},
			Element: &slack.SelectBlockElement{
				Type:        slack.OptTypeUser,
				ActionID:    "lead_select",
				InitialUser: string(incident.Lead),
				Placeholder: &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Select incident lead",
				},
			},
			Optional: true,
		},
	}

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		CallbackID: "edit_incident_details_modal",
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Edit Incident Details",
		},
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Save",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
		Blocks: slack.Blocks{
			BlockSet: blocks,
		},
		PrivateMetadata: incident.ID.String(), // Store incident ID for submission
	}
}

// handleEditIncidentDetailsModalSubmission handles edit incident details modal submission
func (s *SlackInteraction) handleEditIncidentDetailsModalSubmission(ctx context.Context, interaction *slack.InteractionCallback) error {
	ctxlog.From(ctx).Info("Edit incident details modal submitted",
		"user", interaction.User.ID,
		"team", interaction.Team.ID,
		"privateMetadata", interaction.View.PrivateMetadata,
	)

	// Extract incident ID from private metadata
	incidentIDStr := interaction.View.PrivateMetadata
	if incidentIDStr == "" {
		ctxlog.From(ctx).Error("Empty incident ID in private metadata")
		return goerr.New("empty incident ID in private metadata")
	}

	// Parse incident ID
	incidentIDInt, err := strconv.Atoi(incidentIDStr)
	if err != nil {
		ctxlog.From(ctx).Error("Invalid incident ID format",
			"error", err,
			"incidentID", incidentIDStr,
		)
		return goerr.Wrap(err, "invalid incident ID format")
	}
	incidentID := types.IncidentID(incidentIDInt)

	// Extract values from modal
	var title, description string
	var lead types.SlackUserID

	// Extract title
	if titleBlock, ok := interaction.View.State.Values["title_block"]; ok {
		if titleInput, ok := titleBlock["title_input"]; ok {
			title = titleInput.Value
		}
	}

	// Extract description
	if descBlock, ok := interaction.View.State.Values["description_block"]; ok {
		if descInput, ok := descBlock["description_input"]; ok {
			description = descInput.Value
		}
	}

	// Extract lead
	if leadBlock, ok := interaction.View.State.Values["lead_block"]; ok {
		if leadSelect, ok := leadBlock["lead_select"]; ok && leadSelect.SelectedUser != "" {
			lead = types.SlackUserID(leadSelect.SelectedUser)
		}
	}

	// Call the incident usecase to update details
	updatedIncident, err := s.incidentUC.UpdateIncidentDetails(ctx, incidentID, title, description, lead)
	if err != nil {
		ctxlog.From(ctx).Error("Failed to update incident details",
			"error", err,
			"incidentID", incidentID,
			"title", title,
			"description", description,
			"lead", lead,
		)
		return goerr.Wrap(err, "failed to update incident details")
	}

	ctxlog.From(ctx).Info("Incident details updated successfully",
		"incidentID", incidentID,
		"title", updatedIncident.Title,
		"description", updatedIncident.Description,
		"lead", updatedIncident.Lead,
	)

	return nil
}
