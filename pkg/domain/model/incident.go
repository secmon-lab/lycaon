package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// Incident represents an incident in the system
type Incident struct {
	ID                types.IncidentID  // Incident serial number (e.g., 1, 2, 3)
	Title             string            // Incident title (e.g., "database outage")
	Description       string            // Incident description (optional)
	CategoryID        string            // Category ID (e.g., "security_incident", "system_failure")
	ChannelID         types.ChannelID   // Dedicated incident channel ID
	ChannelName       types.ChannelName // Dedicated incident channel name (e.g., "inc-1-database-outage")
	OriginChannelID   types.ChannelID   // Origin channel ID where incident was created
	OriginChannelName types.ChannelName // Origin channel name where incident was created
	TeamID            types.TeamID      // Slack workspace/team ID
	CreatedBy         types.SlackUserID // Slack user ID who created the incident
	CreatedAt         time.Time         // Creation timestamp
	// Status management fields
	Status        types.IncidentStatus // Current status of the incident
	Lead          types.SlackUserID    // Incident lead (Slack user ID)
	InitialTriage bool                 // Whether the incident started with Triage status
}

// CreateIncidentRequest represents parameters for creating an incident
type CreateIncidentRequest struct {
	Title             string
	Description       string
	CategoryID        string
	OriginChannelID   string
	OriginChannelName string
	TeamID            string
	CreatedBy         string
	InitialTriage     bool // Whether to start with Triage status
}

// NewIncident creates a new Incident instance
func NewIncident(prefix string, id types.IncidentID, title, description, categoryID string, originChannelID types.ChannelID, originChannelName types.ChannelName, teamID types.TeamID, createdBy types.SlackUserID, initialTriage bool) (*Incident, error) {
	if id <= 0 {
		return nil, goerr.New("incident ID must be positive")
	}
	if originChannelID == "" {
		return nil, goerr.New("origin channel ID is required")
	}
	if originChannelName == "" {
		return nil, goerr.New("origin channel name is required")
	}
	if createdBy == "" {
		return nil, goerr.New("creator user ID is required")
	}

	channelName := formatIncidentChannelName(prefix, id, title)

	// Set initial status based on triage flag
	var initialStatus types.IncidentStatus
	if initialTriage {
		initialStatus = types.IncidentStatusTriage
	} else {
		initialStatus = types.IncidentStatusHandling
	}

	now := time.Now()

	return &Incident{
		ID:                id,
		Title:             title,
		Description:       description,
		CategoryID:        categoryID,
		ChannelName:       types.ChannelName(channelName),
		OriginChannelID:   originChannelID,
		OriginChannelName: originChannelName,
		TeamID:            teamID,
		CreatedBy:         createdBy,
		CreatedAt:         now,
		Status:            initialStatus,
		Lead:              createdBy, // Creator becomes initial lead
		InitialTriage:     initialTriage,
	}, nil
}

// formatIncidentChannelName creates a Slack-compatible channel name from incident ID and title
func formatIncidentChannelName(prefix string, id types.IncidentID, title string) string {
	// Use "inc" as fallback if prefix is empty to avoid channel names starting with "-"
	if prefix == "" {
		prefix = "inc"
	}
	baseChannelName := fmt.Sprintf("%s-%d", prefix, id)

	if title == "" {
		return baseChannelName
	}

	// Sanitize title for Slack channel name
	sanitized := sanitizeForSlackChannelName(title)
	if sanitized == "" {
		return baseChannelName
	}

	fullName := fmt.Sprintf("%s-%s", baseChannelName, sanitized)

	// Slack channel names must be 80 characters or less
	if len(fullName) > 80 {
		fullName = fullName[:80]
		// Ensure the truncated name doesn't end with a hyphen
		fullName = strings.TrimRight(fullName, "-")
	}

	return fullName
}

// sanitizeForSlackChannelName converts text to be compatible with Slack channel names
// Names must be lowercase, without spaces or periods
// Symbols are replaced with hyphens, multibyte characters are preserved
func sanitizeForSlackChannelName(text string) string {
	if text == "" {
		return ""
	}

	var result strings.Builder

	// Process each rune
	for _, r := range text {
		switch {
		case unicode.IsSpace(r) || r == '.':
			// Replace spaces and periods with hyphens
			result.WriteRune('-')
		case (r >= 'A' && r <= 'Z'):
			// Convert uppercase to lowercase
			result.WriteRune(unicode.ToLower(r))
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_':
			// Keep ASCII letters, numbers, hyphens, underscores
			result.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			// Keep multibyte letters and numbers (Japanese, Chinese, etc.)
			result.WriteRune(r)
		default:
			// Replace symbols and other characters with hyphens
			result.WriteRune('-')
		}
	}

	text = result.String()

	// Remove consecutive hyphens
	re := regexp.MustCompile(`-+`)
	text = re.ReplaceAllString(text, "-")

	// Remove leading/trailing hyphens
	text = strings.Trim(text, "-")

	return text
}
