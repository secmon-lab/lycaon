package model

import (
	"fmt"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
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

// Config represents the unified configuration with categories and severities
type Config struct {
	Categories []Category `yaml:"categories"`
	Severities []Severity `yaml:"severities,omitempty"`
	Assets     []Asset    `yaml:"assets,omitempty"`
}

// Validate validates the entire configuration
func (c *Config) Validate() error {
	// Validate categories
	catConfig := &CategoriesConfig{Categories: c.Categories}
	if err := catConfig.Validate(); err != nil {
		return goerr.Wrap(err, "invalid categories")
	}

	// Validate severities if present (optional for backward compatibility)
	if len(c.Severities) > 0 {
		sevConfig := &SeveritiesConfig{Severities: c.Severities}
		if err := sevConfig.Validate(); err != nil {
			return goerr.Wrap(err, "invalid severities")
		}
	}

	// Validate assets if present (optional)
	if len(c.Assets) > 0 {
		idMap := make(map[types.AssetID]bool)
		for i, asset := range c.Assets {
			if err := asset.Validate(); err != nil {
				return goerr.Wrap(err, "invalid asset at index",
					goerr.V("index", i),
					goerr.V("id", asset.ID))
			}

			if idMap[asset.ID] {
				return goerr.New("duplicate asset ID",
					goerr.V("id", asset.ID))
			}
			idMap[asset.ID] = true
		}
	}

	return nil
}

// GetCategoriesConfig returns CategoriesConfig
func (c *Config) GetCategoriesConfig() *CategoriesConfig {
	return &CategoriesConfig{Categories: c.Categories}
}

// GetSeveritiesConfig returns SeveritiesConfig
func (c *Config) GetSeveritiesConfig() *SeveritiesConfig {
	return &SeveritiesConfig{Severities: c.Severities}
}

// FindCategoryByID finds a category by its ID
func (c *Config) FindCategoryByID(id string) *Category {
	return c.GetCategoriesConfig().FindCategoryByID(id)
}

// FindCategoryByIDWithFallback finds a category or returns unknown category info
func (c *Config) FindCategoryByIDWithFallback(id string) *Category {
	return c.GetCategoriesConfig().FindCategoryByIDWithFallback(id)
}

// IsValidCategoryID checks if the given category ID exists
func (c *Config) IsValidCategoryID(id string) bool {
	return c.GetCategoriesConfig().IsValidCategoryID(id)
}

// FindSeverityByID finds a severity by its ID
func (c *Config) FindSeverityByID(id string) *Severity {
	return c.GetSeveritiesConfig().FindSeverityByID(id)
}

// FindSeverityByIDWithFallback finds a severity or returns unknown severity
func (c *Config) FindSeverityByIDWithFallback(id string) *Severity {
	return c.GetSeveritiesConfig().FindSeverityByIDWithFallback(id)
}

// FindAssetByID finds an asset by its ID
func (c *Config) FindAssetByID(id types.AssetID) *Asset {
	for _, asset := range c.Assets {
		if asset.ID == id {
			result := asset
			return &result
		}
	}
	return nil
}

// FindAssetsByIDs finds multiple assets by their IDs
func (c *Config) FindAssetsByIDs(ids []types.AssetID) []Asset {
	result := make([]Asset, 0, len(ids))
	for _, id := range ids {
		if asset := c.FindAssetByID(id); asset != nil {
			result = append(result, *asset)
		}
	}
	return result
}

// IsValidAssetID checks if the given asset ID exists
func (c *Config) IsValidAssetID(id types.AssetID) bool {
	return c.FindAssetByID(id) != nil
}

// ValidateAssetIDs checks if all provided IDs are valid
func (c *Config) ValidateAssetIDs(ids []types.AssetID) error {
	for _, id := range ids {
		if !c.IsValidAssetID(id) {
			return goerr.New("invalid asset ID",
				goerr.V("id", id))
		}
	}
	return nil
}

// FindAssetByIDWithFallback finds an asset or returns a fallback
func (c *Config) FindAssetByIDWithFallback(id types.AssetID) *Asset {
	if asset := c.FindAssetByID(id); asset != nil {
		return asset
	}
	return &Asset{
		ID:          id,
		Name:        "Unknown Asset",
		Description: "This asset does not exist in current configuration",
	}
}
