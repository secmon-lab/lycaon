package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
)

func TestSeverityValidate(t *testing.T) {
	t.Run("valid severity with level 0", func(t *testing.T) {
		sev := model.Severity{
			ID:          "info",
			Name:        "Info",
			Description: "Information only",
			Level:       0,
		}
		gt.NoError(t, sev.Validate())
	})

	t.Run("valid severity with level 50", func(t *testing.T) {
		sev := model.Severity{
			ID:    "medium",
			Name:  "Medium",
			Level: 50,
		}
		gt.NoError(t, sev.Validate())
	})

	t.Run("valid severity with level 99", func(t *testing.T) {
		sev := model.Severity{
			ID:    "critical",
			Name:  "Critical",
			Level: 99,
		}
		gt.NoError(t, sev.Validate())
	})

	t.Run("error when ID is empty", func(t *testing.T) {
		sev := model.Severity{
			ID:    "",
			Name:  "Test",
			Level: 50,
		}
		gt.Error(t, sev.Validate())
	})

	t.Run("error when Name is empty", func(t *testing.T) {
		sev := model.Severity{
			ID:    "test",
			Name:  "",
			Level: 50,
		}
		gt.Error(t, sev.Validate())
	})

	t.Run("error when Level is -1", func(t *testing.T) {
		sev := model.Severity{
			ID:    "test",
			Name:  "Test",
			Level: -1,
		}
		gt.Error(t, sev.Validate())
	})

	t.Run("error when Level is 100", func(t *testing.T) {
		sev := model.Severity{
			ID:    "test",
			Name:  "Test",
			Level: 100,
		}
		gt.Error(t, sev.Validate())
	})
}

func TestSeverityIsIgnorable(t *testing.T) {
	t.Run("returns true for level 0", func(t *testing.T) {
		sev := model.Severity{Level: 0}
		gt.True(t, sev.IsIgnorable())
	})

	t.Run("returns false for level 1", func(t *testing.T) {
		sev := model.Severity{Level: 1}
		gt.False(t, sev.IsIgnorable())
	})
}

func TestSeverityIsUnknown(t *testing.T) {
	t.Run("returns true for level -1", func(t *testing.T) {
		sev := model.Severity{Level: -1}
		gt.True(t, sev.IsUnknown())
	})

	t.Run("returns false for level 0", func(t *testing.T) {
		sev := model.Severity{Level: 0}
		gt.False(t, sev.IsUnknown())
	})
}

func TestSeveritiesConfigValidate(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		config := model.SeveritiesConfig{
			Severities: []model.Severity{
				{ID: "critical", Name: "Critical", Level: 90},
				{ID: "high", Name: "High", Level: 70},
			},
		}
		gt.NoError(t, config.Validate())
	})

	t.Run("error when severities is empty", func(t *testing.T) {
		config := model.SeveritiesConfig{
			Severities: []model.Severity{},
		}
		gt.Error(t, config.Validate())
	})

	t.Run("error when duplicate ID exists", func(t *testing.T) {
		config := model.SeveritiesConfig{
			Severities: []model.Severity{
				{ID: "critical", Name: "Critical", Level: 90},
				{ID: "critical", Name: "Critical2", Level: 80},
			},
		}
		gt.Error(t, config.Validate())
	})

	t.Run("error when invalid severity exists", func(t *testing.T) {
		config := model.SeveritiesConfig{
			Severities: []model.Severity{
				{ID: "critical", Name: "Critical", Level: 90},
				{ID: "", Name: "Invalid", Level: 50}, // Invalid: empty ID
			},
		}
		gt.Error(t, config.Validate())
	})
}

func TestFindSeverityByID(t *testing.T) {
	config := model.SeveritiesConfig{
		Severities: []model.Severity{
			{ID: "critical", Name: "Critical", Level: 90},
			{ID: "high", Name: "High", Level: 70},
		},
	}

	t.Run("returns severity when ID exists", func(t *testing.T) {
		sev := config.FindSeverityByID("critical")
		gt.V(t, sev).NotNil()
		gt.Equal(t, sev.ID, "critical")
		gt.Equal(t, sev.Name, "Critical")
		gt.Equal(t, sev.Level, 90)
	})

	t.Run("returns nil when ID does not exist", func(t *testing.T) {
		sev := config.FindSeverityByID("nonexistent")
		gt.V(t, sev).Nil()
	})
}

func TestFindSeverityByIDWithFallback(t *testing.T) {
	config := model.SeveritiesConfig{
		Severities: []model.Severity{
			{ID: "critical", Name: "Critical", Level: 90},
			{ID: "unknown", Name: "Unknown", Level: 50},
		},
	}

	t.Run("returns severity when ID exists", func(t *testing.T) {
		sev := config.FindSeverityByIDWithFallback("critical")
		gt.V(t, sev).NotNil()
		gt.Equal(t, sev.ID, "critical")
		gt.Equal(t, sev.Name, "Critical")
	})

	t.Run("returns unknown severity for empty string", func(t *testing.T) {
		sev := config.FindSeverityByIDWithFallback("")
		gt.V(t, sev).NotNil()
		gt.Equal(t, sev.ID, "unknown")
		gt.Equal(t, sev.Name, "Unknown")
		gt.Equal(t, sev.Level, 50)
	})

	t.Run("returns unknown severity for non-existent ID", func(t *testing.T) {
		sev := config.FindSeverityByIDWithFallback("nonexistent")
		gt.V(t, sev).NotNil()
		gt.Equal(t, sev.ID, "unknown")
		gt.Equal(t, sev.Name, "Unknown")
	})

	t.Run("returns fallback when unknown is not configured", func(t *testing.T) {
		configWithoutUnknown := model.SeveritiesConfig{
			Severities: []model.Severity{
				{ID: "critical", Name: "Critical", Level: 90},
			},
		}
		sev := configWithoutUnknown.FindSeverityByIDWithFallback("")
		gt.V(t, sev).NotNil()
		gt.Equal(t, sev.ID, "unknown")
		gt.Equal(t, sev.Name, "Unknown")
		gt.Equal(t, sev.Level, -1) // Fallback has level -1
	})
}
