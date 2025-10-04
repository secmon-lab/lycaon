package model

import (
	"github.com/m-mizutani/goerr/v2"
)

// Severity represents a severity level
type Severity struct {
	ID          string `yaml:"id"`          // Unique identifier
	Name        string `yaml:"name"`        // Display name
	Description string `yaml:"description"` // Description for selection help (optional)
	Level       int    `yaml:"level"`       // Importance level (0-99, -1 for unknown)
}

// Validate validates the severity
func (s *Severity) Validate() error {
	if s.ID == "" {
		return goerr.New("severity ID is required")
	}
	if s.Name == "" {
		return goerr.New("severity name is required")
	}
	if s.Level < 0 || s.Level > 99 {
		return goerr.New("severity level must be between 0 and 99",
			goerr.V("level", s.Level))
	}
	return nil
}

// IsIgnorable returns true if the severity level is 0 (ignorable)
func (s *Severity) IsIgnorable() bool {
	return s.Level == 0
}

// IsUnknown returns true if the severity is unknown (level -1)
func (s *Severity) IsUnknown() bool {
	return s.Level == -1
}

// SeveritiesConfig represents the severities configuration
type SeveritiesConfig struct {
	Severities []Severity `yaml:"severities"`
}

// Validate validates the severities configuration
func (c *SeveritiesConfig) Validate() error {
	if len(c.Severities) == 0 {
		return goerr.New("at least one severity is required")
	}

	idMap := make(map[string]bool)
	for i, sev := range c.Severities {
		if err := sev.Validate(); err != nil {
			return goerr.Wrap(err, "invalid severity at index",
				goerr.V("index", i),
				goerr.V("id", sev.ID))
		}

		if idMap[sev.ID] {
			return goerr.New("duplicate severity ID",
				goerr.V("id", sev.ID))
		}
		idMap[sev.ID] = true
	}

	return nil
}

// FindSeverityByID finds a severity by its ID
func (c *SeveritiesConfig) FindSeverityByID(id string) *Severity {
	for _, sev := range c.Severities {
		if sev.ID == id {
			result := sev
			return &result
		}
	}
	return nil
}

// IsValidSeverityID checks if the given severity ID exists
func (c *SeveritiesConfig) IsValidSeverityID(id string) bool {
	return c.FindSeverityByID(id) != nil
}

// FindSeverityByIDWithFallback finds a severity or returns unknown severity
// Empty string or non-existent ID returns "unknown" severity
func (c *SeveritiesConfig) FindSeverityByIDWithFallback(id string) *Severity {
	// Empty string is treated as unknown
	if id == "" {
		if unknown := c.FindSeverityByID("unknown"); unknown != nil {
			return unknown
		}
		// Fallback if unknown severity is not configured
		return &Severity{
			ID:          "unknown",
			Name:        "Unknown",
			Description: "Unknown severity",
			Level:       -1,
		}
	}

	// Return if ID exists
	if sev := c.FindSeverityByID(id); sev != nil {
		return sev
	}

	// Non-existent ID is also treated as unknown
	if unknown := c.FindSeverityByID("unknown"); unknown != nil {
		return unknown
	}

	// Fallback if unknown severity is not configured
	return &Severity{
		ID:          "unknown",
		Name:        "Unknown",
		Description: "Unknown severity",
		Level:       -1,
	}
}
