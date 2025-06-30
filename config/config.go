package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
}

type DatabaseConfig struct {
	DBType           string `yaml:"type"`
	ConnectionString string `yaml:"connection_string,omitempty"`
	File             string `yaml:"file,omitempty"`
}

func LoadConfig(configPath string) (*Config, error) {
	// TODO: fix this
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func (d *DatabaseConfig) GetConnectionString() (string, error) {
	switch d.DBType {
	case "postgres", "mysql":
		if d.ConnectionString == "" {
			return "", fmt.Errorf("Connection string is required for %s connection", d.DBType)
		}

		return d.ConnectionString, nil

	case "sqlite":
		if d.File == "" {
			d.File = "database.db"
		}
		return d.File, nil

	default:
		return "", fmt.Errorf("unsupported Database type: %s", d.DBType)
	}
}
