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
	switch task.Status {
	case model.TaskStatusTodo:
		statusLine = "📝 To Do"
	case model.TaskStatusFollowUp:
		statusLine = "🔄 Follow Up"
	case model.TaskStatusCompleted:
		statusLine = "✅ Completed"
	default:
		statusLine = "❓ Unknown"
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

	// Action buttons with structured ActionID
	actionElements := buildTaskActionButtons(task)
	if len(actionElements) > 0 {
		blocks = append(blocks, slack.NewActionBlock("", actionElements...))
	}

	return blocks
}

// buildTaskActionButtons creates action buttons with structured ActionIDs
func buildTaskActionButtons(task *model.Task) []slack.BlockElement {
	var buttons []slack.BlockElement

	// Status change buttons based on current status
	switch task.Status {
	case model.TaskStatusTodo:
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				fmt.Sprintf("task:status_change:%s:follow_up", task.ID),
				"mark_follow_up",
				slack.NewTextBlockObject(slack.PlainTextType, "🔄 Mark as Follow Up", false, false),
			),
			slack.NewButtonBlockElement(
				fmt.Sprintf("task:status_change:%s:completed", task.ID),
				"mark_completed",
				slack.NewTextBlockObject(slack.PlainTextType, "✅ Mark as Completed", false, false),
			),
		)
		// Set style for primary action
		buttons[1].(*slack.ButtonBlockElement).Style = slack.StylePrimary

	case model.TaskStatusFollowUp:
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				fmt.Sprintf("task:status_change:%s:todo", task.ID),
				"mark_todo",
				slack.NewTextBlockObject(slack.PlainTextType, "📝 Mark as To Do", false, false),
			),
			slack.NewButtonBlockElement(
				fmt.Sprintf("task:status_change:%s:completed", task.ID),
				"mark_completed",
				slack.NewTextBlockObject(slack.PlainTextType, "✅ Mark as Completed", false, false),
			),
		)
		// Set style for primary action
		buttons[1].(*slack.ButtonBlockElement).Style = slack.StylePrimary

	case model.TaskStatusCompleted:
		buttons = append(buttons,
			slack.NewButtonBlockElement(
				fmt.Sprintf("task:status_change:%s:todo", task.ID),
				"mark_todo",
				slack.NewTextBlockObject(slack.PlainTextType, "📝 Mark as To Do", false, false),
			),
			slack.NewButtonBlockElement(
				fmt.Sprintf("task:status_change:%s:follow_up", task.ID),
				"mark_follow_up",
				slack.NewTextBlockObject(slack.PlainTextType, "🔄 Mark as Follow Up", false, false),
			),
		)
		// Set style for todo action
		buttons[0].(*slack.ButtonBlockElement).Style = slack.StyleDanger
	}

	// Common edit button
	editButton := slack.NewButtonBlockElement(
		fmt.Sprintf("task:edit:%s", task.ID),
		"edit",
		slack.NewTextBlockObject(slack.PlainTextType, "Edit", false, false),
	)
	buttons = append(buttons, editButton)

	return buttons
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
			string(model.TaskStatusTodo),
			slack.NewTextBlockObject(slack.PlainTextType, "To Do", false, false),
			nil,
		),
		slack.NewOptionBlockObject(
			string(model.TaskStatusFollowUp),
			slack.NewTextBlockObject(slack.PlainTextType, "Follow Up", false, false),
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
		CallbackID: fmt.Sprintf("task_edit_submit:%d:%s", task.IncidentID, task.ID),
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
