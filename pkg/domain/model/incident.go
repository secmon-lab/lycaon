package model

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/m-mizutani/goerr/v2"
)

// Incident represents an incident in the system
type Incident struct {
	ID                int       // Incident serial number (e.g., 1, 2, 3)
	Title             string    // Incident title (e.g., "database outage")
	ChannelID         string    // Dedicated incident channel ID
	ChannelName       string    // Dedicated incident channel name (e.g., "inc-1-database-outage")
	OriginChannelID   string    // Origin channel ID where incident was created
	OriginChannelName string    // Origin channel name where incident was created
	CreatedBy         string    // Slack user ID who created the incident
	CreatedAt         time.Time // Creation timestamp
}

// NewIncident creates a new Incident instance
func NewIncident(id int, title, originChannelID, originChannelName, createdBy string) (*Incident, error) {
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

	channelName := formatIncidentChannelName(id, title)

	return &Incident{
		ID:                id,
		Title:             title,
		ChannelName:       channelName,
		OriginChannelID:   originChannelID,
		OriginChannelName: originChannelName,
		CreatedBy:         createdBy,
		CreatedAt:         time.Now(),
	}, nil
}

// formatIncidentChannelName creates a Slack-compatible channel name from incident ID and title
func formatIncidentChannelName(id int, title string) string {
	baseChannelName := fmt.Sprintf("inc-%d", id)

	if title == "" {
		return baseChannelName
	}

	// Sanitize title for Slack channel name
	sanitized := sanitizeForSlackChannelName(title)
	if sanitized == "" {
		return baseChannelName
	}

	return fmt.Sprintf("%s-%s", baseChannelName, sanitized)
}

// sanitizeForSlackChannelName converts text to be compatible with Slack channel names
// Names must be lowercase, without spaces or periods, and can't be longer than 80 characters
// Symbols are replaced with hyphens, multibyte characters are preserved
// Length limit is interpreted as 80 bytes (safe side)
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

	// Limit length (80 bytes total, "inc-XXX-" is 8 bytes, so 72 bytes left for title)
	maxTitleBytes := 72
	if len(text) > maxTitleBytes {
		// Truncate at byte boundary, not character boundary for safety
		text = text[:maxTitleBytes]
	}

	// Remove trailing hyphens again after truncation
	text = strings.TrimRight(text, "-")

	return text
}
