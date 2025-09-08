package llm

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/slack-go/slack"
)

//go:embed templates/*.md
var templateFS embed.FS

// LLMService handles LLM operations for various purposes
type LLMService struct {
	llmClient gollem.LLMClient
}

// IncidentSummary represents the structured response from LLM for incident creation
type IncidentSummary struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// TemplateMessage represents a message for template rendering
type TemplateMessage struct {
	Timestamp string
	User      string
	Text      string
}

// IncidentSummaryTemplateData contains data for incident summary template
type IncidentSummaryTemplateData struct {
	Messages []TemplateMessage
}

// NewLLMService creates a new LLMService instance
func NewLLMService(llmClient gollem.LLMClient) *LLMService {
	return &LLMService{
		llmClient: llmClient,
	}
}

// GenerateIncidentSummary generates title and description for an incident based on message history
func (s *LLMService) GenerateIncidentSummary(ctx context.Context, messages []slack.Message) (*IncidentSummary, error) {
	if len(messages) == 0 {
		return nil, goerr.New("no messages provided for summary generation")
	}

	// Build template data from messages
	templateData := s.buildTemplateData(messages)

	// Generate prompt using template
	prompt, err := s.renderIncidentSummaryTemplate(templateData)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to render incident summary template")
	}

	// Create session with JSON content type
	session, err := s.llmClient.NewSession(ctx, gollem.WithSessionContentType(gollem.ContentTypeJSON))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create LLM session")
	}

	// Generate response
	response, err := session.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate LLM response")
	}

	// Check if response has content
	if len(response.Texts) == 0 || response.Texts[0] == "" {
		return nil, goerr.New("empty response from LLM")
	}

	// Parse JSON response
	var summary IncidentSummary
	if err := json.Unmarshal([]byte(response.Texts[0]), &summary); err != nil {
		return nil, goerr.Wrap(err, "failed to parse LLM response as JSON",
			goerr.V("response", response.Texts[0]),
		)
	}

	// Validate required fields
	if summary.Title == "" {
		return nil, goerr.New("LLM response missing required title field")
	}
	if summary.Description == "" {
		return nil, goerr.New("LLM response missing required description field")
	}

	return &summary, nil
}

// buildTemplateData converts Slack messages to template data structure
func (s *LLMService) buildTemplateData(messages []slack.Message) *IncidentSummaryTemplateData {
	templateMessages := make([]TemplateMessage, 0, len(messages))

	for _, msg := range messages {
		// Skip empty messages
		if msg.Text == "" {
			continue
		}

		// Format timestamp for display
		timestamp := ""
		if msg.Timestamp != "" {
			if ts, err := parseSlackTimestamp(msg.Timestamp); err == nil {
				timestamp = ts.Format("15:04")
			} else {
				timestamp = msg.Timestamp // Fallback to raw timestamp
			}
		}

		templateMessages = append(templateMessages, TemplateMessage{
			Timestamp: timestamp,
			User:      msg.User,
			Text:      msg.Text,
		})
	}

	return &IncidentSummaryTemplateData{
		Messages: templateMessages,
	}
}

// renderIncidentSummaryTemplate renders the incident summary template with data
func (s *LLMService) renderIncidentSummaryTemplate(data *IncidentSummaryTemplateData) (string, error) {
	// Read template from embedded filesystem
	templateContent, err := templateFS.ReadFile("templates/incident_summary.md")
	if err != nil {
		return "", goerr.Wrap(err, "failed to read incident summary template")
	}

	// Parse template
	tmpl, err := template.New("incident_summary").Parse(string(templateContent))
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse incident summary template")
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute incident summary template")
	}

	return buf.String(), nil
}

// parseSlackTimestamp parses Slack timestamp format (Unix timestamp with microseconds)
func parseSlackTimestamp(timestamp string) (time.Time, error) {
	// Slack timestamps are in format "1234567890.123456"
	parts := strings.Split(timestamp, ".")
	if len(parts) != 2 {
		return time.Time{}, goerr.New("invalid timestamp format",
			goerr.V("timestamp", timestamp),
		)
	}

	// Parse Unix timestamp (seconds part only)
	unix, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, goerr.Wrap(err, "failed to parse timestamp",
			goerr.V("timestamp", timestamp),
		)
	}

	return time.Unix(unix, 0), nil
}
