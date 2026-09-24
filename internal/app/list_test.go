package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

func TestListSessionsIncludesAllPersistentVersions(t *testing.T) {
	root := t.TempDir()
	for _, version := range []string{"v1", "v2"} {
		sessionDir := filepath.Join(root, "runtime", version, "session-"+version)
		if err := os.MkdirAll(sessionDir, 0700); err != nil {
			t.Fatalf("MkdirAll(%q) returned error: %v", sessionDir, err)
		}
		if err := sessionmeta.WriteMetadata(sessionDir, sessionmeta.Metadata{
			ID:                "id-" + version,
			PID:               os.Getpid(),
			ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
			ProcessGroupID:    sessionmeta.CurrentProcessGroupID(),
			Name:              "session-" + version,
			Version:           version,
			Mode:              sessionmeta.ModeNormal,
			CreatedAt:         time.Now(),
		}); err != nil {
			t.Fatalf("WriteMetadata(%q) returned error: %v", sessionDir, err)
		}
	}

	application := &App{
		Persistent: true,
		Paths: runtime.Paths{
			Root: root,
		},
	}
	sessions, err := application.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() returned error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("ListSessions() returned %d sessions, want 2", len(sessions))
	}
}
