package slack_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	slackSvc "github.com/secmon-lab/lycaon/pkg/service/slack"
)

func TestFormatIncidentChannelName(t *testing.T) {
	testCases := []struct {
		name           string
		incidentNumber int
		expected       string
	}{
		{"Single digit", 1, "inc-001"},
		{"Double digit", 10, "inc-010"},
		{"Triple digit", 100, "inc-100"},
		{"Four digit", 1000, "inc-1000"},
		{"Large number", 9999, "inc-9999"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := slackSvc.FormatIncidentChannelName(tc.incidentNumber)
			gt.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatIncidentChannelNameWithPrefix(t *testing.T) {
	testCases := []struct {
		name           string
		prefix         string
		incidentNumber int
		expected       string
	}{
		{"Default prefix", "inc", 1, "inc-001"},
		{"Custom prefix security", "security", 1, "security-001"},
		{"Custom prefix incident", "incident", 10, "incident-010"},
		{"Custom prefix alert", "alert", 100, "alert-100"},
		{"Empty prefix", "", 1, "-001"},
		{"Long prefix", "emergency", 1, "emergency-001"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := slackSvc.FormatIncidentChannelNameWithPrefix(tc.prefix, tc.incidentNumber)
			gt.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatIncidentChannelNameBackwardCompatibility(t *testing.T) {
	// Test that the original function still works with "inc" prefix
	result := slackSvc.FormatIncidentChannelName(42)
	expected := slackSvc.FormatIncidentChannelNameWithPrefix("inc", 42)
	gt.Equal(t, expected, result)
	gt.Equal(t, "inc-042", result)
}
