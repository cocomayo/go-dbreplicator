package destination

import (
	"context"
	"encoding/json"
	"os"

	"go-dbreplicator/internal/engine"
)

type FileDestination struct {
	FilePath string
}

func NewFileDestination(path string) *FileDestination {
	return &FileDestination{FilePath: path}
}

func (f *FileDestination) SaveData(ctx context.Context, records []engine.Record) error {
	if len(records) == 0 {
		return nil // Nothing to save
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	// Overwrite the destination file to simulate successful insertion
	return os.WriteFile(f.FilePath, data, 0644)
}
