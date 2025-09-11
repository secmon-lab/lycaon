package model

import "github.com/m-mizutani/goerr/v2"

// Category represents an incident category
type Category struct {
	ID           string   `yaml:"id"`                        // Unique identifier (e.g., "security_incident")
	Name         string   `yaml:"name"`                      // Display name
	Description  string   `yaml:"description"`               // Description for selection help
	InviteUsers  []string `yaml:"invite_users,omitempty"`   // User IDs or @usernames to invite
	InviteGroups []string `yaml:"invite_groups,omitempty"` // Group IDs or @groupnames to invite
}

// Validate validates the category
func (c *Category) Validate() error {
	if c.ID == "" {
		return goerr.New("category ID is required")
	}
	if c.Name == "" {
		return goerr.New("category name is required")
	}
	// Description is optional
	return nil
}
