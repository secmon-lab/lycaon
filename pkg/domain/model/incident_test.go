package model_test

import (
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

func TestNewIncident(t *testing.T) {
	t.Run("Valid incident creation", func(t *testing.T) {
		incident, err := model.NewIncident(
			"inc", // prefix
			types.IncidentID(1),
			"database outage",
			"test description",
			"system_failure",
			types.ChannelID("C12345"),
			types.ChannelName("general"),
			types.TeamID("T12345"),
			types.SlackUserID("U67890"),
			false, // initialTriage
		)
		gt.NoError(t, err).Required()
		gt.Equal(t, 1, incident.ID)
		gt.Equal(t, "inc-1-database-outage", incident.ChannelName)
		gt.Equal(t, "database outage", incident.Title)
		gt.Equal(t, "C12345", incident.OriginChannelID)
		gt.Equal(t, "general", incident.OriginChannelName)
		gt.Equal(t, "U67890", incident.CreatedBy)
		gt.True(t, time.Since(incident.CreatedAt) < time.Second)
		gt.Equal(t, "", incident.ChannelID) // Not set yet
	})

	t.Run("Invalid ID", func(t *testing.T) {
		incident, err := model.NewIncident(
			"inc", // prefix
			0,
			"test",
			"",
			"unknown",
			types.ChannelID("C12345"),
			types.ChannelName("general"),
			types.TeamID("T12345"),
			types.SlackUserID("U67890"),
			false, // initialTriage
		)
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("ID must be positive")
	})

	t.Run("Empty origin channel ID", func(t *testing.T) {
		incident, err := model.NewIncident(
			"inc", // prefix
			1,
			"test",
			"",
			"unknown",
			types.ChannelID(""),
			types.ChannelName("general"),
			types.TeamID("T12345"),
			types.SlackUserID("U67890"),
			false, // initialTriage
		)
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("origin channel ID is required")
	})

	t.Run("Empty origin channel name", func(t *testing.T) {
		incident, err := model.NewIncident(
			"inc", // prefix
			1,
			"test",
			"",
			"unknown",
			types.ChannelID("C12345"),
			types.ChannelName(""),
			types.TeamID("T12345"),
			types.SlackUserID("U67890"),
			false, // initialTriage
		)
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("origin channel name is required")
	})

	t.Run("Empty creator", func(t *testing.T) {
		incident, err := model.NewIncident(
			"inc", // prefix
			1,
			"test",
			"",
			"unknown",
			types.ChannelID("C12345"),
			types.ChannelName("general"),
			types.TeamID("T12345"),
			types.SlackUserID(""),
			false, // initialTriage
		)
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("creator user ID is required")
	})
}

func TestIncidentChannelNameFormatting(t *testing.T) {
	testCases := []struct {
		id           int
		expectedName string
	}{
		{1, "inc-1"},
		{10, "inc-10"},
		{99, "inc-99"},
		{100, "inc-100"},
		{999, "inc-999"},
		{1000, "inc-1000"},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedName, func(t *testing.T) {
			incident, err := model.NewIncident(
				"inc", // prefix
				types.IncidentID(tc.id),
				"",
				"",
				"unknown",
				types.ChannelID("C12345"),
				types.ChannelName("general"),
				types.TeamID("T12345"),
				types.SlackUserID("U67890"),
				false, // initialTriage
			)
			gt.NoError(t, err).Required()
			gt.Equal(t, types.ChannelName(tc.expectedName), incident.ChannelName)
		})
	}
}

func TestIncidentTitleInChannelName(t *testing.T) {
	testCases := []struct {
		name         string
		id           int
		title        string
		expectedName string
	}{
		{"Empty title", 1, "", "inc-1"},
		{"Simple title", 1, "database outage", "inc-1-database-outage"},
		{"Title with spaces", 1, "database is down", "inc-1-database-is-down"},
		{"Title with special characters", 1, "API server failure!", "inc-1-api-server-failure"},
		{"Long title", 1, "this is a very long title that should be truncated somehow", "inc-1-this-is-a-very-long-title-that-should-be-truncated-somehow"},
		{"Title with multiple spaces", 1, "   multiple   spaces   ", "inc-1-multiple-spaces"},
		{"Title with unicode", 1, "database-error-unicode", "inc-1-database-error-unicode"},
		{"Mixed case title", 1, "Database OutAge", "inc-1-database-outage"},
		// Test 80-character limit with different ID lengths
		{"Large ID with long title", 10000, "this-is-a-very-long-title-that-should-be-truncated-to-fit-within-eighty-chars", "inc-10000-this-is-a-very-long-title-that-should-be-truncated-to-fit-within-eight"},
		{"Very large ID with title", 999999, "database-outage-critical-priority-affecting-all-users-worldwide", "inc-999999-database-outage-critical-priority-affecting-all-users-worldwide"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			incident, err := model.NewIncident(
				"inc", // prefix
				types.IncidentID(tc.id),
				tc.title,
				"",
				"unknown",
				types.ChannelID("C12345"),
				types.ChannelName("general"),
				types.TeamID("T12345"),
				types.SlackUserID("U67890"),
				false, // initialTriage
			)
			gt.NoError(t, err).Required()
			gt.Equal(t, types.ChannelName(tc.expectedName), incident.ChannelName)
			gt.Equal(t, tc.title, incident.Title)
			// Ensure channel name never exceeds 80 characters
			gt.True(t, len(string(incident.ChannelName)) <= 80)
		})
	}
}

func TestIncidentWithCustomPrefix(t *testing.T) {
	testCases := []struct {
		name         string
		prefix       string
		id           int
		title        string
		expectedName string
	}{
		{"Custom prefix security", "security", 1, "data breach", "security-1-data-breach"},
		{"Custom prefix incident", "incident", 2, "", "incident-2"},
		{"Custom prefix alert", "alert", 100, "system down", "alert-100-system-down"},
		{"Empty prefix fallback", "", 1, "test", "inc-1-test"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			incident, err := model.NewIncident(
				tc.prefix,
				types.IncidentID(tc.id),
				tc.title,
				"",
				"unknown",
				types.ChannelID("C12345"),
				types.ChannelName("general"),
				types.TeamID("T12345"),
				types.SlackUserID("U67890"),
				false, // initialTriage
			)
			gt.NoError(t, err).Required()
			gt.Equal(t, types.ChannelName(tc.expectedName), incident.ChannelName)
		})
	}
}

func TestSanitizeForSlackChannelName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic cases
		{"Empty string", "", ""},
		{"Lowercase letters", "hello", "hello"},
		{"Uppercase letters", "HELLO", "hello"},
		{"Numbers", "test123", "test123"},
		{"Hyphens and underscores", "test-name_version", "test-name_version"},

		// Spaces
		{"Single space", "hello world", "hello-world"},
		{"Multiple spaces", "hello   world", "hello-world"},
		{"Leading and trailing spaces", "  hello world  ", "hello-world"},

		// Special characters
		{"Exclamation mark", "hello!", "hello"},
		{"Question mark", "hello?", "hello"},
		{"Period", "hello.world", "hello-world"},
		{"Multiple periods", "version.2.0.1", "version-2-0-1"},
		{"At symbol", "hello@world", "hello-world"},
		{"Hash symbol", "hello#world", "hello-world"},
		{"Dollar sign", "hello$world", "hello-world"},
		{"Percent", "hello%world", "hello-world"},
		{"Ampersand", "hello&world", "hello-world"},
		{"Parentheses", "hello(world)", "hello-world"},
		{"Brackets", "hello[world]", "hello-world"},
		{"Braces", "hello{world}", "hello-world"},

		// Multibyte characters (should be preserved)
		{"Hiragana", "hiragana-text", "hiragana-text"},
		{"Katakana", "katakana-text", "katakana-text"},
		{"Kanji", "kanji-text", "kanji-text"},
		{"Mixed multibyte", "database-error-mb", "database-error-mb"},
		{"Multibyte with English", "hello-world-mb", "hello-world-mb"},
		{"English with multibyte", "world-hello-mb", "world-hello-mb"},
		{"Multibyte with spaces", "data base mb", "data-base-mb"},

		// Other multibyte characters (should be preserved)
		{"Chinese", "hello-cn", "hello-cn"},
		{"Korean", "hello-kr", "hello-kr"},
		{"Arabic", "hello-ar", "hello-ar"},
		{"Russian", "privet", "privet"},
		{"German umlauts", "schoen", "schoen"},
		{"French accents", "cafe", "cafe"},
		{"Spanish accents", "nino", "nino"},

		// Mixed cases
		{"Mixed English and accented", "hello cafe", "hello-cafe"},
		{"Mixed case with special chars", "Hello World!", "hello-world"},
		{"Numbers with special chars", "version-2.0!", "version-2-0"},

		// Consecutive hyphens
		{"Multiple hyphens", "hello---world", "hello-world"},
		{"Mixed separators", "hello___---___world", "hello___-___world"},

		// Edge cases with hyphens
		{"Leading hyphen", "-hello", "hello"},
		{"Trailing hyphen", "hello-", "hello"},
		{"Only hyphens", "---", ""},
		{"Only underscores", "___", "___"},

		// No length truncation in sanitize function anymore - handled in formatIncidentChannelName
		{"Long title", "this-is-a-very-long-title-that-should-fit-within-the-limit-exactly", "this-is-a-very-long-title-that-should-fit-within-the-limit-exactly"},
		{"Very long title", "this-is-a-very-long-title-that-exceeds-the-maximum-allowed-length-and-should-be-truncated", "this-is-a-very-long-title-that-exceeds-the-maximum-allowed-length-and-should-be-truncated"},

		// Complex real-world examples
		{"Database outage", "Database Outage", "database-outage"},
		{"API server down", "API Server Down!", "api-server-down"},
		{"Network connectivity issue", "Network Connectivity Issue", "network-connectivity-issue"},
		{"Service timeout error", "Service timeout error (urgent)", "service-timeout-error-urgent"},

		// Emoji and symbols (should be replaced with hyphens)
		{"Emoji", "database 💥 error", "database-error"},
		{"Multiple emoji", "🚨 urgent 🔥 issue 🚨", "urgent-issue"},
		{"Currency symbols", "cost $100 €50 ¥1000", "cost-100-50-1000"},

		// Multibyte symbols (should be replaced with hyphens)
		{"Multibyte symbols exclamation", "data!error?", "data-error"},
		{"Multibyte punctuation colon", "server:stopped", "server-stopped"},
		{"Multibyte brackets", "database<<error>>", "database-error"},
		{"Multibyte quotes", "server[error]occurred", "server-error-occurred"},
		{"Full-width punctuation", "API(error)", "api-error"},
		{"Mixed multibyte symbols", "data·base#error", "data-base-error"},
		{"Multibyte brackets important", "[important]system-failure", "important-system-failure"},
		{"Wave dash", "server~stopped", "server-stopped"},

		// Edge cases with multibyte characters
		{"Only multibyte", "database-mb", "database-mb"},
		{"Multibyte with spaces", "data base error mb", "data-base-error-mb"},
		{"Mixed multibyte and ASCII", "db database-mb error", "db-database-mb-error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := model.SanitizeForSlackChannelName(tc.input)
			gt.Equal(t, tc.expected, result)

			// Verify Slack channel name constraints (excluding length which is now handled elsewhere)
			if result != "" {
				// Must not start or end with hyphens
				gt.True(t, result[0] != '-')
				gt.True(t, result[len(result)-1] != '-')

				// Must not contain consecutive hyphens
				gt.False(t, strings.Contains(result, "--"))

				// Must not contain spaces or periods
				gt.False(t, strings.ContainsRune(result, ' '))
				gt.False(t, strings.ContainsRune(result, '.'))
			}
		})
	}
}
