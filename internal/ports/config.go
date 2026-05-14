package ports

import "github.com/pauvalls/arx/internal/domain"

// ConfigReader defines the interface for reading configuration files
type ConfigReader interface {
	// Read reads and parses a configuration file
	// Supports YAML, JSON, and other formats based on file extension
	Read(configPath string) (*domain.Config, error)

	// Validate validates the configuration structure
	// This is a convenience method that calls config.Validate()
	Validate(config *domain.Config) error
}
