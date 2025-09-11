package slack

import (
	"fmt"

	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	"github.com/slack-go/slack"
)

// BuildTaskMessage creates Slack blocks for a single task
func BuildTaskMessage(task *model.Task, assigneeUsername string) []slack.Block {
	blocks := []slack.Block{}

	// Header section with title and minimal details
	headerText := fmt.Sprintf("*%s*", task.Title)
	
	// Build status and assignee line
	var statusLine string
	if task.Status == model.TaskStatusCompleted {
		statusLine = "✅ Completed"
	} else {
		statusLine = "🔄 In Progress"
	}
	
	// Add assignee
	if assigneeUsername != "" {
		statusLine += fmt.Sprintf(" • @%s", assigneeUsername)
	} else if task.AssigneeID != "" {
		statusLine += fmt.Sprintf(" • <@%s>", task.AssigneeID)
	}
	
	// Create a single section with title and status line
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, fmt.Sprintf("%s\n_%s_", headerText, statusLine), false, false),
		nil,
		nil,
	))

	// Action buttons
	editButton := slack.NewButtonBlockElement(
		fmt.Sprintf("task_edit_%s", task.ID),
		"edit",
		slack.NewTextBlockObject(slack.PlainTextType, "Edit", false, false),
	)

	if task.Status != model.TaskStatusCompleted {
		completeButton := slack.NewButtonBlockElement(
			fmt.Sprintf("task_complete_%s", task.ID),
			"complete",
			slack.NewTextBlockObject(slack.PlainTextType, "Complete", false, false),
		)
		completeButton.Style = slack.StylePrimary

		blocks = append(blocks, slack.NewActionBlock(
			"",
			editButton,
			completeButton,
		))
	} else {
		uncompleteButton := slack.NewButtonBlockElement(
			fmt.Sprintf("task_uncomplete_%s", task.ID),
			"uncomplete",
			slack.NewTextBlockObject(slack.PlainTextType, "Uncomplete", false, false),
		)
		uncompleteButton.Style = slack.StyleDanger

		blocks = append(blocks, slack.NewActionBlock(
			"",
			editButton,
			uncompleteButton,
		))
	}

	return blocks
}

// BuildTaskListMessage creates Slack blocks for a task list
func BuildTaskListMessage(tasks []*model.Task, incident *model.Incident) []slack.Block {
	blocks := []slack.Block{}

	// Header
	headerText := fmt.Sprintf("📋 *Task List for Incident #%d*", incident.ID)
	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, headerText, false, false),
		nil,
		nil,
	))

	if len(tasks) == 0 {
		// No tasks message
		blocks = append(blocks, slack.NewSectionBlock(
			slack.NewTextBlockObject(slack.PlainTextType, "No tasks have been created yet.", false, false),
			nil,
			nil,
		))
		return blocks
	}

	// Task list
	var taskListText string
	incompletedCount := 0
	completedCount := 0

	for _, task := range tasks {
		statusEmoji := "🔄"
		if task.Status == model.TaskStatusCompleted {
			statusEmoji = "✅"
			completedCount++
		} else {
			incompletedCount++
		}

		// Generate Slack message URL if messageTS is available
		var taskText string
		// Use task's own channel ID if available, otherwise fall back to incident channel
		channelID := task.ChannelID
		if channelID == "" {
			channelID = incident.ChannelID
		}
		
		if task.MessageTS != "" && channelID != "" {
			url := task.GetSlackMessageURL(channelID)
			taskText = fmt.Sprintf("%s <%s|%s>", statusEmoji, url, task.Title)
		} else {
			taskText = fmt.Sprintf("%s %s", statusEmoji, task.Title)
		}

		// Add assignee if present
		if task.AssigneeID != "" {
			taskText = fmt.Sprintf("%s - <@%s>", taskText, task.AssigneeID)
		}

		taskListText += taskText + "\n"
	}

	blocks = append(blocks, slack.NewSectionBlock(
		slack.NewTextBlockObject(slack.MarkdownType, taskListText, false, false),
		nil,
		nil,
	))

	// Summary
	summaryText := fmt.Sprintf("*Incomplete:* %d  |  *Completed:* %d  |  *Total:* %d",
		incompletedCount, completedCount, len(tasks))
	blocks = append(blocks, slack.NewContextBlock(
		"",
		slack.NewTextBlockObject(slack.MarkdownType, summaryText, false, false),
	))

	return blocks
}

// BuildTaskEditModal creates a modal for editing a task
func BuildTaskEditModal(task *model.Task, channelMembers []types.SlackUserID) slack.ModalViewRequest {
	// Title input
	titleInput := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject(slack.PlainTextType, "Task Title", false, false),
		"title",
	)
	titleInput.InitialValue = task.Title

	titleBlock := slack.NewInputBlock(
		"title_block",
		slack.NewTextBlockObject(slack.PlainTextType, "Title", false, false),
		nil,
		titleInput,
	)

	// Description input
	descriptionInput := slack.NewPlainTextInputBlockElement(
		slack.NewTextBlockObject(slack.PlainTextType, "Task Details", false, false),
		"description",
	)
	descriptionInput.Multiline = true
	descriptionInput.InitialValue = task.Description

	descriptionBlock := slack.NewInputBlock(
		"description_block",
		slack.NewTextBlockObject(slack.PlainTextType, "Description", false, false),
		nil,
		descriptionInput,
	)
	descriptionBlock.Optional = true

	// Status selector
	statusOptions := []*slack.OptionBlockObject{
		slack.NewOptionBlockObject(
			string(model.TaskStatusIncompleted),
			slack.NewTextBlockObject(slack.PlainTextType, "Incomplete", false, false),
			nil,
		),
		slack.NewOptionBlockObject(
			string(model.TaskStatusCompleted),
			slack.NewTextBlockObject(slack.PlainTextType, "Completed", false, false),
			nil,
		),
	}

	var initialStatus *slack.OptionBlockObject
	for _, opt := range statusOptions {
		if opt.Value == string(task.Status) {
			initialStatus = opt
			break
		}
	}

	statusSelect := slack.NewOptionsSelectBlockElement(
		slack.OptTypeStatic,
		slack.NewTextBlockObject(slack.PlainTextType, "Select Status", false, false),
		"status",
		statusOptions...,
	)
	if initialStatus != nil {
		statusSelect.InitialOption = initialStatus
	}

	statusBlock := slack.NewInputBlock(
		"status_block",
		slack.NewTextBlockObject(slack.PlainTextType, "Status", false, false),
		nil,
		statusSelect,
	)

	// Assignee selector
	assigneeSelect := slack.NewOptionsSelectBlockElement(
		slack.OptTypeUser,
		slack.NewTextBlockObject(slack.PlainTextType, "Select Assignee", false, false),
		"assignee",
	)
	if task.AssigneeID != "" {
		assigneeSelect.InitialUser = string(task.AssigneeID)
	}

	assigneeBlock := slack.NewInputBlock(
		"assignee_block",
		slack.NewTextBlockObject(slack.PlainTextType, "Assignee", false, false),
		nil,
		assigneeSelect,
	)
	assigneeBlock.Optional = true

	// Build modal
	modal := slack.ModalViewRequest{
		Type:       slack.ViewType("modal"),
		Title:      slack.NewTextBlockObject(slack.PlainTextType, "Edit Task", false, false),
		Submit:     slack.NewTextBlockObject(slack.PlainTextType, "Save", false, false),
		Close:      slack.NewTextBlockObject(slack.PlainTextType, "Cancel", false, false),
		CallbackID: fmt.Sprintf("task_edit_submit_%s", task.ID),
		Blocks: slack.Blocks{
			BlockSet: []slack.Block{
				titleBlock,
				descriptionBlock,
				statusBlock,
				assigneeBlock,
			},
		},
	}

	return modal
}