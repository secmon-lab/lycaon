package usecase_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// TestCallbackIDParsing tests the new callback ID parsing logic
func TestCallbackIDParsing(t *testing.T) {
	testCases := []struct {
		name         string
		callbackID   string
		expectError  bool
		expectedInc  uint64
		expectedTask string
	}{
		{
			name:         "Valid callback ID",
			callbackID:   "task_edit_submit:42:test-task-12345",
			expectError:  false,
			expectedInc:  42,
			expectedTask: "test-task-12345",
		},
		{
			name:         "Valid callback ID with large incident ID",
			callbackID:   "task_edit_submit:999999:another-task",
			expectError:  false,
			expectedInc:  999999,
			expectedTask: "another-task",
		},
		{
			name:         "Valid callback ID with colon in task ID",
			callbackID:   "task_edit_submit:42:task:with:colons:12345",
			expectError:  false,
			expectedInc:  42,
			expectedTask: "task:with:colons:12345",
		},
		{
			name:        "Invalid format - missing colon",
			callbackID:  "task_edit_submit42test-task",
			expectError: true,
		},
		{
			name:        "Invalid format - too many parts",
			callbackID:  "task_edit_submit:42:test:task:12345",
			expectError: false,
			expectedInc: 42,
			expectedTask: "test:task:12345",
		},
		{
			name:        "Invalid format - no parts after prefix",
			callbackID:  "task_edit_submit:",
			expectError: true,
		},
		{
			name:        "Invalid incident ID - not a number",
			callbackID:  "task_edit_submit:abc:test-task",
			expectError: true,
		},
		{
			name:        "Invalid incident ID - negative number",
			callbackID:  "task_edit_submit:-1:test-task",
			expectError: true,
		},
		{
			name:        "Old format - should fail",
			callbackID:  "task_edit_submit_test-task-12345",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate the parsing logic from slack_interaction.go
			callbackData := strings.TrimPrefix(tc.callbackID, "task_edit_submit:")
			parts := strings.SplitN(callbackData, ":", 2)

			if len(parts) != 2 {
				if tc.expectError {
					return // Expected error
				}
				t.Fatal("Expected valid format but got invalid parts count")
			}

			incidentIDStr, taskIDStr := parts[0], parts[1]
			incidentID, err := strconv.ParseUint(incidentIDStr, 10, 64)

			if err != nil {
				if tc.expectError {
					return // Expected error
				}
				t.Fatalf("Expected valid incident ID but got error: %v", err)
			}

			if tc.expectError {
				t.Fatal("Expected error but parsing succeeded")
			}

			// Verify results
			gt.Equal(t, tc.expectedInc, incidentID)
			gt.Equal(t, tc.expectedTask, taskIDStr)
		})
	}
}

// TestCallbackIDCompatibility tests round-trip compatibility
func TestCallbackIDCompatibility(t *testing.T) {
	testCases := []struct {
		incidentID types.IncidentID
		taskID     types.TaskID
	}{
		{1, "task1"},
		{42, "test-task-12345"},
		{999999, "very-long-task-id-with-many-characters"},
		{0, "task0"},
		{123, "task:with:multiple:colons:456"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("incident_%d_task_%s", tc.incidentID, tc.taskID), func(t *testing.T) {
			// Generate callback ID as the service would
			callbackID := fmt.Sprintf("task_edit_submit:%d:%s", tc.incidentID, tc.taskID)

			// Parse it back (simulating the parsing logic)
			const prefix = "task_edit_submit:"
			if !strings.HasPrefix(callbackID, prefix) {
				t.Fatal("callback ID should have correct prefix")
			}

			callbackData := strings.TrimPrefix(callbackID, prefix)
			parts := strings.SplitN(callbackData, ":", 2)
			gt.Equal(t, 2, len(parts))

			incidentIDStr, taskIDStr := parts[0], parts[1]

			// Verify incident ID
			parsedIncidentID, err := strconv.ParseUint(incidentIDStr, 10, 64)
			gt.NoError(t, err)
			gt.Equal(t, uint64(tc.incidentID), parsedIncidentID)

			// Verify task ID
			gt.Equal(t, string(tc.taskID), taskIDStr)
		})
	}
}