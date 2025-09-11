package slack

import (
	"context"
	"regexp"
	"strings"

	"github.com/m-mizutani/ctxlog"
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/interfaces"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
	slackblocks "github.com/secmon-lab/lycaon/pkg/service/slack"
	"github.com/secmon-lab/lycaon/pkg/utils/async"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

var (
	// taskCommandPattern matches task command patterns like "<@BOT123> t" or "<@BOT123> task" or "@lycaon t" or "@lycaon task"
	taskCommandPattern = regexp.MustCompile(`(<@\w+>|@\w+)\s+(t|task)(\s|$)`)
	// taskTitlePattern matches task commands with titles like "<@BOT123> t 'task title'" or "<@BOT123> task 'task title'"
	taskTitlePattern = regexp.MustCompile(`(<@\w+>|@\w+)\s+(t|task)\s+(.+)`)
)

// EventHandler handles Slack events
type EventHandler struct {
	messageUC   interfaces.SlackMessage
	taskUC      interfaces.Task
	incidentUC  interfaces.Incident
	slackClient interfaces.SlackClient
}

// NewEventHandler creates a new event handler
func NewEventHandler(ctx context.Context, messageUC interfaces.SlackMessage, taskUC interfaces.Task, incidentUC interfaces.Incident, slackClient interfaces.SlackClient) *EventHandler {
	return &EventHandler{
		messageUC:   messageUC,
		taskUC:      taskUC,
		incidentUC:  incidentUC,
		slackClient: slackClient,
	}
}

// HandleEvent handles a Slack event
func (h *EventHandler) HandleEvent(ctx context.Context, event *slackevents.EventsAPIEvent) error {
	if event == nil {
		return goerr.New("event is nil")
	}

	ctxlog.From(ctx).Debug("Handling Slack event",
		"type", event.Type,
		"innerEvent", event.InnerEvent.Type,
	)

	// Handle different event types
	switch ev := event.InnerEvent.Data.(type) {
	case *slackevents.MessageEvent:
		return h.handleMessageEvent(ctx, ev)

	case *slackevents.AppMentionEvent:
		return h.handleAppMentionEvent(ctx, ev)

	default:
		ctxlog.From(ctx).Debug("Unhandled event type",
			"type", event.InnerEvent.Type,
		)
		return nil
	}
}

// handleMessageEvent handles message events
// Controller responsibility: Basic validation, then async dispatch for processing
func (h *EventHandler) handleMessageEvent(ctx context.Context, event *slackevents.MessageEvent) error {
	logger := ctxlog.From(ctx)

	// Skip bot messages to prevent loops
	if event.BotID != "" {
		logger.Debug("Skipping bot message", "botID", event.BotID)
		return nil
	}

	// Skip messages without text
	if event.Text == "" {
		logger.Debug("Skipping empty message")
		return nil
	}

	// Skip thread messages (optional - depends on requirements)
	if event.ThreadTimeStamp != "" && event.ThreadTimeStamp != event.TimeStamp {
		logger.Debug("Skipping thread message",
			"threadTS", event.ThreadTimeStamp,
			"messageTS", event.TimeStamp,
		)
		return nil
	}

	logger.Info("Processing message event",
		"user", event.User,
		"channel", event.Channel,
		"text", event.Text,
		"ts", event.TimeStamp,
	)

	// Dispatch message processing asynchronously to return 200 immediately
	backgroundCtx := async.NewBackgroundContext(ctx)
	async.Dispatch(backgroundCtx, func(asyncCtx context.Context) error {
		if err := h.messageUC.ProcessMessage(asyncCtx, event); err != nil {
			logger.Error("Message processing failed",
				"error", err,
				"user", event.User,
				"channel", event.Channel,
			)
			// Log error but don't propagate - async processing
		}
		return nil
	})

	// Return immediately to send 200 response to Slack
	logger.Debug("Message dispatched for async processing",
		"user", event.User,
		"channel", event.Channel,
	)
	return nil
}

// handleAppMentionEvent handles app mention events
// Controller responsibility: Basic validation, then async dispatch for all processing
func (h *EventHandler) handleAppMentionEvent(ctx context.Context, event *slackevents.AppMentionEvent) error {
	logger := ctxlog.From(ctx)

	logger.Info("App mentioned",
		"user", event.User,
		"channel", event.Channel,
		"text", event.Text,
		"ts", event.TimeStamp,
	)

	// Dispatch all app mention processing asynchronously to return 200 immediately
	backgroundCtx := async.NewBackgroundContext(ctx)
	async.Dispatch(backgroundCtx, func(asyncCtx context.Context) error {
		h.processAppMentionAsync(asyncCtx, event)
		return nil
	})

	// Return immediately to send 200 response to Slack
	logger.Debug("App mention dispatched for async processing",
		"user", event.User,
		"channel", event.Channel,
	)
	return nil
}

// processAppMentionAsync processes app mention events asynchronously
// UseCase orchestration: Handle message saving, task commands, and incident creation
func (h *EventHandler) processAppMentionAsync(ctx context.Context, event *slackevents.AppMentionEvent) {
	logger := ctxlog.From(ctx)

	// Convert AppMentionEvent to MessageEvent for processing
	messageEvent := &slackevents.MessageEvent{
		Type:            "message",
		User:            event.User,
		Text:            event.Text,
		TimeStamp:       event.TimeStamp,
		ThreadTimeStamp: event.ThreadTimeStamp,
		Channel:         event.Channel,
	}

	// Save the message
	if err := h.messageUC.ProcessMessage(ctx, messageEvent); err != nil {
		logger.Error("Failed to save message", "error", err)
		return
	}

	// Convert to domain model for incident trigger check
	message := &model.Message{
		ID:        types.MessageID(messageEvent.ClientMsgID),
		UserID:    types.SlackUserID(event.User),
		ChannelID: types.ChannelID(event.Channel),
		Text:      event.Text,
		EventTS:   types.EventTS(event.TimeStamp),
	}

	// First check if it's a basic incident trigger (before any heavy processing)
	if h.messageUC.IsBasicIncidentTrigger(ctx, message) {
		// Send immediate context message to acknowledge the command
		if err := h.messageUC.SendProcessingMessage(ctx, event.Channel, event.TimeStamp); err != nil {
			logger.Warn("Failed to send processing message",
				"error", err,
				"channel", event.Channel,
			)
		}
	}

	// Check for task commands
	if h.isTaskCommand(event.Text) {
		if err := h.handleTaskCommand(ctx, event); err != nil {
			logger.Error("Task command handling failed", "error", err)
		}
		return
	}

	// Check if message triggers incident creation (this may do LLM analysis)
	cmd := h.messageUC.ParseIncidentCommand(ctx, message)
	if cmd.IsIncidentTrigger {
		ctxlog.From(ctx).Info("Incident trigger detected from mention",
			"user", event.User,
			"channel", event.Channel,
			"title", cmd.Title,
		)

		// Send incident creation prompt with title and description
		if err := h.messageUC.SendIncidentMessage(ctx, event.Channel, event.TimeStamp, cmd.Title, cmd.Description, cmd.CategoryID); err != nil {
			ctxlog.From(ctx).Error("Failed to send incident prompt",
				"error", err,
				"channel", event.Channel,
			)
		}
	}
	// No response for non-incident mentions - just save the message
}

// isTaskCommand checks if the message is a task command
func (h *EventHandler) isTaskCommand(text string) bool {
	// Match patterns like "<@BOT123> t" or "<@BOT123> task" or "@lycaon t" or "@lycaon task"
	return taskCommandPattern.MatchString(text)
}

// handleTaskCommand handles task-related commands
func (h *EventHandler) handleTaskCommand(ctx context.Context, event *slackevents.AppMentionEvent) error {
	logger := ctxlog.From(ctx)

	// Find incident for this channel
	incident, err := h.findIncidentByChannel(ctx, types.ChannelID(event.Channel))
	if err != nil {
		logger.Warn("Failed to find incident for channel", "error", err, "channel", event.Channel)
		return h.sendTaskErrorMessage(ctx, event.Channel, event.TimeStamp, "Please create an incident first.")
	}

	// Parse task command
	taskTitle := h.parseTaskTitle(event.Text)

	if taskTitle == "" {
		// No task title provided, show task list
		return h.showTaskList(ctx, event, incident)
	} else {
		// Create new task
		return h.createTask(ctx, event, incident, taskTitle)
	}
}

// findIncidentByChannel finds an incident by channel ID
func (h *EventHandler) findIncidentByChannel(ctx context.Context, channelID types.ChannelID) (*model.Incident, error) {
	return h.incidentUC.GetIncidentByChannelID(ctx, channelID)
}

// parseTaskTitle extracts task title from the command text
func (h *EventHandler) parseTaskTitle(text string) string {
	// Remove mentions and task command, extract the title
	// Pattern: <@BOT123> t "task title" or <@BOT123> task "task title" or @lycaon t "task title" or @lycaon task "task title"
	matches := taskTitlePattern.FindStringSubmatch(text)
	if len(matches) > 3 {
		title := strings.TrimSpace(matches[3])
		// Remove quotes if present
		if strings.HasPrefix(title, `"`) && strings.HasSuffix(title, `"`) {
			title = strings.Trim(title, `"`)
		}
		return title
	}
	return ""
}

// showTaskList displays the task list for an incident
func (h *EventHandler) showTaskList(ctx context.Context, event *slackevents.AppMentionEvent, incident *model.Incident) error {
	logger := ctxlog.From(ctx)

	// Get tasks for the incident
	tasks, err := h.taskUC.ListTasks(ctx, incident.ID)
	if err != nil {
		logger.Error("Failed to list tasks", "error", err, "incidentID", incident.ID)
		return h.sendTaskErrorMessage(ctx, event.Channel, event.TimeStamp, "Failed to retrieve task list.")
	}

	// Build task list message
	blocks := slackblocks.BuildTaskListMessage(tasks, incident)

	// Send message
	_, _, err = h.slackClient.PostMessage(ctx, event.Channel, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		return goerr.Wrap(err, "failed to post task list message")
	}

	return nil
}

// createTask creates a new task
func (h *EventHandler) createTask(ctx context.Context, event *slackevents.AppMentionEvent, incident *model.Incident, title string) error {
	logger := ctxlog.From(ctx)

	// Create task (pass empty messageTS initially, will be updated after posting)
	task, err := h.taskUC.CreateTask(ctx, incident.ID, title, types.SlackUserID(event.User), types.ChannelID(event.Channel), "")
	if err != nil {
		logger.Error("Failed to create task", "error", err, "title", title)
		return h.sendTaskErrorMessage(ctx, event.Channel, event.TimeStamp, "Failed to create task.")
	}

	// Build task message
	blocks := slackblocks.BuildTaskMessage(task, "")

	// Post task message
	_, timestamp, err := h.slackClient.PostMessage(ctx, event.Channel, slack.MsgOptionBlocks(blocks...))
	if err != nil {
		logger.Error("Failed to post task message", "error", err)
		return goerr.Wrap(err, "failed to post task message")
	}

	// Update task with message timestamp for link generation
	updateReq := interfaces.TaskUpdateRequest{
		MessageTS: &timestamp,
	}
	if _, err := h.taskUC.UpdateTask(ctx, task.ID, updateReq); err != nil {
		logger.Warn("Failed to update task with message timestamp", "error", err, "taskID", task.ID)
	}

	logger.Info("Task created successfully", "taskID", task.ID, "title", title, "incidentID", incident.ID)
	return nil
}

// sendTaskErrorMessage sends an error message for task operations as a thread reply
func (h *EventHandler) sendTaskErrorMessage(ctx context.Context, channel, threadTS, message string) error {
	_, _, err := h.slackClient.PostMessage(ctx, channel,
		slack.MsgOptionText(message, false),
		slack.MsgOptionTS(threadTS),
	)
	return err
}
