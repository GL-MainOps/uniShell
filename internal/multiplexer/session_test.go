package multiplexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	runtimepkg "gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

func TestMetadataRoundTrip(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

	created := time.Date(
		2026,
		8,
		25,
		1,
		0,
		0,
		0,
		time.UTC,
	)

	want := Metadata{
		ID:                "abc123",
		PID:               os.Getpid(),
		ProcessStartTicks: runtimepkg.CurrentProcessStartTicks(),
		Name:              "default",
		Multiplexer:       "tmux",
		Endpoint: filepath.Join(
			runtimePath,
			"multiplexer",
			"tmux.sock",
		),
		CreatedAt: created,
		Version:   "development",
		Mode:      sessionmeta.ModeMultiplexer,
	}

	if err := WriteMetadata(runtimePath, want); err != nil {
		t.Fatalf("WriteMetadata() returned error: %v", err)
	}

	data, err := os.ReadFile(MetadataPath(runtimePath))
	if err != nil {
		t.Fatalf("ReadMetadata file: %v", err)
	}

	if !strings.Contains(string(data), "\n  \"id\"") {
		t.Fatalf(
			"metadata JSON is not indented:\n%s",
			data,
		)
	}

	got, err := ReadMetadata(runtimePath)
	if err != nil {
		t.Fatalf("ReadMetadata() returned error: %v", err)
	}

	if got.ID != want.ID {
		t.Fatalf("ID = %q, want %q", got.ID, want.ID)
	}

	if got.Name != want.Name {
		t.Fatalf("Name = %q, want %q", got.Name, want.Name)
	}

	if got.Multiplexer != want.Multiplexer {
		t.Fatalf(
			"Multiplexer = %q, want %q",
			got.Multiplexer,
			want.Multiplexer,
		)
	}

	if got.Endpoint != want.Endpoint {
		t.Fatalf(
			"Endpoint = %q, want %q",
			got.Endpoint,
			want.Endpoint,
		)
	}

	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Fatalf(
			"CreatedAt = %v, want %v",
			got.CreatedAt,
			want.CreatedAt,
		)
	}
}

func TestRemoveMetadataIsIdempotent(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	if err := RemoveMetadata(runtimePath); err != nil {
		t.Fatalf(
			"RemoveMetadata() returned error for missing metadata: %v",
			err,
		)
	}
}
