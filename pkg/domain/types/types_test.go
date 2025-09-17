package types_test

import (
	"testing"

	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

func TestIncidentStatusValidation(t *testing.T) {
	tests := []struct {
		name     string
		status   types.IncidentStatus
		expected bool
	}{
		{"Valid TRIAGE", types.IncidentStatusTriage, true},
		{"Valid HANDLING", types.IncidentStatusHandling, true},
		{"Valid MONITORING", types.IncidentStatusMonitoring, true},
		{"Valid CLOSED", types.IncidentStatusClosed, true},
		{"Invalid empty", types.IncidentStatus(""), false},
		{"Valid lowercase", types.IncidentStatus("triage"), true},
		{"Invalid mixed case", types.IncidentStatus("Triage"), false},
		{"Invalid unknown", types.IncidentStatus("UNKNOWN"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.IsValid()
			if result != tt.expected {
				t.Errorf("IncidentStatus(%q).IsValid() = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestIncidentStatusString(t *testing.T) {
	tests := []struct {
		status   types.IncidentStatus
		expected string
	}{
		{types.IncidentStatusTriage, "triage"},
		{types.IncidentStatusHandling, "handling"},
		{types.IncidentStatusMonitoring, "monitoring"},
		{types.IncidentStatusClosed, "closed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("IncidentStatus(%q).String() = %q, want %q", tt.status, result, tt.expected)
			}
		})
	}
}
