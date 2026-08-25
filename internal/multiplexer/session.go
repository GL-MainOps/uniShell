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
	NativeName  string    `json:"native_name,omitempty"`
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
		return fmt.Errorf(
			"create multiplexer metadata directory: %w",
			err,
		)
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf(
			"encode multiplexer metadata: %w",
			err,
		)
	}

	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0600,
	)
	if err != nil {
		return fmt.Errorf(
			"open multiplexer metadata: %w",
			err,
		)
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf(
			"write multiplexer metadata: %w",
			err,
		)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf(
			"close multiplexer metadata: %w",
			err,
		)
	}

	return nil
}

func ReadMetadata(runtimePath string) (Metadata, error) {
	data, err := os.ReadFile(MetadataPath(runtimePath))
	if err != nil {
		return Metadata{}, err
	}

	var metadata Metadata

	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, fmt.Errorf(
			"decode multiplexer metadata: %w",
			err,
		)
	}

	if metadata.ID == "" ||
		metadata.Name == "" ||
		metadata.Multiplexer == "" {
		return Metadata{}, fmt.Errorf(
			"invalid multiplexer metadata",
		)
	}

	return metadata, nil
}

func RemoveMetadata(runtimePath string) error {
	path := MetadataPath(runtimePath)

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf(
			"remove multiplexer metadata: %w",
			err,
		)
	}

	return nil
}
