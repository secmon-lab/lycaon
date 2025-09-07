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
					"New Incident",
					false,
					false,
				),
			).WithStyle(slack.StyleDanger),
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
func (b *BlockBuilder) BuildIncidentChannelWelcomeBlocks(incidentID int, originChannelName string, createdBy string) []slack.Block {
	return []slack.Block{
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
	}
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
