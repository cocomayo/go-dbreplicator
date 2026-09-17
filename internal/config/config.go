package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// LocalConfig holds settings for mock files.
type LocalConfig struct {
	Path string `yaml:"path"`
}

// MockConfig holds settings for mock APIs.
type MockConfig struct {
	ApiLink string `yaml:"api_link"`
}

// DBConfig holds settings for real databases.
type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// ConnectionConfig wraps the chosen type and its specific settings.
type ConnectionConfig struct {
	Type  string       `yaml:"type"`
	Local *LocalConfig `yaml:"local,omitempty"`
	Mock  *MockConfig  `yaml:"mock,omitempty"`
	DB    *DBConfig    `yaml:"db,omitempty"`
}

// AppConfig is the main configuration file.
type AppConfig struct {
	PollingInterval int              `yaml:"polling_interval_seconds"`
	Source          ConnectionConfig `yaml:"source"`
	Destination     ConnectionConfig `yaml:"destination"`
}

// Load reads and parses the YAML configuration.
func Load(filename string) (*AppConfig, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
