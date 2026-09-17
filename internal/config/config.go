package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// LocalConfig, MockConfig, DBConfig, and ConnectionConfig remain the same as before...
type LocalConfig struct {
	Path string `yaml:"path"`
}

type MockConfig struct {
	ApiLink string `yaml:"api_link"`
}

type DBConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type SourceConfig struct {
	Type string   `yaml:"type"`
	File string   `yaml:"file,omitempty"` // For LOCAL mock files
	DB   DBConfig `yaml:"db,omitempty"`   // For real DBs
	// The Query field has been removed from here
}

type ConnectionConfig struct {
	Type  string       `yaml:"type"`
	Local *LocalConfig `yaml:"local,omitempty"`
	Mock  *MockConfig  `yaml:"mock,omitempty"`
	DB    DBConfig     `yaml:"db,omitempty"`
}

// NEW: BrokerConfig holds the message queue connection details
type BrokerConfig struct {
	Type         string `yaml:"type"`
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	TopicOrQueue string `yaml:"topic_or_queue"`
}

// AppConfig is the main configuration file.
type AppConfig struct {
	PollingInterval int              `yaml:"polling_interval_seconds"`
	Broker          *BrokerConfig    `yaml:"broker,omitempty"` // Added broker pointer
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
