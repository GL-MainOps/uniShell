package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMetadataRoundTrip(t *testing.T) {
	runtimePath := t.TempDir()

	created := time.Date(
		2026,
		9,
		2,
		17,
		30,
		0,
		0,
		time.UTC,
	)

	want := Metadata{
		ID:                "43b9ab54912bf09a03cc414bf7697bf1",
		PID:               12345,
		ProcessStartTicks: 987654,
		CreatedAt:         created,
		Version:           "development",
		Mode:              ModeNormal,
		ShellName:         "bash",
		ShellPath:         "/bin/bash",
	}

	if err := WriteMetadata(runtimePath, want); err != nil {
		t.Fatalf(
			"WriteMetadata() returned error: %v",
			err,
		)
	}

	path := MetadataPath(runtimePath)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"read metadata file: %v",
			err,
		)
	}

	if !strings.Contains(
		string(data),
		"\n  \"id\"",
	) {
		t.Fatalf(
			"metadata is not indented JSON:\n%s",
			data,
		)
	}

	got, err := ReadMetadata(runtimePath)
	if err != nil {
		t.Fatalf(
			"ReadMetadata() returned error: %v",
			err,
		)
	}

	if got.ID != want.ID {
		t.Fatalf(
			"ID = %q, want %q",
			got.ID,
			want.ID,
		)
	}

	if got.PID != want.PID {
		t.Fatalf(
			"PID = %d, want %d",
			got.PID,
			want.PID,
		)
	}

	if got.ProcessStartTicks != want.ProcessStartTicks {
		t.Fatalf(
			"ProcessStartTicks = %d, want %d",
			got.ProcessStartTicks,
			want.ProcessStartTicks,
		)
	}

	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf(
			"CreatedAt = %s, want %s",
			got.CreatedAt,
			want.CreatedAt,
		)
	}

	if got.Version != want.Version {
		t.Fatalf(
			"Version = %q, want %q",
			got.Version,
			want.Version,
		)
	}

	if got.Mode != want.Mode {
		t.Fatalf(
			"Mode = %q, want %q",
			got.Mode,
			want.Mode,
		)
	}

	if got.ShellName != want.ShellName {
		t.Fatalf(
			"ShellName = %q, want %q",
			got.ShellName,
			want.ShellName,
		)
	}

	if got.ShellPath != want.ShellPath {
		t.Fatalf(
			"ShellPath = %q, want %q",
			got.ShellPath,
			want.ShellPath,
		)
	}
}

func TestMetadataPathIsSessionLocalAndHidden(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"development",
		"session-id",
	)

	got := MetadataPath(runtimePath)

	want := filepath.Join(
		runtimePath,
		".session.json",
	)

	if got != want {
		t.Fatalf(
			"MetadataPath() = %q, want %q",
			got,
			want,
		)
	}

	if filepath.Base(got) != ".session.json" {
		t.Fatalf(
			"metadata filename = %q, want %q",
			filepath.Base(got),
			".session.json",
		)
	}
}

func TestWriteMetadataRejectsEmptyRuntimePath(t *testing.T) {
	err := WriteMetadata(
		"",
		Metadata{},
	)

	if err == nil {
		t.Fatal(
			"WriteMetadata() returned nil error",
		)
	}
}

func TestReadMetadataRejectsInvalidMetadata(t *testing.T) {
	runtimePath := t.TempDir()

	if err := os.WriteFile(
		MetadataPath(runtimePath),
		[]byte(`{
			"id": "session",
			"pid": 1234,
			"process_start_ticks": 1,
			"created_at": "2026-09-02T17:30:00Z",
			"version": "development",
			"mode": "invalid"
		}`),
		0600,
	); err != nil {
		t.Fatalf(
			"write invalid metadata: %v",
			err,
		)
	}

	if _, err := ReadMetadata(runtimePath); err == nil {
		t.Fatal(
			"ReadMetadata() returned nil error",
		)
	}
}

func TestRemoveMetadataIsIdempotent(t *testing.T) {
	runtimePath := t.TempDir()

	if err := RemoveMetadata(runtimePath); err != nil {
		t.Fatalf(
			"RemoveMetadata() returned error for missing metadata: %v",
			err,
		)
	}

	metadata := Metadata{
		ID:                "session",
		PID:               1234,
		ProcessStartTicks: 1,
		CreatedAt:         time.Now().UTC(),
		Version:           "development",
		Mode:              ModeNormal,
	}

	if err := WriteMetadata(runtimePath, metadata); err != nil {
		t.Fatalf(
			"WriteMetadata() returned error: %v",
			err,
		)
	}

	if err := RemoveMetadata(runtimePath); err != nil {
		t.Fatalf(
			"RemoveMetadata() returned error: %v",
			err,
		)
	}

	if _, err := os.Stat(
		MetadataPath(runtimePath),
	); !os.IsNotExist(err) {
		t.Fatalf(
			"metadata still exists, stat error = %v",
			err,
		)
	}

	if err := RemoveMetadata(runtimePath); err != nil {
		t.Fatalf(
			"second RemoveMetadata() returned error: %v",
			err,
		)
	}
}
