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
	SeverityID  string `json:"severity_id"`
}

// ChannelInfo represents Slack channel information for LLM context
type ChannelInfo struct {
	Name        string
	Topic       string
	Purpose     string
	IsPrivate   bool
	MemberCount int
}

// TemplateMessage represents a message for template rendering
type TemplateMessage struct {
	Timestamp string
	User      string
	Text      string
}

// IncidentAnalysisTemplateData contains data for comprehensive incident analysis template
type IncidentAnalysisTemplateData struct {
	Categories       []model.Category
	Severities       []model.Severity
	Messages         []TemplateMessage
	AdditionalPrompt string
	ChannelInfo      *ChannelInfo
}

// NewLLMService creates a new LLMService instance
func NewLLMService(llmClient gollem.LLMClient) *LLMService {
	return &LLMService{
		llmClient: llmClient,
	}
}

// AnalyzeIncident performs comprehensive incident analysis in a single LLM call
// Returns title, description, category_id, and severity_id all at once
func (s *LLMService) AnalyzeIncident(ctx context.Context, messages []slack.Message, categories *model.CategoriesConfig, severities *model.SeveritiesConfig) (*IncidentSummary, error) {
	if len(messages) == 0 {
		return nil, goerr.New("no messages provided for incident analysis")
	}

	// Build template data without additional context
	templateData := IncidentAnalysisTemplateData{
		Categories: categories.Categories,
		Messages:   s.buildTemplateMessages(messages),
	}
	if severities != nil {
		templateData.Severities = severities.Severities
	}

	return s.analyze(ctx, templateData, categories, severities)
}

// AnalyzeIncidentWithContext performs comprehensive incident analysis with additional context
// This method extends AnalyzeIncident to include additional prompt and channel information
func (s *LLMService) AnalyzeIncidentWithContext(ctx context.Context, messages []slack.Message, categories *model.CategoriesConfig, severities *model.SeveritiesConfig, additionalPrompt string, channelInfo *slack.Channel) (*IncidentSummary, error) {
	if len(messages) == 0 {
		return nil, goerr.New("no messages provided for incident analysis")
	}

	// Build template data with additional context
	templateData := IncidentAnalysisTemplateData{
		Categories:       categories.Categories,
		Messages:         s.buildTemplateMessages(messages),
		AdditionalPrompt: additionalPrompt,
		ChannelInfo:      s.buildChannelInfo(channelInfo),
	}
	if severities != nil {
		templateData.Severities = severities.Severities
	}

	return s.analyze(ctx, templateData, categories, severities)
}

// analyze performs the common LLM analysis logic
// This is the shared implementation used by both AnalyzeIncident and AnalyzeIncidentWithContext
func (s *LLMService) analyze(ctx context.Context, templateData IncidentAnalysisTemplateData, categories *model.CategoriesConfig, severities *model.SeveritiesConfig) (*IncidentSummary, error) {
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
	if summary.CategoryID == "" || !categories.IsValidCategoryID(summary.CategoryID) {
		summary.CategoryID = "unknown"
	}

	// Validate severity ID
	if summary.SeverityID != "" && severities != nil {
		if !severities.IsValidSeverityID(summary.SeverityID) {
			summary.SeverityID = "" // Clear invalid severity ID
		}
	}

	return &summary, nil
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

		// Use Username if available, otherwise fall back to User ID
		user := msg.Username
		if user == "" {
			user = msg.User
		}

		templateMessages = append(templateMessages, TemplateMessage{
			Timestamp: timestamp,
			User:      user,
			Text:      msg.Text,
		})
	}

	return templateMessages
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

// buildChannelInfo converts Slack channel to ChannelInfo for template rendering
func (s *LLMService) buildChannelInfo(channel *slack.Channel) *ChannelInfo {
	if channel == nil {
		return nil
	}
	return &ChannelInfo{
		Name:        channel.Name,
		Topic:       channel.Topic.Value,
		Purpose:     channel.Purpose.Value,
		IsPrivate:   channel.IsPrivate,
		MemberCount: channel.NumMembers,
	}
}
