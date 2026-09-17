package factory

import (
	"errors"

	"go-dbreplicator/internal/config"
	"go-dbreplicator/internal/destination"
	"go-dbreplicator/internal/engine"
	"go-dbreplicator/internal/source"
)

// NewSource dynamically creates the extraction dependency.
func NewSource(cfg config.ConnectionConfig) (engine.Source, error) {
	switch cfg.Type {
	case "LOCAL":
		if cfg.Local == nil || cfg.Local.Path == "" {
			return nil, errors.New("LOCAL source requires a 'local' config block with a 'path'")
		}
		return source.NewFileSource(cfg.Local.Path), nil

	case "MOCK":
		if cfg.Mock == nil || cfg.Mock.ApiLink == "" {
			return nil, errors.New("MOCK source requires a 'mock' config block with an 'api_link'")
		}
		// Future implementation:
		// return source.NewApiSource(cfg.Mock.ApiLink), nil
		return nil, errors.New("MOCK API source not implemented yet")

	case "DB":
		if cfg.DB == nil || cfg.DB.Host == "" || cfg.DB.User == "" {
			return nil, errors.New("DB source requires a complete 'db' config block")
		}
		// Future implementation:
		// connString := fmt.Sprintf("oracle://%s:%s@%s:%d/%s", cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Database)
		// return source.NewOracleSource(connString), nil
		return nil, errors.New("DB source not implemented yet")

	default:
		return nil, errors.New("unsupported source type: " + cfg.Type)
	}
}

// NewDestination dynamically creates the loading dependency.
func NewDestination(cfg config.ConnectionConfig) (engine.Destination, error) {
	switch cfg.Type {
	case "LOCAL":
		if cfg.Local == nil || cfg.Local.Path == "" {
			return nil, errors.New("LOCAL destination requires a 'local' config block with a 'path'")
		}
		return destination.NewFileDestination(cfg.Local.Path), nil

	case "DB":
		if cfg.DB == nil || cfg.DB.Host == "" || cfg.DB.User == "" {
			return nil, errors.New("DB destination requires a complete 'db' config block")
		}
		// Future implementation: SQL Server connection logic here
		return nil, errors.New("DB destination not implemented yet")

	default:
		return nil, errors.New("unsupported destination type: " + cfg.Type)
	}
}
