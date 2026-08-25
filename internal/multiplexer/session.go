package multiplexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const metadataFileName = "session.json"

type Metadata struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Multiplexer string    `json:"multiplexer"`
	Endpoint    string    `json:"endpoint,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func MetadataPath(runtimePath string) string {
	return filepath.Join(
		runtimePath,
		"multiplexer",
		metadataFileName,
	)
}

func WriteMetadata(runtimePath string, metadata Metadata) error {
	path := MetadataPath(runtimePath)

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create multiplexer metadata directory: %w", err)
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encode multiplexer metadata: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write multiplexer metadata: %w", err)
	}

	return nil
}

func ReadMetadata(runtimePath string) (Metadata, error) {
	path := MetadataPath(runtimePath)

	data, err := os.ReadFile(path)
	if err != nil {
		return Metadata{}, fmt.Errorf("read multiplexer metadata: %w", err)
	}

	var metadata Metadata

	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, fmt.Errorf(
			"decode multiplexer metadata: %w",
			err,
		)
	}

	return metadata, nil
}

func RemoveMetadata(runtimePath string) error {
	path := MetadataPath(runtimePath)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove multiplexer metadata: %w", err)
	}

	return nil
}
