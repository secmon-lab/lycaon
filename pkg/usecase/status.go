package usecase

import (
	"context"
	"strconv"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/slack-go/slack"
	slackgo "github.com/slack-go/slack"
)

// StatusUseCase provides status management functionality
type StatusUseCase struct {
	repo        interfaces.Repository
	slackClient interfaces.SlackClient
}

// NewStatusUseCase creates a new StatusUseCase instance
func NewStatusUseCase(repo interfaces.Repository, slackClient interfaces.SlackClient) *StatusUseCase {
	return &StatusUseCase{
		repo:        repo,
		slackClient: slackClient,
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
				SlackUserID: history.ChangedBy,
				Name:        string(history.ChangedBy), // Fallback to slack ID
				Email:       "",
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
	blocks := uc.buildStatusMessageBlocks(incident, leadName)

	// Post message to Slack
	_, _, err = uc.slackClient.PostMessage(ctx, string(channelID), slackgo.MsgOptionBlocks(blocks...))
	if err != nil {
		return goerr.Wrap(err, "failed to post status message to Slack")
	}

	return nil
}

// buildStatusMessageBlocks creates Slack message blocks for status display
func (uc *StatusUseCase) buildStatusMessageBlocks(incident *model.Incident, leadName string) []slackgo.Block {
	statusEmoji := uc.getStatusEmoji(incident.Status)

	blocks := []slackgo.Block{
		&slackgo.SectionBlock{
			Type: slackgo.MBTSection,
			Text: &slackgo.TextBlockObject{
				Type: slackgo.MarkdownType,
				Text: "*Incident Status*",
			},
		},
		&slackgo.SectionBlock{
			Type: slackgo.MBTSection,
			Fields: []*slackgo.TextBlockObject{
				{
					Type: slackgo.MarkdownType,
					Text: "*Title:*\n" + incident.Title,
				},
				{
					Type: slackgo.MarkdownType,
					Text: "*Status:*\n" + statusEmoji + " " + string(incident.Status),
				},
				{
					Type: slackgo.MarkdownType,
					Text: "*Lead:*\n" + leadName,
				},
				{
					Type: slackgo.MarkdownType,
					Text: "*Description:*\n" + incident.Description,
				},
			},
		},
		&slackgo.ActionBlock{
			Type:    slackgo.MBTAction,
			BlockID: "status_actions",
			Elements: &slackgo.BlockElements{
				ElementSet: []slackgo.BlockElement{
					&slackgo.ButtonBlockElement{
						Type:     slackgo.METButton,
						ActionID: "edit_incident_status",
						Text: &slackgo.TextBlockObject{
							Type: slackgo.PlainTextType,
							Text: "Edit",
						},
						Style: slackgo.StylePrimary,
						Value: incident.ID.String(),
					},
				},
			},
		},
	}

	return blocks
}

// HandleEditStatusAction handles Slack edit status action by opening a status selection modal
func (uc *StatusUseCase) HandleEditStatusAction(ctx context.Context, incidentIDStr string, userID types.SlackUserID, triggerID string) error {
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
	modalView := uc.buildStatusSelectionModal(incident)

	// Open modal
	_, err = uc.slackClient.OpenView(ctx, triggerID, modalView)
	if err != nil {
		return goerr.Wrap(err, "failed to open status selection modal")
	}

	return nil
}

// buildStatusSelectionModal creates a modal for status selection
func (uc *StatusUseCase) buildStatusSelectionModal(incident *model.Incident) slack.ModalViewRequest {
	// Create status options
	statusOptions := []*slack.OptionBlockObject{}
	statuses := []types.IncidentStatus{
		types.IncidentStatusTriage,
		types.IncidentStatusHandling,
		types.IncidentStatusMonitoring,
		types.IncidentStatusClosed,
	}

	for _, status := range statuses {
		emoji := uc.getStatusEmoji(status)
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
				Type:        slack.OptTypeStatic,
				ActionID:    "status_select",
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
				Type:     slack.METPlainTextInput,
				ActionID: "note_input",
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
		PrivateMetadata: incident.ID.String(), // Store incident ID for submission
	}
}

// getStatusEmoji returns emoji for status display
func (uc *StatusUseCase) getStatusEmoji(status types.IncidentStatus) string {
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

// getStatusColor returns color for status display
func (uc *StatusUseCase) getStatusColor(status types.IncidentStatus) string {
	switch status {
	case types.IncidentStatusTriage:
		return "#f59e0b" // Amber
	case types.IncidentStatusHandling:
		return "#f44336" // Red
	case types.IncidentStatusMonitoring:
		return "#ff9800" // Orange
	case types.IncidentStatusClosed:
		return "#4caf50" // Green
	default:
		return "#9e9e9e" // Grey
	}
}
