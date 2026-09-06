package runtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

func assertSessionMetadata(
	t *testing.T,
	runtimePath string,
	sessionID string,
	mode sessionmeta.Mode,
) sessionmeta.Metadata {
	t.Helper()

	metadata, err := sessionmeta.ReadMetadata(runtimePath)
	if err != nil {
		t.Fatalf("ReadMetadata() returned error: %v", err)
	}

	if metadata.ID != sessionID {
		t.Fatalf(
			"metadata session ID = %q, want %q",
			metadata.ID,
			sessionID,
		)
	}

	if metadata.PID != os.Getpid() {
		t.Fatalf(
			"metadata PID = %d, want %d",
			metadata.PID,
			os.Getpid(),
		)
	}

	if metadata.ProcessStartTicks == 0 {
		t.Fatal("metadata process start time is zero")
	}

	if metadata.CreatedAt.IsZero() {
		t.Fatal("metadata creation time is zero")
	}

	if metadata.Version != "1.0.0" {
		t.Fatalf(
			"metadata version = %q, want %q",
			metadata.Version,
			"1.0.0",
		)
	}

	if metadata.Mode != mode {
		t.Fatalf(
			"metadata mode = %q, want %q",
			metadata.Mode,
			mode,
		)
	}

	return metadata
}

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

	assertSessionMetadata(
		t,
		session.Paths.Runtime,
		session.ID,
		sessionmeta.ModeNormal,
	)

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

	metadataPath := sessionmeta.MetadataPath(
		session.Paths.Runtime,
	)

	info, err := os.Stat(metadataPath)
	if err != nil {
		t.Fatalf("stat session metadata: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Fatalf(
			"session metadata permissions = %o, want 600",
			info.Mode().Perm(),
		)
	}
}

func TestSessionMetadataContainsIdentity(t *testing.T) {
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
	metadata, err := sessionmeta.ReadMetadata(
		session.Paths.Runtime,
	)
	if err != nil {
		t.Fatalf("ReadMetadata() returned error: %v", err)
	}

	if metadata.ID != session.ID {
		t.Fatalf(
			"metadata session ID = %q, want %q",
			metadata.ID,
			session.ID,
		)
	}

	if metadata.PID != os.Getpid() {
		t.Fatalf(
			"metadata PID = %d, want %d",
			metadata.PID,
			os.Getpid(),
		)
	}

	if metadata.ProcessStartTicks == 0 {
		t.Fatal("metadata process start time is zero")
	}

	if metadata.CreatedAt.IsZero() {
		t.Fatal("metadata creation time is zero")
	}

	if metadata.Version != "1.0.0" {
		t.Fatalf(
			"metadata version = %q, want %q",
			metadata.Version,
			"1.0.0",
		)
	}

	if metadata.Mode != sessionmeta.ModeNormal {
		t.Fatalf(
			"metadata mode = %q, want %q",
			metadata.Mode,
			sessionmeta.ModeNormal,
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

	assertSessionMetadata(
		t,
		session.Paths.Runtime,
		session.ID,
		sessionmeta.ModeNormal,
	)

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

	assertSessionMetadata(
		t,
		session.Paths.Runtime,
		session.ID,
		sessionmeta.ModeNormal,
	)

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

	metadata := sessionmeta.Metadata{
		ID:                "stale-session",
		PID:               999999999,
		ProcessStartTicks: 1,
		CreatedAt:         time.Unix(1, 0).UTC(),
		Version:           "1.0.0",
		Mode:              sessionmeta.ModeNormal,
	}
	if err := sessionmeta.WriteMetadata(
		sessionRuntime,
		metadata,
	); err != nil {
		t.Fatalf(
			"WriteMetadata() returned error: %v",
			err,
		)
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

	assertSessionMetadata(
		t,
		session.Paths.Runtime,
		session.ID,
		sessionmeta.ModeNormal,
	)

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

func TestSessionDefaultsToNormalMode(t *testing.T) {
	paths, err := NewPaths(t.TempDir(), "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	if session.Mode != SessionModeNormal {
		t.Fatalf(
			"session mode = %q, want %q",
			session.Mode,
			SessionModeNormal,
		)
	}
}

func TestSessionSupportsMultiplexerMode(t *testing.T) {
	paths, err := NewPaths(t.TempDir(), "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	if err := session.SetMode(SessionModeMultiplexer); err != nil {
		t.Fatalf("SetMode() returned error: %v", err)
	}

	if session.Mode != SessionModeMultiplexer {
		t.Fatalf(
			"session mode = %q, want %q",
			session.Mode,
			SessionModeMultiplexer,
		)
	}
}

func TestSessionRejectsUnknownMode(t *testing.T) {
	paths, err := NewPaths(t.TempDir(), "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	err = session.SetMode(SessionMode("unknown"))
	if err == nil {
		t.Fatal("SetMode() returned nil error")
	}
}

func TestCleanupStalePreservesMultiplexerSession(t *testing.T) {
	paths, err := NewPaths(t.TempDir(), "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	if err := session.SetMode(SessionModeMultiplexer); err != nil {
		t.Fatalf("SetMode() returned error: %v", err)
	}

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	assertSessionMetadata(
		t,
		session.Paths.Runtime,
		session.ID,
		sessionmeta.ModeMultiplexer,
	)

	if err := CleanupStale(paths); err != nil {
		t.Fatalf("CleanupStale() returned error: %v", err)
	}

	if _, err := os.Stat(session.Paths.Runtime); err != nil {
		t.Fatalf(
			"multiplexer runtime was removed as stale: %v",
			err,
		)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}
}

func TestNewSessionWithModeCreatesNormalSession(t *testing.T) {
	paths, err := NewPaths(t.TempDir(), "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSessionWithMode(
		paths,
		SessionModeNormal,
	)
	if err != nil {
		t.Fatalf("NewSessionWithMode() returned error: %v", err)
	}

	if session.Mode != SessionModeNormal {
		t.Fatalf(
			"session mode = %q, want %q",
			session.Mode,
			SessionModeNormal,
		)
	}
}

func TestNewSessionWithModeCreatesMultiplexerSession(t *testing.T) {
	paths, err := NewPaths(t.TempDir(), "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSessionWithMode(
		paths,
		SessionModeMultiplexer,
	)
	if err != nil {
		t.Fatalf("NewSessionWithMode() returned error: %v", err)
	}

	if session.Mode != SessionModeMultiplexer {
		t.Fatalf(
			"session mode = %q, want %q",
			session.Mode,
			SessionModeMultiplexer,
		)
	}
}

func TestNewSessionWithModeRejectsUnknownMode(t *testing.T) {
	paths, err := NewPaths(t.TempDir(), "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	_, err = NewSessionWithMode(
		paths,
		SessionMode("unknown"),
	)
	if err == nil {
		t.Fatal("NewSessionWithMode() returned nil error")
	}
}

func TestCleanupStaleIgnoresMultiplexerSessions(t *testing.T) {
	paths, err := NewPaths(t.TempDir(), "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := NewSessionWithMode(
		paths,
		SessionModeMultiplexer,
	)
	if err != nil {
		t.Fatalf("NewSessionWithMode() returned error: %v", err)
	}

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	assertSessionMetadata(
		t,
		session.Paths.Runtime,
		session.ID,
		sessionmeta.ModeMultiplexer,
	)

	if err := CleanupStale(paths); err != nil {
		t.Fatalf("CleanupStale() returned error: %v", err)
	}

	if _, err := os.Stat(session.Paths.Runtime); err != nil {
		t.Fatalf(
			"multiplexer session was removed by CleanupStale(): %v",
			err,
		)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}
}
