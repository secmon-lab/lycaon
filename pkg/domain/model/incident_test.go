package model_test

import (
	"strings"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
)

func TestNewIncident(t *testing.T) {
	t.Run("Valid incident creation", func(t *testing.T) {
		incident, err := model.NewIncident(
			1,
			"database outage",
			"C12345",
			"general",
			"U67890",
		)
		gt.NoError(t, err)
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
			0,
			"test",
			"C12345",
			"general",
			"U67890",
		)
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("ID must be positive")
	})

	t.Run("Empty origin channel ID", func(t *testing.T) {
		incident, err := model.NewIncident(
			1,
			"test",
			"",
			"general",
			"U67890",
		)
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("origin channel ID is required")
	})

	t.Run("Empty origin channel name", func(t *testing.T) {
		incident, err := model.NewIncident(
			1,
			"test",
			"C12345",
			"",
			"U67890",
		)
		gt.Error(t, err)
		gt.V(t, incident).Nil()
		gt.S(t, err.Error()).Contains("origin channel name is required")
	})

	t.Run("Empty creator", func(t *testing.T) {
		incident, err := model.NewIncident(
			1,
			"test",
			"C12345",
			"general",
			"",
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
				tc.id,
				"",
				"C12345",
				"general",
				"U67890",
			)
			gt.NoError(t, err)
			gt.Equal(t, tc.expectedName, incident.ChannelName)
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
		{"Title with unicode", 1, "データベースエラー", "inc-1-データベースエラー"},
		{"Mixed case title", 1, "Database OutAge", "inc-1-database-outage"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			incident, err := model.NewIncident(
				tc.id,
				tc.title,
				"C12345",
				"general",
				"U67890",
			)
			gt.NoError(t, err)
			gt.Equal(t, tc.expectedName, incident.ChannelName)
			gt.Equal(t, tc.title, incident.Title)
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

		// Japanese characters (should be preserved)
		{"Hiragana", "こんにちは", "こんにちは"},
		{"Katakana", "コンニチハ", "コンニチハ"},
		{"Kanji", "漢字", "漢字"},
		{"Mixed Japanese", "データベースエラー", "データベースエラー"},
		{"Japanese with English", "hello世界", "hello世界"},
		{"English with Japanese", "世界hello", "世界hello"},
		{"Japanese with spaces", "データ ベース", "データ-ベース"},

		// Other multibyte characters (should be preserved)
		{"Chinese", "你好", "你好"},
		{"Korean", "안녕하세요", "안녕하세요"},
		{"Arabic", "مرحبا", "مرحبا"},
		{"Russian", "привет", "привет"},
		{"German umlauts", "schön", "schön"},
		{"French accents", "café", "café"},
		{"Spanish accents", "niño", "niño"},

		// Mixed cases
		{"Mixed English and accented", "hello café", "hello-café"},
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

		// Length testing (72 bytes max for title part)
		{"Max length", "this-is-a-very-long-title-that-should-fit-within-the-limit-exactly", "this-is-a-very-long-title-that-should-fit-within-the-limit-exactly"},
		{"Over max length", "this-is-a-very-long-title-that-exceeds-the-maximum-allowed-length-and-should-be-truncated", "this-is-a-very-long-title-that-exceeds-the-maximum-allowed-length-and-sh"},

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
		{"Japanese symbols", "データ！エラー？", "データ-エラー"},
		{"Japanese punctuation", "サーバー：停止中", "サーバー-停止中"},
		{"Chinese symbols", "数据库《错误》", "数据库-错误"},
		{"Korean symbols", "서버「오류」발생", "서버-오류-발생"},
		{"Full-width punctuation", "ＡＰＩ（エラー）", "ＡＰＩ-エラー"},
		{"Mixed multibyte symbols", "データ・ベース＃エラー", "データ-ベース-エラー"},
		{"Japanese brackets", "【重要】システム障害", "重要-システム障害"},
		{"Wave dash", "サーバー〜停止", "サーバー-停止"},

		// Edge cases with multibyte characters
		{"Only multibyte", "データベース", "データベース"},
		{"Multibyte with spaces", "データ ベース エラー", "データ-ベース-エラー"},
		{"Mixed multibyte and ASCII", "db データベース error", "db-データベース-error"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := model.SanitizeForSlackChannelName(tc.input)
			gt.Equal(t, tc.expected, result)

			// Verify Slack channel name constraints
			if result != "" {
				// Must be 72 bytes or less (for title part)
				gt.True(t, len(result) <= 72)

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
