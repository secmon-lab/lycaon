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

func TestConfigValidate(t *testing.T) {
	t.Run("valid configuration with categories and severities", func(t *testing.T) {
		config := model.Config{
			Categories: []model.Category{
				{ID: "security_incident", Name: "Security Incident"},
				{ID: "unknown", Name: "Unknown"},
			},
			Severities: []model.Severity{
				{ID: "critical", Name: "Critical", Level: 90},
				{ID: "high", Name: "High", Level: 70},
			},
		}
		gt.NoError(t, config.Validate())
	})

	t.Run("valid configuration with categories only (backward compatibility)", func(t *testing.T) {
		config := model.Config{
			Categories: []model.Category{
				{ID: "security_incident", Name: "Security Incident"},
				{ID: "unknown", Name: "Unknown"},
			},
		}
		gt.NoError(t, config.Validate())
	})

	t.Run("error when categories are invalid", func(t *testing.T) {
		config := model.Config{
			Categories: []model.Category{
				{ID: "", Name: "Invalid"}, // Invalid: empty ID
			},
		}
		gt.Error(t, config.Validate())
	})

	t.Run("error when severities are invalid", func(t *testing.T) {
		config := model.Config{
			Categories: []model.Category{
				{ID: "security_incident", Name: "Security Incident"},
				{ID: "unknown", Name: "Unknown"},
			},
			Severities: []model.Severity{
				{ID: "", Name: "Invalid", Level: 50}, // Invalid: empty ID
			},
		}
		gt.Error(t, config.Validate())
	})
}

func TestConfigGetCategoriesConfig(t *testing.T) {
	config := model.Config{
		Categories: []model.Category{
			{ID: "security_incident", Name: "Security Incident"},
			{ID: "unknown", Name: "Unknown"},
		},
	}

	catConfig := config.GetCategoriesConfig()
	gt.V(t, catConfig).NotNil()
	gt.Equal(t, len(catConfig.Categories), 2)
	gt.Equal(t, catConfig.Categories[0].ID, "security_incident")
}

func TestConfigGetSeveritiesConfig(t *testing.T) {
	config := model.Config{
		Categories: []model.Category{
			{ID: "security_incident", Name: "Security Incident"},
			{ID: "unknown", Name: "Unknown"},
		},
		Severities: []model.Severity{
			{ID: "critical", Name: "Critical", Level: 90},
		},
	}

	sevConfig := config.GetSeveritiesConfig()
	gt.V(t, sevConfig).NotNil()
	gt.Equal(t, len(sevConfig.Severities), 1)
	gt.Equal(t, sevConfig.Severities[0].ID, "critical")
}
