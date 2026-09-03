package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

func writeCleanTestMetadata(
	t *testing.T,
	runtimeDir string,
	mode sessionmeta.Mode,
	name string,
) {
	t.Helper()

	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		t.Fatalf("create runtime directory: %v", err)
	}

	metadata := sessionmeta.Metadata{
		ID:                name + "-id",
		PID:               os.Getpid(),
		ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
		CreatedAt:         time.Now().UTC(),
		Version:           "development",
		Mode:              mode,
		Name:              name,
	}

	if mode == sessionmeta.ModeMultiplexer {
		metadata.Multiplexer = "test"
		metadata.NativeName = "native-" + name
		metadata.Endpoint = filepath.Join(
			runtimeDir,
			"multiplexer",
			"test.sock",
		)
	}

	if err := sessionmeta.WriteMetadata(
		runtimeDir,
		metadata,
	); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
}

func TestDiscoverCleanSessionsIncludesNormalSession(t *testing.T) {
	root := t.TempDir()

	runtimeDir := filepath.Join(
		root,
		"normal-session",
	)

	writeCleanTestMetadata(
		t,
		runtimeDir,
		sessionmeta.ModeNormal,
		"development",
	)

	application := &App{
		Paths: runtime.Paths{
			Runtime: root,
		},
	}

	sessions, err := application.DiscoverCleanSessions()
	if err != nil {
		t.Fatalf(
			"DiscoverCleanSessions() returned error: %v",
			err,
		)
	}

	if len(sessions) != 1 {
		t.Fatalf(
			"session count = %d, want %d",
			len(sessions),
			1,
		)
	}

	if sessions[0].Metadata.Mode != sessionmeta.ModeNormal {
		t.Fatalf(
			"session mode = %q, want %q",
			sessions[0].Metadata.Mode,
			sessionmeta.ModeNormal,
		)
	}
}

func TestDiscoverCleanSessionsReturnsEmptyWhenRuntimeMissing(
	t *testing.T,
) {
	application := &App{
		Paths: runtime.Paths{
			Runtime: filepath.Join(
				t.TempDir(),
				"missing",
			),
		},
	}

	sessions, err := application.DiscoverCleanSessions()
	if err != nil {
		t.Fatalf(
			"DiscoverCleanSessions() returned error: %v",
			err,
		)
	}

	if len(sessions) != 0 {
		t.Fatalf(
			"session count = %d, want %d",
			len(sessions),
			0,
		)
	}
}
