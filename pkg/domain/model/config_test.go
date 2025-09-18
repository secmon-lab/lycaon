package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
)

// getTestCategories returns categories for testing purposes
func getTestCategories() *model.CategoriesConfig {
	return &model.CategoriesConfig{
		Categories: []model.Category{
			{
				ID:           "security_incident",
				Name:         "Security Incident",
				Description:  "Security-related incidents including unauthorized access and malware infections",
				InviteUsers:  []string{"@security-lead"},
				InviteGroups: []string{"@security-team"},
			},
			{
				ID:           "system_failure",
				Name:         "System Failure",
				Description:  "System or service failures and outages",
				InviteUsers:  []string{"@sre-lead"},
				InviteGroups: []string{"@sre-oncall"},
			},
			{
				ID:          "performance_issue",
				Name:        "Performance Issue",
				Description: "System performance degradation or response time issues",
			},
			{
				ID:          "unknown",
				Name:        "Unknown",
				Description: "Incidents that cannot be categorized",
			},
		},
	}
}

func TestCategoriesConfig_IsValidCategoryID(t *testing.T) {
	// Use test categories
	config := getTestCategories()

	testCases := []struct {
		name       string
		categoryID string
		expected   bool
	}{
		{
			name:       "Valid category - security_incident",
			categoryID: "security_incident",
			expected:   true,
		},
		{
			name:       "Valid category - system_failure",
			categoryID: "system_failure",
			expected:   true,
		},
		{
			name:       "Valid category - performance_issue",
			categoryID: "performance_issue",
			expected:   true,
		},
		{
			name:       "Valid category - unknown",
			categoryID: "unknown",
			expected:   true,
		},
		{
			name:       "Invalid category - nonexistent",
			categoryID: "nonexistent_category",
			expected:   false,
		},
		{
			name:       "Invalid category - empty string",
			categoryID: "",
			expected:   false,
		},
		{
			name:       "Invalid category - wrong case",
			categoryID: "Security_Incident",
			expected:   false,
		},
		{
			name:       "Invalid category - partial match",
			categoryID: "security",
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := config.IsValidCategoryID(tc.categoryID)
			gt.Equal(t, tc.expected, result)
		})
	}
}

func TestCategoriesConfig_IsValidCategoryID_CustomConfig(t *testing.T) {
	// Test with custom categories config
	config := &model.CategoriesConfig{
		Categories: []model.Category{
			{
				ID:          "custom_category",
				Name:        "Custom Category",
				Description: "A custom test category",
			},
			{
				ID:          "another_custom",
				Name:        "Another Custom",
				Description: "Another test category",
			},
			{
				ID:          "unknown",
				Name:        "Unknown",
				Description: "Fallback category",
			},
		},
	}

	testCases := []struct {
		name       string
		categoryID string
		expected   bool
	}{
		{
			name:       "Valid custom category",
			categoryID: "custom_category",
			expected:   true,
		},
		{
			name:       "Valid another custom category",
			categoryID: "another_custom",
			expected:   true,
		},
		{
			name:       "Valid unknown category",
			categoryID: "unknown",
			expected:   true,
		},
		{
			name:       "Invalid - default category not in custom config",
			categoryID: "security_incident",
			expected:   false,
		},
		{
			name:       "Invalid - nonexistent category",
			categoryID: "does_not_exist",
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := config.IsValidCategoryID(tc.categoryID)
			gt.Equal(t, tc.expected, result)
		})
	}
}
