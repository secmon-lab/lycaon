package model

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

// Asset represents an infrastructure component or service
type Asset struct {
	ID          types.AssetID `yaml:"id"`          // Unique identifier (e.g., "web_frontend")
	Name        string        `yaml:"name"`        // Display name
	Description string        `yaml:"description"` // Description for selection help (optional)
}

// Validate validates the asset
func (a *Asset) Validate() error {
	if a.ID == "" {
		return goerr.New("asset ID is required")
	}
	if a.Name == "" {
		return goerr.New("asset name is required")
	}
	// Description is optional
	return nil
}
