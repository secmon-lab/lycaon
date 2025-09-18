package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
)

func TestCategoriesConfig_IsValidCategoryID(t *testing.T) {
	// Use default categories for testing
	config := model.GetDefaultCategories()

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