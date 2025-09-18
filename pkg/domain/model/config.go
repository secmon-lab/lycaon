package model

import (
	"fmt"

	"github.com/m-mizutani/goerr/v2"
)

// CategoriesConfig represents the categories configuration
type CategoriesConfig struct {
	Categories []Category `yaml:"categories"`
}

// Validate validates the categories configuration
func (c *CategoriesConfig) Validate() error {
	if len(c.Categories) == 0 {
		return goerr.New("at least one category is required")
	}

	// Check for duplicate IDs
	idMap := make(map[string]bool)
	for i, cat := range c.Categories {
		// Validate each category
		if err := cat.Validate(); err != nil {
			return goerr.Wrap(err, "invalid category at index",
				goerr.V("index", i),
				goerr.V("id", cat.ID))
		}

		// Check for duplicate IDs
		if idMap[cat.ID] {
			return goerr.New("duplicate category ID",
				goerr.V("id", cat.ID))
		}
		idMap[cat.ID] = true
	}

	// Ensure "unknown" category exists
	if !idMap["unknown"] {
		return goerr.New("'unknown' category is required")
	}

	return nil
}

// FindCategoryByID finds a category by its ID
func (c *CategoriesConfig) FindCategoryByID(id string) *Category {
	for _, cat := range c.Categories {
		if cat.ID == id {
			// Return a copy to prevent modification
			result := cat
			return &result
		}
	}
	return nil
}

// IsValidCategoryID checks if the given category ID exists in the configuration
func (c *CategoriesConfig) IsValidCategoryID(id string) bool {
	return c.FindCategoryByID(id) != nil
}

// FindCategoryByIDWithFallback finds a category or returns unknown category info
func (c *CategoriesConfig) FindCategoryByIDWithFallback(id string) *Category {
	if cat := c.FindCategoryByID(id); cat != nil {
		return cat
	}
	// Return a temporary category object for display
	return &Category{
		ID:          id,
		Name:        fmt.Sprintf("Unknown Category (ID: %s)", id),
		Description: "This category does not exist in current configuration",
	}
}
