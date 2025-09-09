package slack

import (
	"fmt"

	"github.com/secmon-lab/lycaon/pkg/domain/model"
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
func (b *BlockBuilder) BuildIncidentCreatedBlocks(channelName, channelID, title, categoryID string, categories *model.CategoriesConfig) []slack.Block {
	channelLink := fmt.Sprintf("<#%s>", channelID)
	category := categories.FindCategoryByIDWithFallback(categoryID)

	var message string
	if title != "" {
		message = fmt.Sprintf("✅ Incident channel %s has been created for: *%s*\n*Category:* %s", channelLink, title, category.Name)
	} else {
		message = fmt.Sprintf("✅ Incident channel %s has been created\n*Category:* %s", channelLink, category.Name)
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
func (b *BlockBuilder) BuildIncidentChannelWelcomeBlocks(incidentID int, originChannelName string, createdBy string, description string, categoryID string, categories *model.CategoriesConfig) []slack.Block {
	category := categories.FindCategoryByIDWithFallback(categoryID)

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
				fmt.Sprintf("This incident was created from *#%s* by <@%s>\n*Category:* %s", originChannelName, createdBy, category.Name),
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
func (b *BlockBuilder) BuildIncidentEditModal(requestID, title, description, categoryID string, categories []model.Category) slack.ModalViewRequest {
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
