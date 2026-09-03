package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const MetadataFileName = ".session.json"

type Mode string

const (
	ModeNormal      Mode = "normal"
	ModeMultiplexer Mode = "multiplexer"
)

type Metadata struct {
	ID                string    `json:"id"`
	PID               int       `json:"pid"`
	ProcessStartTicks uint64    `json:"process_start_ticks"`
	CreatedAt         time.Time `json:"created_at"`
	Version           string    `json:"version"`
	Mode              Mode      `json:"mode"`
	ShellName         string    `json:"shell_name,omitempty"`
	ShellPath         string    `json:"shell_path,omitempty"`
	Name              string    `json:"name,omitempty"`
	NativeName        string    `json:"native_name,omitempty"`
	Multiplexer       string    `json:"multiplexer,omitempty"`
	Endpoint          string    `json:"endpoint,omitempty"`
}

func MetadataPath(runtimePath string) string {
	return filepath.Join(runtimePath, MetadataFileName)
}

func WriteMetadata(
	runtimePath string,
	metadata Metadata,
) error {
	if runtimePath == "" {
		return fmt.Errorf("session runtime path cannot be empty")
	}

	path := MetadataPath(runtimePath)

	data, err := json.MarshalIndent(
		metadata,
		"",
		"  ",
	)
	if err != nil {
		return fmt.Errorf(
			"encode session metadata: %w",
			err,
		)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf(
			"write session metadata: %w",
			err,
		)
	}

	return nil
}

func ReadMetadata(
	runtimePath string,
) (Metadata, error) {
	if runtimePath == "" {
		return Metadata{}, fmt.Errorf(
			"session runtime path cannot be empty",
		)
	}

	data, err := os.ReadFile(
		MetadataPath(runtimePath),
	)
	if err != nil {
		return Metadata{}, fmt.Errorf(
			"read session metadata: %w",
			err,
		)
	}

	var metadata Metadata

	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, fmt.Errorf(
			"decode session metadata: %w",
			err,
		)
	}

	if err := validateMetadata(metadata); err != nil {
		return Metadata{}, err
	}

	return metadata, nil
}

func RemoveMetadata(
	runtimePath string,
) error {
	if runtimePath == "" {
		return fmt.Errorf("session runtime path cannot be empty")
	}

	if err := os.Remove(
		MetadataPath(runtimePath),
	); err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf(
			"remove session metadata: %w",
			err,
		)
	}

	return nil
}

func validateMetadata(metadata Metadata) error {
	if metadata.ID == "" {
		return fmt.Errorf(
			"invalid session metadata: missing ID",
		)
	}

	if metadata.PID <= 0 {
		return fmt.Errorf(
			"invalid session metadata: invalid PID",
		)
	}

	if metadata.ProcessStartTicks == 0 {
		return fmt.Errorf(
			"invalid session metadata: missing process start ticks",
		)
	}

	if metadata.CreatedAt.IsZero() {
		return fmt.Errorf(
			"invalid session metadata: missing creation time",
		)
	}

	if metadata.Version == "" {
		return fmt.Errorf(
			"invalid session metadata: missing version",
		)
	}

	switch metadata.Mode {
	case ModeNormal, ModeMultiplexer:
		return nil

	default:
		return fmt.Errorf(
			"invalid session metadata: unsupported mode %q",
			metadata.Mode,
		)
	}
}
