package config

import (
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"gopkg.in/yaml.v3"
)

// LoadCategoriesFromFile loads categories from YAML file
func LoadCategoriesFromFile(path string) (*model.CategoriesConfig, error) {
	if path == "" {
		return nil, goerr.New("configuration file path is required")
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, goerr.Wrap(err, "configuration file not found",
				goerr.V("path", path))
		}
		return nil, goerr.Wrap(err, "failed to read configuration file",
			goerr.V("path", path))
	}

	// Parse YAML
	var config model.CategoriesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, goerr.Wrap(err, "failed to parse YAML configuration",
			goerr.V("path", path))
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid configuration",
			goerr.V("path", path))
	}

	return &config, nil
}
