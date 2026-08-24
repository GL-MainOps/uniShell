package runtime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareCreatesIsolatedRuntimeDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	if session.Paths.Runtime == paths.Runtime {
		t.Fatal("session runtime must differ from version runtime")
	}

	if !IsWithinRoot(paths, session.Paths.Runtime) {
		t.Fatalf("session runtime escaped runtime root: %q", session.Paths.Runtime)
	}

	for _, path := range []string{
		paths.Root,
		paths.Runtime,
		session.Paths.Runtime,
		session.Paths.Bin,
		session.Paths.Config,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %q: %v", path, err)
		}

		if !info.IsDir() {
			t.Fatalf("%q is not a directory", path)
		}

		if info.Mode().Perm() != 0700 {
			t.Fatalf(
				"%q permissions = %o, want 700",
				path,
				info.Mode().Perm(),
			)
		}
	}

	markerPath := filepath.Join(
		session.Paths.Runtime,
		sessionMarkerName,
	)

	info, err := os.Stat(markerPath)
	if err != nil {
		t.Fatalf("stat session marker: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Fatalf(
			"session marker permissions = %o, want 600",
			info.Mode().Perm(),
		)
	}
}

func TestSessionMarkerContainsIdentity(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	markerPath := filepath.Join(
		session.Paths.Runtime,
		sessionMarkerName,
	)

	marker, err := readSessionMarker(markerPath)
	if err != nil {
		t.Fatalf("readSessionMarker() returned error: %v", err)
	}

	if marker.SessionID != session.ID {
		t.Fatalf(
			"marker session ID = %q, want %q",
			marker.SessionID,
			session.ID,
		)
	}

	if marker.PID != os.Getpid() {
		t.Fatalf(
			"marker PID = %d, want %d",
			marker.PID,
			os.Getpid(),
		)
	}

	if marker.ProcessStartTicks == 0 {
		t.Fatal("marker process start time is zero")
	}

	if marker.StartedAt == 0 {
		t.Fatal("marker start time is zero")
	}

	if marker.Version != "1.0.0" {
		t.Fatalf(
			"marker version = %q, want %q",
			marker.Version,
			"1.0.0",
		)
	}
}

func TestConcurrentSessionsAreIsolated(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	first, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession(first) returned error: %v", err)
	}

	second, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession(second) returned error: %v", err)
	}

	if first.ID == second.ID {
		t.Fatal("concurrent sessions received the same ID")
	}

	if first.Paths.Runtime == second.Paths.Runtime {
		t.Fatal("concurrent sessions share the same runtime path")
	}

	if err := first.Prepare(); err != nil {
		t.Fatalf("first Prepare() returned error: %v", err)
	}

	if err := second.Prepare(); err != nil {
		t.Fatalf("second Prepare() returned error: %v", err)
	}

	if _, err := os.Stat(first.Paths.Runtime); err != nil {
		t.Fatalf("first runtime disappeared: %v", err)
	}

	if _, err := os.Stat(second.Paths.Runtime); err != nil {
		t.Fatalf("second runtime disappeared: %v", err)
	}

	if err := first.Cleanup(); err != nil {
		t.Fatalf("first Cleanup() returned error: %v", err)
	}

	if _, err := os.Stat(second.Paths.Runtime); err != nil {
		t.Fatalf(
			"second runtime was affected by first cleanup: %v",
			err,
		)
	}

	if _, err := os.Stat(first.Paths.Runtime); !os.IsNotExist(err) {
		t.Fatalf("first runtime still exists: %q", first.Paths.Runtime)
	}

	if err := second.Cleanup(); err != nil {
		t.Fatalf("second Cleanup() returned error: %v", err)
	}
}

func TestCleanupRemovesOnlySessionRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}

	if _, err := os.Stat(session.Paths.Runtime); !os.IsNotExist(err) {
		t.Fatalf(
			"session runtime still exists: %q",
			session.Paths.Runtime,
		)
	}

	if _, err := os.Stat(paths.Runtime); err != nil {
		t.Fatalf(
			"version runtime should remain after session cleanup: %v",
			err,
		)
	}
}

func TestCleanupIsIdempotent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("first Cleanup() returned error: %v", err)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("second Cleanup() returned error: %v", err)
	}
}

func TestCleanupStaleRemovesDeadSession(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	if err := os.MkdirAll(paths.Runtime, 0700); err != nil {
		t.Fatalf("create version runtime: %v", err)
	}

	sessionRuntime := filepath.Join(paths.Runtime, "stale-session")

	if err := os.MkdirAll(
		filepath.Join(sessionRuntime, "bin"),
		0700,
	); err != nil {
		t.Fatalf("create stale runtime: %v", err)
	}

	marker := sessionMarker{
		SessionID:         "stale-session",
		PID:               999999999,
		ProcessStartTicks: 1,
		StartedAt:         1,
		Version:           "1.0.0",
	}

	data, err := json.Marshal(marker)
	if err != nil {
		t.Fatalf("marshal marker: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sessionRuntime, sessionMarkerName),
		data,
		0600,
	); err != nil {
		t.Fatalf("write stale marker: %v", err)
	}

	if err := CleanupStale(paths); err != nil {
		t.Fatalf("CleanupStale() returned error: %v", err)
	}

	if _, err := os.Stat(sessionRuntime); !os.IsNotExist(err) {
		t.Fatalf("stale session still exists: %q", sessionRuntime)
	}

	if _, err := os.Stat(paths.Runtime); err != nil {
		t.Fatalf("version runtime should remain: %v", err)
	}
}

func TestCleanupStalePreservesLiveSession(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	if err := CleanupStale(paths); err != nil {
		t.Fatalf("CleanupStale() returned error: %v", err)
	}

	if _, err := os.Stat(session.Paths.Runtime); err != nil {
		t.Fatalf("live session was removed: %v", err)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}
}

func TestCleanupStaleIgnoresUnmarkedDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	unknown := filepath.Join(paths.Runtime, "unknown")

	if err := os.MkdirAll(
		filepath.Join(unknown, "bin"),
		0700,
	); err != nil {
		t.Fatalf("create unknown runtime: %v", err)
	}

	if err := CleanupStale(paths); err != nil {
		t.Fatalf("CleanupStale() returned error: %v", err)
	}

	if _, err := os.Stat(unknown); err != nil {
		t.Fatalf(
			"unmarked directory should be preserved: %v",
			err,
		)
	}

	if err := os.RemoveAll(paths.Runtime); err != nil {
		t.Fatalf("cleanup test runtime: %v", err)
	}
}

func TestCleanupStaleIgnoresMissingRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	if err := CleanupStale(paths); err != nil {
		t.Fatalf("CleanupStale() returned error: %v", err)
	}
}

func TestIsWithinRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{
			name:   "runtime directory",
			target: paths.Runtime,
			want:   true,
		},
		{
			name:   "binary directory",
			target: paths.Bin,
			want:   true,
		},
		{
			name:   "file inside runtime",
			target: filepath.Join(paths.Bin, "fzf"),
			want:   true,
		},
		{
			name:   "outside root",
			target: filepath.Join(root, "..", "outside"),
			want:   false,
		},
		{
			name:   "filesystem root",
			target: string(filepath.Separator),
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsWithinRoot(paths, test.target); got != test.want {
				t.Fatalf(
					"IsWithinRoot(%q) = %v, want %v",
					test.target,
					got,
					test.want,
				)
			}
		})
	}
}
