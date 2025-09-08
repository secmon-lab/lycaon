package slack

import (
	"fmt"

	"github.com/slack-go/slack"
)

// BlockBuilder provides methods to build Slack message blocks
type BlockBuilder struct{}

// NewBlockBuilder creates a new BlockBuilder instance
func NewBlockBuilder() *BlockBuilder {
	return &BlockBuilder{}
}

// BuildIncidentPromptBlocks builds blocks for incident creation prompt
func (b *BlockBuilder) BuildIncidentPromptBlocks(requestID, title string) []slack.Block {
	var promptText string
	if title != "" {
		promptText = fmt.Sprintf("Do you want to create an incident for: *%s*?", title)
	} else {
		promptText = "Do you want to create an incident?"
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
		slack.NewActionBlock(
			"incident_creation",
			slack.NewButtonBlockElement(
				"create_incident",
				requestID, // Pass request ID as value
				slack.NewTextBlockObject(
					slack.PlainTextType,
					"Declare",
					false,
					false,
				),
			).WithStyle(slack.StylePrimary),
			slack.NewButtonBlockElement(
				"edit_incident",
				requestID, // Pass request ID as value
				slack.NewTextBlockObject(
					slack.PlainTextType,
					"Edit",
					false,
					false,
				),
			), // No style = default/secondary appearance
		),
	}
}

// BuildIncidentCreatedBlocks builds blocks for incident created notification
func (b *BlockBuilder) BuildIncidentCreatedBlocks(channelName, channelID, title string) []slack.Block {
	channelLink := fmt.Sprintf("<#%s>", channelID)
	var message string
	if title != "" {
		message = fmt.Sprintf("✅ Incident channel %s has been created for: *%s*", channelLink, title)
	} else {
		message = fmt.Sprintf("✅ Incident channel %s has been created", channelLink)
	}

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
func (b *BlockBuilder) BuildIncidentChannelWelcomeBlocks(incidentID int, originChannelName string, createdBy string, description string) []slack.Block {
	blocks := []slack.Block{
		slack.NewHeaderBlock(
			slack.NewTextBlockObject(
				slack.PlainTextType,
				fmt.Sprintf("Incident #%d", incidentID),
				false,
				false,
			),
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				fmt.Sprintf("This incident was created from *#%s* by <@%s>", originChannelName, createdBy),
				false,
				false,
			),
			nil,
			nil,
		),
	}

	// Add description if provided
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

	blocks = append(blocks,
		slack.NewDividerBlock(),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				slack.MarkdownType,
				"Please use this channel to coordinate incident response activities.",
				false,
				false,
			),
			nil,
			nil,
		),
	)

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
func (b *BlockBuilder) BuildIncidentEditModal(requestID, title string) slack.ModalViewRequest {
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
	// Make it multiline
	descriptionInput := descriptionBlock.Element.(*slack.PlainTextInputBlockElement)
	descriptionInput.Multiline = true
	descriptionBlock.Optional = true

	return slack.ModalViewRequest{
		Type:            slack.ViewType("modal"),
		Title:           slack.NewTextBlockObject(slack.PlainTextType, "Create Incident", false, false),
		Submit:          slack.NewTextBlockObject(slack.PlainTextType, "Declare", false, false),
		Close:           slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		CallbackID:      "incident_creation_modal",
		PrivateMetadata: requestID, // Store request ID in private metadata
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				titleBlock,
				descriptionBlock,
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
