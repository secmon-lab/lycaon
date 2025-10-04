package usecase

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/slack-go/slack"
)

// StatusUseCase provides status management functionality
type StatusUseCase struct {
	repo         interfaces.Repository
	slackClient  interfaces.SlackClient
	config       *model.Config
	blockBuilder interfaces.BlockBuilder
}

// NewStatusUseCase creates a new StatusUseCase instance
func NewStatusUseCase(repo interfaces.Repository, slackClient interfaces.SlackClient, config *model.Config, blockBuilder interfaces.BlockBuilder) *StatusUseCase {
	return &StatusUseCase{
		repo:         repo,
		slackClient:  slackClient,
		config:       config,
		blockBuilder: blockBuilder,
	}
}

// UpdateStatus updates the incident status and records the change in history
func (uc *StatusUseCase) UpdateStatus(ctx context.Context, incidentID types.IncidentID, incidentStatus types.IncidentStatus, userID types.SlackUserID, note string) error {
	// Validate input
	if err := incidentID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid incident ID")
	}

	if !incidentStatus.IsValid() {
		return goerr.New("invalid status", goerr.V("status", incidentStatus))
	}

	if userID == "" {
		return goerr.New("user ID is required")
	}

	// Get existing incident to check current status
	incident, err := uc.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return goerr.Wrap(err, "failed to get incident")
	}

	// Check if status is actually changing
	if incident.Status == incidentStatus {
		return goerr.New("status is already set to the same value",
			goerr.V("currentStatus", incident.Status),
			goerr.V("newStatus", incidentStatus))
	}

	// Create status history entry
	statusHistory, err := model.NewStatusHistory(incidentID, incidentStatus, userID, note)
	if err != nil {
		return goerr.Wrap(err, "failed to create status history")
	}

	// Add status history to repository
	if err := uc.repo.AddStatusHistory(ctx, statusHistory); err != nil {
		return goerr.Wrap(err, "failed to add status history")
	}

	// Update incident status
	if err := uc.repo.UpdateIncidentStatus(ctx, incidentID, incidentStatus); err != nil {
		return goerr.Wrap(err, "failed to update incident status")
	}

	return nil
}

// GetStatusHistory retrieves status history for an incident with user information
func (uc *StatusUseCase) GetStatusHistory(ctx context.Context, incidentID types.IncidentID) ([]*model.StatusHistoryWithUser, error) {
	if err := incidentID.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid incident ID")
	}

	// Get status histories
	histories, err := uc.repo.GetStatusHistories(ctx, incidentID)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to get status histories")
	}

	// Enrich with user information
	result := make([]*model.StatusHistoryWithUser, 0, len(histories))
	for _, history := range histories {
		// Get user information
		user, err := uc.repo.GetUserBySlackID(ctx, history.ChangedBy)
		if err != nil {
			// If user not found, create a minimal user record
			user = &model.User{
				ID:    types.UserID(history.ChangedBy),
				Name:  string(history.ChangedBy), // Fallback to slack ID
				Email: "",
			}
		}

		historyWithUser := model.NewStatusHistoryWithUser(history, user)
		result = append(result, historyWithUser)
	}

	return result, nil
}

// PostStatusMessage posts a status message to the incident channel
func (uc *StatusUseCase) PostStatusMessage(ctx context.Context, channelID types.ChannelID, incidentID types.IncidentID) error {
	if channelID == "" {
		return goerr.New("channel ID is required")
	}

	if err := incidentID.Validate(); err != nil {
		return goerr.Wrap(err, "invalid incident ID")
	}

	// Get incident information
	incident, err := uc.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return goerr.Wrap(err, "failed to get incident")
	}

	// Get lead user information
	var leadName string
	if incident.Lead != "" {
		leadUser, err := uc.repo.GetUserBySlackID(ctx, incident.Lead)
		if err == nil && leadUser != nil {
			leadName = leadUser.Name
		} else {
			leadName = string(incident.Lead)
		}
	} else {
		leadName = "Not assigned"
	}

	// Build status message blocks
	blocks := uc.BuildStatusMessageBlocks(incident, leadName)

	// Post message to Slack
	_, _, err = uc.slackClient.PostMessage(ctx, string(channelID), slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return goerr.Wrap(err, "failed to post status message to Slack")
	}

	return nil
}

// BuildStatusMessageBlocks creates Slack message blocks for status display
func (uc *StatusUseCase) BuildStatusMessageBlocks(incident *model.Incident, leadName string) []slack.Block {
	return uc.blockBuilder.BuildStatusMessageBlocks(incident, leadName, uc.config)
}

// HandleEditStatusAction handles Slack edit status action by opening a status selection modal
func (uc *StatusUseCase) HandleEditStatusAction(ctx context.Context, incidentIDStr string, userID types.SlackUserID, triggerID string, channelID string, messageTS string) error {
	// Parse incident ID
	incidentIDInt, err := strconv.Atoi(incidentIDStr)
	if err != nil {
		return goerr.Wrap(err, "invalid incident ID")
	}
	incidentID := types.IncidentID(incidentIDInt)

	// Get incident to verify it exists and get current status
	incident, err := uc.repo.GetIncident(ctx, incidentID)
	if err != nil {
		return goerr.Wrap(err, "failed to get incident")
	}

	// Build status selection modal
	modalView := uc.buildStatusSelectionModal(incident, channelID, messageTS)

	// Open modal
	_, err = uc.slackClient.OpenView(ctx, triggerID, modalView)
	if err != nil {
		return goerr.Wrap(err, "failed to open status selection modal")
	}

	return nil
}

// buildStatusSelectionModal creates a modal for status selection
func (uc *StatusUseCase) buildStatusSelectionModal(incident *model.Incident, channelID string, messageTS string) slack.ModalViewRequest {
	// Create status options
	statusOptions := []*slack.OptionBlockObject{}
	statuses := []types.IncidentStatus{
		types.IncidentStatusTriage,
		types.IncidentStatusHandling,
		types.IncidentStatusMonitoring,
		types.IncidentStatusClosed,
	}

	for _, status := range statuses {
		emoji := getStatusEmoji(status)
		statusOptions = append(statusOptions, &slack.OptionBlockObject{
			Text: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: emoji + " " + string(status),
			},
			Value: string(status),
		})
	}

	blocks := []slack.Block{
		&slack.SectionBlock{
			Type: slack.MBTSection,
			Text: &slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: "*Select new status for incident:*",
			},
		},
		&slack.InputBlock{
			Type:    slack.MBTInput,
			BlockID: "status_block",
			Label: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "Status",
			},
			Element: &slack.SelectBlockElement{
				Type:     slack.OptTypeStatic,
				ActionID: "status_select",
				Placeholder: &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Choose a status...",
				},
				Options: statusOptions,
			},
		},
		&slack.InputBlock{
			Type:     slack.MBTInput,
			BlockID:  "note_block",
			Optional: true,
			Label: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: "Note (optional)",
			},
			Element: &slack.PlainTextInputBlockElement{
				Type:      slack.METPlainTextInput,
				ActionID:  "note_input",
				Multiline: true,
				Placeholder: &slack.TextBlockObject{
					Type: slack.PlainTextType,
					Text: "Add a note about this status change...",
				},
			},
		},
	}

	return slack.ModalViewRequest{
		Type:       slack.VTModal,
		CallbackID: "status_change_modal",
		Title: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Change Status",
		},
		Submit: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Update",
		},
		Close: &slack.TextBlockObject{
			Type: slack.PlainTextType,
			Text: "Cancel",
		},
		Blocks: slack.Blocks{
			BlockSet: blocks,
		},
		PrivateMetadata: uc.buildPrivateMetadata(incident.ID.String(), channelID, messageTS), // Store context for submission
	}
}

// StatusChangePrivateMetadata represents context data stored in status change modal private_metadata
type StatusChangePrivateMetadata struct {
	IncidentID       string `json:"incident_id"`
	ChannelID        string `json:"channel_id"`
	MessageTimestamp string `json:"message_timestamp"`
}

// buildPrivateMetadata creates base64-encoded JSON private metadata
func (uc *StatusUseCase) buildPrivateMetadata(incidentID, channelID, messageTS string) string {
	context := StatusChangePrivateMetadata{
		IncidentID:       incidentID,
		ChannelID:        channelID,
		MessageTimestamp: messageTS,
	}

	jsonData, err := json.Marshal(context)
	if err != nil {
		// Should not happen with simple struct
		return incidentID
	}

	return base64.StdEncoding.EncodeToString(jsonData)
}

// parseStatusChangePrivateMetadata parses base64-encoded JSON private metadata
func parseStatusChangePrivateMetadata(privateMetadata string) (*StatusChangePrivateMetadata, error) {
	if privateMetadata == "" {
		return nil, goerr.New("private metadata is empty")
	}

	// Decode base64
	jsonData, err := base64.StdEncoding.DecodeString(privateMetadata)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to decode base64 private metadata")
	}

	var context StatusChangePrivateMetadata
	if err := json.Unmarshal(jsonData, &context); err != nil {
		return nil, goerr.Wrap(err, "failed to unmarshal private metadata JSON")
	}

	return &context, nil
}

// getStatusEmoji returns emoji for status display in modal options
func getStatusEmoji(status types.IncidentStatus) string {
	switch status {
	case types.IncidentStatusTriage:
		return "🟡"
	case types.IncidentStatusHandling:
		return "🔴"
	case types.IncidentStatusMonitoring:
		return "🟠"
	case types.IncidentStatusClosed:
		return "🟢"
	default:
		return "⚪"
	}
}

// UpdateOriginalStatusMessage updates the original status message with new incident status
func (uc *StatusUseCase) UpdateOriginalStatusMessage(ctx context.Context, channelID types.ChannelID, messageTS string, incident *model.Incident) error {
	if channelID == "" || messageTS == "" {
		return goerr.New("channelID and messageTS are required",
			goerr.V("channelID", channelID),
			goerr.V("messageTS", messageTS))
	}

	// Get lead user information
	var leadName string
	if incident.Lead != "" {
		leadUser, err := uc.repo.GetUserBySlackID(ctx, incident.Lead)
		if err == nil && leadUser != nil {
			leadName = leadUser.Name
		} else {
			leadName = string(incident.Lead)
		}
	} else {
		leadName = "Not assigned"
	}

	// Build updated status message blocks
	blocks := uc.BuildStatusMessageBlocks(incident, leadName)

	// Update the original message
	_, _, _, err := uc.slackClient.UpdateMessage(ctx, string(channelID), messageTS, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return goerr.Wrap(err, "failed to update original status message",
			goerr.V("channelID", channelID),
			goerr.V("messageTS", messageTS),
			goerr.V("incidentID", incident.ID))
	}

	return nil
}

// HandleStatusChangeModalSubmission handles status change modal submission processing
func (uc *StatusUseCase) HandleStatusChangeModalSubmission(ctx context.Context, privateMetadata string, statusValue, noteValue, userID string) error {
	// Parse private metadata to extract context information
	context, err := parseStatusChangePrivateMetadata(privateMetadata)
	if err != nil {
		return goerr.Wrap(err, "failed to parse private metadata")
	}

	incidentIDInt, err := strconv.Atoi(context.IncidentID)
	if err != nil {
		return goerr.Wrap(err, "invalid incident ID in private metadata")
	}
	incidentID := types.IncidentID(incidentIDInt)

	if statusValue == "" {
		return goerr.New("no status selected")
	}

	newStatus := types.IncidentStatus(statusValue)
	slackUserID := types.SlackUserID(userID)

	// Call status update usecase
	err = uc.UpdateStatus(ctx, incidentID, newStatus, slackUserID, noteValue)
	if err != nil {
		return goerr.Wrap(err, "failed to update incident status")
	}

	// Update original status message if context contains message information
	if context.ChannelID != "" && context.MessageTimestamp != "" {
		// Get updated incident information
		incident, err := uc.repo.GetIncident(ctx, incidentID)
		if err != nil {
			return goerr.Wrap(err, "failed to get incident for status message update")
		}

		// Update the original status message
		if err := uc.UpdateOriginalStatusMessage(ctx, types.ChannelID(context.ChannelID), context.MessageTimestamp, incident); err != nil {
			return goerr.Wrap(err, "failed to update original status message")
		}
	}

	return nil
}
