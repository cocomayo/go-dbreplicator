package source

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"go-dbreplicator/internal/engine"
)

type FileSource struct {
	FilePath string
}

func NewFileSource(path string) *FileSource {
	return &FileSource{FilePath: path}
}

func (f *FileSource) FetchData(ctx context.Context, since time.Time) ([]engine.Record, error) {
	file, err := os.ReadFile(f.FilePath)
	if err != nil {
		return nil, err
	}

	var allRecords []engine.Record
	if err := json.Unmarshal(file, &allRecords); err != nil {
		return nil, err
	}

	var newRecords []engine.Record
	for _, record := range allRecords {
		// Extract the timestamp and check if it's newer than our last run
		if updatedAtStr, ok := record["updated_at"].(string); ok {
			updatedAt, _ := time.Parse(time.RFC3339, updatedAtStr)
			if updatedAt.After(since) {
				newRecords = append(newRecords, record)
			}
		}
	}
	return newRecords, nil
}
