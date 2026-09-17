package mapper

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"go-dbreplicator/internal/engine"

	"gopkg.in/yaml.v3"
)

// MappingConfig represents the structure of your mapping YAML files
type MappingConfig struct {
	Mapping map[string]any `yaml:"mapping"`
}

// Load reads and parses a YAML mapping file from disk
func Load(path string) (map[string]interface{}, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file: %w", err)
	}

	var cfg MappingConfig
	if err := yaml.Unmarshal(fileBytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse mapping YAML: %w", err)
	}

	return cfg.Mapping, nil
}

// MappedDestination is a decorator that intercepts records, applies YAML transformation rules,
// and forwards the transformed records to the underlying database destination writer.
type MappedDestination struct {
	Base    engine.Destination
	Mapping map[string]any
}

// SaveData implements the engine.Destination interface with mapping and JSON transformation logic
func (md *MappedDestination) SaveData(ctx context.Context, records []engine.Record) error {
	var transformedRecords []engine.Record
	log.Printf("[DEBUG] Data to transform: %+v", records)
	for _, rawRecord := range records {
		transformed := make(engine.Record)

		for destField, ruleValue := range md.Mapping {
			switch val := ruleValue.(type) {
			case string:
				// Case A: Standard 1:1 field mapping (e.g., id -> id_kaskel)
				if sourceVal, exists := rawRecord[val]; exists {
					transformed[destField] = sourceVal
				}
			case map[string]any:
				// Case B: Nested map building (e.g., packing fields into the free_text JSON column)
				subMap := make(map[string]any)
				for subKey, srcField := range val {
					if fieldName, ok := srcField.(string); ok {
						if sourceVal, exists := rawRecord[fieldName]; exists {
							subMap[subKey] = sourceVal
						}
					}
				}
				// Marshal the accumulated map into a valid JSON string
				jsonBytes, err := json.Marshal(subMap)
				if err != nil {
					// Fallback or handle error gracefully; store empty JSON or skip
					transformed[destField] = "{}"
				} else {
					transformed[destField] = string(jsonBytes)
				}
			}
		}

		transformedRecords = append(transformedRecords, transformed)
	}

	// Inside SaveData in internal/mapper/mapper.go, right before calling md.Base.SaveData:
	for i, rec := range transformedRecords {
		log.Printf("[DEBUG] Ready to insert record %d: %+v", i, rec)
	}

	// Forward the fully transformed batch down to the base database writer
	return md.Base.SaveData(ctx, transformedRecords)
}
