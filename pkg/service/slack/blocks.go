package slack

import (
	"fmt"
	"strings"

	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/slack-go/slack"
)

// Block and Action ID constants for Slack interactions
const (
	BlockIDSeverityInput   = "severity_block"
	ActionIDSeveritySelect = "severity_select"
)

// GetSeverityEmoji returns emoji based on severity level
func GetSeverityEmoji(level int) string {
	switch {
	case level >= 80:
		return "🚨" // Critical
	case level >= 50:
		return "⚠️" // High/Medium
	case level >= 10:
		return "ℹ️" // Low/Info
	default:
		return "✅" // Ignorable (level 0)
	}
}

// formatSeverityText formats severity for display with emoji
func formatSeverityText(severity *model.Severity) string {
	if severity == nil {
		return "❓ Unknown"
	}
	emoji := GetSeverityEmoji(severity.Level)
	return fmt.Sprintf("%s %s", emoji, severity.Name)
}

// BlockBuilder provides methods to build Slack message blocks
type BlockBuilder struct{}

// NewBlockBuilder creates a new BlockBuilder instance
func NewBlockBuilder() *BlockBuilder {
	return &BlockBuilder{}
}

// buildSeverityInputBlock creates a severity selection input block
func buildSeverityInputBlock(severityID string, severities []model.Severity) *slack.InputBlock {
	var severityOptions []*slack.OptionBlockObject
	var initialOption *slack.OptionBlockObject

	// Build options from severities config
	if len(severities) == 0 {
		// Add a default unknown severity if no severities are available
		option := slack.NewOptionBlockObject(
			"unknown",
			slack.NewTextBlockObject(slack.PlainTextType, "Unknown", false, false),
			slack.NewTextBlockObject(slack.PlainTextType, "Unknown severity", false, false),
		)
		severityOptions = append(severityOptions, option)
		if severityID == "unknown" || severityID == "" {
			initialOption = option
		}
	} else {
		for _, severity := range severities {
			// Truncate description to avoid Slack limits (max 75 chars for option description)
			description := severity.Description
			if len(description) > 75 {
				description = description[:72] + "..."
			}

			option := slack.NewOptionBlockObject(
				severity.ID,
				slack.NewTextBlockObject(slack.PlainTextType, severity.Name, false, false),
				slack.NewTextBlockObject(slack.PlainTextType, description, false, false),
			)
			severityOptions = append(severityOptions, option)

			// Set initial selection if this is the current severity
			if severity.ID == severityID {
				initialOption = option
			}
		}
	}

	severityBlock := slack.NewInputBlock(
		BlockIDSeverityInput,
		slack.NewTextBlockObject(
			slack.PlainTextType,
			"Severity (optional)",
			false,
			false,
		),
		nil,
		slack.NewOptionsSelectBlockElement(
			"static_select",
			slack.NewTextBlockObject(
				slack.PlainTextType,
				"Select incident severity",
				false,
				false,
			),
			ActionIDSeveritySelect,
			severityOptions...,
		),
	)

	// Set initial option if we have a matching severity
	if initialOption != nil {
		if selectElement, ok := severityBlock.Element.(*slack.SelectBlockElement); ok {
			selectElement.InitialOption = initialOption
		}
	}

	// Make severity optional
	severityBlock.Optional = true

	return severityBlock
}

// BuildIncidentPromptBlocks builds blocks for incident creation prompt
func (b *BlockBuilder) BuildIncidentPromptBlocks(requestID, title, description, categoryID, severityID string, config *model.Config) []slack.Block {
	blocks := []slack.Block{}

	// Title as header
	headerText := "Incident"
	if title != "" {
		headerText = title
	}
	blocks = append(blocks, slack.NewHeaderBlock(
		slack.NewTextBlockObject(
			slack.PlainTextType,
			headerText,
			false,
			false,
		),
	))

	// Category and Severity as fields (side by side)
	fields := []*slack.TextBlockObject{}
	if categoryID != "" {
		categoryText := categoryID
		if config != nil {
			category := config.FindCategoryByIDWithFallback(categoryID)
			categoryText = fmt.Sprintf("📂 %s", category.Name)
		}
		fields = append(fields, slack.NewTextBlockObject(
			slack.MarkdownType,
			fmt.Sprintf("*Category:*\n%s", categoryText),
			false,
			false,
		))
	}
	if severityID != "" {
		severityText := severityID
		if config != nil {
			severity := config.FindSeverityByIDWithFallback(severityID)
			severityText = formatSeverityText(severity)
		}
		fields = append(fields, slack.NewTextBlockObject(
			slack.MarkdownType,
			fmt.Sprintf("*Severity:*\n%s", severityText),
			false,
			false,
		))
	}
	if len(fields) > 0 {
		blocks = append(blocks, slack.NewSectionBlock(nil, fields, nil))
	}

	// Description section
	if description != "" {
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("*Description:*\n%s", description),
				false,
				false,
			),
			nil,
			nil,
		))
	}

	// Divider for visual separation
	blocks = append(blocks, slack.NewDividerBlock())

	// Action buttons
	blocks = append(blocks, slack.NewActionBlock(
		"incident_creation",
		slack.NewButtonBlockElement(
			"create_incident",
			requestID,
			slack.NewTextBlockObject(
				slack.PlainTextType,
				"Declare",
				false,
				false,
			),
		).WithStyle(slack.StylePrimary),
		slack.NewButtonBlockElement(
			"edit_incident",
			requestID,
			slack.NewTextBlockObject(
				slack.PlainTextType,
				"Edit",
				false,
				false,
			),
		),
	))

	return blocks
}

// BuildIncidentCreatedBlocks builds blocks for incident created notification
func (b *BlockBuilder) BuildIncidentCreatedBlocks(channelName, channelID, title, categoryID, severityID string, config *model.Config) []slack.Block {
	channelLink := fmt.Sprintf("<#%s>", channelID)
	category := config.FindCategoryByIDWithFallback(categoryID)

	var message string
	if title != "" {
		message = fmt.Sprintf("✅ Incident channel %s has been created for: *%s*\n*Category:* %s", channelLink, title, category.Name)
	} else {
		message = fmt.Sprintf("✅ Incident channel %s has been created\n*Category:* %s", channelLink, category.Name)
	}

	// Add severity if available
	severity := config.FindSeverityByIDWithFallback(severityID)
	severityText := formatSeverityText(severity)
	message = fmt.Sprintf("%s\n*Severity:* %s", message, severityText)

	return []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				message,
				false,
				false,
			),
			nil,
			nil,
		),
	}
}

// BuildIncidentChannelWelcomeBlocks builds blocks for the welcome message in the incident channel
func (b *BlockBuilder) BuildIncidentChannelWelcomeBlocks(incident *model.Incident, originChannelName string, leadName string, config *model.Config) []slack.Block {
	// Use BuildStatusMessageBlocks as base
	blocks := b.BuildStatusMessageBlocks(incident, leadName, config)

	// Insert origin info after header
	originInfo := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			slack.MarkdownType,
			fmt.Sprintf("This incident was created from *#%s* by <@%s>", originChannelName, incident.CreatedBy),
			false,
			false,
		),
		nil,
		nil,
	)

	// Insert after header (index 0) and before divider (index 1)
	blocks = append(blocks[:1], append([]slack.Block{originInfo}, blocks[1:]...)...)

	// Add welcome message before action buttons
	// Find action block index (should be last)
	actionIndex := len(blocks) - 1

	welcomeSection := slack.NewSectionBlock(
		slack.NewTextBlockObject(
			slack.MarkdownType,
			"Please use this channel to coordinate incident response activities.",
			false,
			false,
		),
		nil,
		nil,
	)

	// Insert welcome section before action buttons
	blocks = append(blocks[:actionIndex], append([]slack.Block{welcomeSection}, blocks[actionIndex:]...)...)

	return blocks
}

// BuildErrorBlocks builds blocks for error messages
func (b *BlockBuilder) BuildErrorBlocks(errorMessage string) []slack.Block {
	return []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("❌ Error: %s", errorMessage),
				false,
				false,
			),
			nil,
			nil,
		),
	}
}

// BuildIncidentEditModal builds the modal view for editing incident details
func (b *BlockBuilder) BuildIncidentEditModal(requestID, title, description, categoryID, severityID string, categories []model.Category, severities []model.Severity) slack.ModalViewRequest {
	// Title input block
	titleBlock := slack.NewInputBlock(
		"title_block",
		slack.NewTextBlockObject(
			slack.PlainTextType,
			"Title",
			false,
			false,
		),
		nil,
		slack.NewPlainTextInputBlockElement(
			slack.NewTextBlockObject(
				slack.PlainTextType,
				"Enter incident title",
				false,
				false,
			),
			"title_input",
		),
	)

	// Set initial value if title is provided
	if title != "" {
		titleInput := titleBlock.Element.(*slack.PlainTextInputBlockElement)
		titleInput.InitialValue = title
	}

	// Description input block
	descriptionBlock := slack.NewInputBlock(
		"description_block",
		slack.NewTextBlockObject(
			slack.PlainTextType,
			"Description (optional)",
			false,
			false,
		),
		nil,
		slack.NewPlainTextInputBlockElement(
			slack.NewTextBlockObject(
				slack.PlainTextType,
				"Enter incident description",
				false,
				false,
			),
			"description_input",
		),
	)
	// Make it multiline and set initial value if description is provided
	descriptionInput := descriptionBlock.Element.(*slack.PlainTextInputBlockElement)
	descriptionInput.Multiline = true
	if description != "" {
		descriptionInput.InitialValue = description
	}
	descriptionBlock.Optional = true

	// Category selection block
	var categoryOptions []*slack.OptionBlockObject
	var initialOption *slack.OptionBlockObject

	// Ensure we have at least one category option
	if len(categories) == 0 {
		// Add a default unknown category if no categories are available
		option := slack.NewOptionBlockObject(
			"unknown",
			slack.NewTextBlockObject(slack.PlainTextType, "Unknown", false, false),
			slack.NewTextBlockObject(slack.PlainTextType, "Incidents that cannot be categorized", false, false),
		)
		categoryOptions = append(categoryOptions, option)
		if categoryID == "unknown" || categoryID == "" {
			initialOption = option
		}
	} else {
		for _, category := range categories {
			// Truncate description to avoid Slack limits (max 75 chars for option description)
			description := category.Description
			if len(description) > 75 {
				description = description[:72] + "..."
			}

			option := slack.NewOptionBlockObject(
				category.ID,
				slack.NewTextBlockObject(slack.PlainTextType, category.Name, false, false),
				slack.NewTextBlockObject(slack.PlainTextType, description, false, false),
			)
			categoryOptions = append(categoryOptions, option)

			// Set initial selection if this is the current category
			if category.ID == categoryID {
				initialOption = option
			}
		}
	}

	categoryBlock := slack.NewInputBlock(
		"category_block",
		slack.NewTextBlockObject(
			slack.PlainTextType,
			"Category",
			false,
			false,
		),
		nil,
		slack.NewOptionsSelectBlockElement(
			"static_select",
			slack.NewTextBlockObject(
				slack.PlainTextType,
				"Select incident category",
				false,
				false,
			),
			"category_select",
			categoryOptions...,
		),
	)

	// Set initial option if we have a matching category
	if initialOption != nil {
		if selectElement, ok := categoryBlock.Element.(*slack.SelectBlockElement); ok {
			selectElement.InitialOption = initialOption
		}
	}

	// Build severity selection block
	severityBlock := buildSeverityInputBlock(severityID, severities)

	return slack.ModalViewRequest{
		Type:            slack.ViewType("modal"),
		Title:           slack.NewTextBlockObject(slack.PlainTextType, "Edit Incident", false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, "Declare", false, false),
		Close:           slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		CallbackID:      "incident_edit_modal",
		PrivateMetadata: requestID, // Store request ID in private metadata
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				titleBlock,
				descriptionBlock,
				categoryBlock,
				severityBlock,
			},
		},
	}
}

// BuildIncidentPromptUsedBlocks builds blocks for incident prompt after it has been used (buttons disabled)
func (b *BlockBuilder) BuildIncidentPromptUsedBlocks(title string) []slack.Block {
	var promptText string
	if title != "" {
		promptText = fmt.Sprintf("✅ Incident declared for: *%s*", title)
	} else {
		promptText = "✅ Incident declared"
	}

	return []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				promptText,
				false,
				false,
			),
			nil,
			nil,
		),
	}
}

// BuildIncidentProcessingBlocks builds blocks to show that incident command is being processed
func (b *BlockBuilder) BuildIncidentProcessingBlocks() []slack.Block {
	return b.BuildContextBlocks("🔄 Processing incident command...")
}

// BuildContextBlocks builds generic context blocks with the given message
func (b *BlockBuilder) BuildContextBlocks(message string) []slack.Block {
	return []slack.Block{
		slack.NewContextBlock(
			"",
			slack.NewTextBlockObject(
				slack.MarkdownType,
				message,
				false,
				false,
			),
		),
	}
}

// BuildStatusMessageBlocks creates Slack message blocks for status display
func (b *BlockBuilder) BuildStatusMessageBlocks(incident *model.Incident, leadName string, config *model.Config) []slack.Block {
	statusEmoji := getStatusEmoji(incident.Status)

	// Build category text
	var category *model.Category
	if config != nil {
		category = config.FindCategoryByIDWithFallback(incident.CategoryID)
	}
	categoryText := "Unknown"
	if category != nil {
		categoryText = fmt.Sprintf("📂 %s", category.Name)
	}

	// Build severity text
	var severity *model.Severity
	if config != nil {
		severity = config.FindSeverityByIDWithFallback(incident.SeverityID.String())
	}
	severityText := formatSeverityText(severity)

	blocks := []slack.Block{
		&slack.HeaderBlock{
			Type: slack.MBTHeader,
			Text: &slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: incident.Title,
			},
		},
		&slack.DividerBlock{
			Type: slack.MBTDivider,
		},
		&slack.SectionBlock{
			Type: slack.MBTSection,
			Fields: []*slack.TextBlockObject{
				{
					Type: slack.MarkdownType,
					Text: "*Status:*\n" + statusEmoji + " " + string(incident.Status),
				},
				{
					Type: slack.MarkdownType,
					Text: "*Lead:*\n<@" + leadName + ">",
				},
				{
					Type: slack.MarkdownType,
					Text: "*Category:*\n" + categoryText,
				},
				{
					Type: slack.MarkdownType,
					Text: "*Severity:*\n" + severityText,
				},
			},
		},
		&slack.SectionBlock{
			Type: slack.MBTSection,
			Text: &slack.TextBlockObject{
				Type: slack.MarkdownType,
				Text: "*Description:*\n" + strings.ReplaceAll(incident.Description, "\n", " "),
			},
		},
		&slack.ActionBlock{
			Type:    slack.MBTAction,
			BlockID: "status_actions",
			Elements: &slack.BlockElements{
				ElementSet: []slack.BlockElement{
					&slack.ButtonBlockElement{
						Type:     slack.METButton,
						ActionID: "edit_incident_status",
						Text: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "Status update",
						},
						Style: slack.StylePrimary,
						Value: incident.ID.String(),
					},
					&slack.ButtonBlockElement{
						Type:     slack.METButton,
						ActionID: "edit_incident_details",
						Text: &slack.TextBlockObject{
							Type: slack.PlainTextType,
							Text: "Edit incident",
						},
						Style: slack.StyleDefault,
						Value: incident.ID.String(),
					},
				},
			},
		},
	}

	return blocks
}

// getStatusEmoji returns emoji based on incident status
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
