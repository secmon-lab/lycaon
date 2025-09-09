package llm

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"strconv"
	"text/template"
	"time"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gollem"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/slack-go/slack"
)

// Error tags for categorization
var (
	ErrTagInvalidJSON     = goerr.NewTag("invalid_json")
	ErrTagMissingField    = goerr.NewTag("missing_field")
	ErrTagEmptyResponse   = goerr.NewTag("empty_response")
	ErrTagTemplateFailure = goerr.NewTag("template_failure")
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
	CategoryID  string `json:"category_id"`
}

// CategorySelection represents the LLM response for category selection
// Deprecated: Use IncidentSummary instead which includes category_id
type CategorySelection struct {
	CategoryID string `json:"category_id"`
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

// CategorySelectionTemplateData contains data for category selection template
type CategorySelectionTemplateData struct {
	Categories []model.Category
	Messages   []TemplateMessage
}

// IncidentAnalysisTemplateData contains data for comprehensive incident analysis template
type IncidentAnalysisTemplateData struct {
	Categories []model.Category
	Messages   []TemplateMessage
}

// NewLLMService creates a new LLMService instance
func NewLLMService(llmClient gollem.LLMClient) *LLMService {
	return &LLMService{
		llmClient: llmClient,
	}
}

// AnalyzeIncident performs comprehensive incident analysis in a single LLM call
// Returns title, description, and category_id all at once
func (s *LLMService) AnalyzeIncident(ctx context.Context, messages []slack.Message, categories *model.CategoriesConfig) (*IncidentSummary, error) {
	if len(messages) == 0 {
		return nil, goerr.New("no messages provided for incident analysis")
	}

	// Build template data
	templateData := IncidentAnalysisTemplateData{
		Categories: categories.Categories,
		Messages:   s.buildTemplateMessages(messages),
	}

	// Generate prompt using the unified template
	prompt, err := s.renderIncidentAnalysisTemplate(templateData)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to render incident analysis template",
			goerr.T(ErrTagTemplateFailure))
	}

	// Create session with JSON content type
	session, err := s.llmClient.NewSession(ctx, gollem.WithSessionContentType(gollem.ContentTypeJSON))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create LLM session")
	}

	// Generate response from LLM
	response, err := session.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to generate LLM response")
	}

	// Check if response has content
	if len(response.Texts) == 0 || response.Texts[0] == "" {
		return nil, goerr.New("empty response from LLM",
			goerr.T(ErrTagEmptyResponse))
	}

	// Parse JSON response
	var summary IncidentSummary
	if err := json.Unmarshal([]byte(response.Texts[0]), &summary); err != nil {
		return nil, goerr.Wrap(err, "failed to parse LLM response as JSON",
			goerr.V("response", response.Texts[0]),
			goerr.T(ErrTagInvalidJSON))
	}

	// Validate response
	if summary.Title == "" {
		return nil, goerr.New("LLM response missing title",
			goerr.T(ErrTagMissingField),
			goerr.V("field", "title"))
	}
	if summary.Description == "" {
		return nil, goerr.New("LLM response missing description",
			goerr.T(ErrTagMissingField),
			goerr.V("field", "description"))
	}

	// Validate category ID
	if summary.CategoryID == "" {
		summary.CategoryID = "unknown"
	} else {
		// Check if the selected category is valid
		found := false
		for _, cat := range categories.Categories {
			if cat.ID == summary.CategoryID {
				found = true
				break
			}
		}
		if !found {
			summary.CategoryID = "unknown"
		}
	}

	return &summary, nil
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
	return &IncidentSummaryTemplateData{
		Messages: s.buildTemplateMessages(messages),
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

// SelectIncidentCategory selects an appropriate category for an incident based on messages
func (s *LLMService) SelectIncidentCategory(ctx context.Context, messages []slack.Message, categories *model.CategoriesConfig) (string, error) {
	if len(messages) == 0 {
		return "unknown", goerr.New("no messages provided for category selection")
	}

	if categories == nil || len(categories.Categories) == 0 {
		return "unknown", goerr.New("no categories provided")
	}

	// Build template data
	templateData := &CategorySelectionTemplateData{
		Categories: categories.Categories,
		Messages:   s.buildTemplateMessages(messages),
	}

	// Generate prompt using template
	prompt, err := s.renderCategorySelectionTemplate(templateData)
	if err != nil {
		return "unknown", goerr.Wrap(err, "failed to render category selection template")
	}

	// Create session with JSON content type
	session, err := s.llmClient.NewSession(ctx, gollem.WithSessionContentType(gollem.ContentTypeJSON))
	if err != nil {
		return "unknown", goerr.Wrap(err, "failed to create LLM session")
	}

	// Generate response
	response, err := session.GenerateContent(ctx, gollem.Text(prompt))
	if err != nil {
		return "unknown", goerr.Wrap(err, "failed to generate LLM response")
	}

	// Check if response has content
	if len(response.Texts) == 0 || response.Texts[0] == "" {
		return "unknown", goerr.New("empty response from LLM")
	}

	// Parse JSON response
	var selection CategorySelection
	if err := json.Unmarshal([]byte(response.Texts[0]), &selection); err != nil {
		return "unknown", goerr.Wrap(err, "failed to parse LLM response as JSON",
			goerr.V("response", response.Texts[0]),
		)
	}

	// Validate the selected category exists
	if selection.CategoryID == "" {
		return "unknown", nil
	}

	// Check if the selected category is valid
	for _, cat := range categories.Categories {
		if cat.ID == selection.CategoryID {
			return selection.CategoryID, nil
		}
	}

	// If category not found, return unknown
	return "unknown", nil
}

// buildTemplateMessages converts Slack messages to template messages
func (s *LLMService) buildTemplateMessages(messages []slack.Message) []TemplateMessage {
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

	return templateMessages
}

// renderCategorySelectionTemplate renders the category selection template with data
func (s *LLMService) renderCategorySelectionTemplate(data *CategorySelectionTemplateData) (string, error) {
	// Read template from embedded filesystem
	templateContent, err := templateFS.ReadFile("templates/incident_category.md")
	if err != nil {
		return "", goerr.Wrap(err, "failed to read category selection template")
	}

	// Parse template
	tmpl, err := template.New("incident_category").Parse(string(templateContent))
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse category selection template")
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute category selection template")
	}

	return buf.String(), nil
}

// renderIncidentAnalysisTemplate renders the comprehensive incident analysis template
func (s *LLMService) renderIncidentAnalysisTemplate(data IncidentAnalysisTemplateData) (string, error) {
	// Load template from embedded filesystem
	templateContent, err := templateFS.ReadFile("templates/incident_analysis.md")
	if err != nil {
		return "", goerr.Wrap(err, "failed to read incident analysis template")
	}

	// Parse template
	tmpl, err := template.New("incident_analysis").Parse(string(templateContent))
	if err != nil {
		return "", goerr.Wrap(err, "failed to parse incident analysis template")
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", goerr.Wrap(err, "failed to execute incident analysis template")
	}

	return buf.String(), nil
}

// parseSlackTimestamp parses Slack timestamp format (Unix timestamp with microseconds)
func parseSlackTimestamp(timestamp string) (time.Time, error) {
	// Slack timestamps are in format "1234567890.123456"
	ts, err := strconv.ParseFloat(timestamp, 64)
	if err != nil {
		return time.Time{}, goerr.Wrap(err, "failed to parse timestamp",
			goerr.V("timestamp", timestamp),
		)
	}
	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	return time.Unix(sec, nsec), nil
}
