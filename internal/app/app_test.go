package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gitlab.com/mainops/uniShell/internal/bundle"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	"gitlab.com/mainops/uniShell/internal/runtime"
	"gitlab.com/mainops/uniShell/internal/shell"
)

func TestNewUsesDefaultRuntimeRoot(t *testing.T) {
	t.Setenv("UNISHELL_RUNTIME_DIR", "")
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	want := "/var/tmp/.lesscache"

	if application.Paths.Root != want {
		t.Fatalf("runtime root = %q, want %q", application.Paths.Root, want)
	}
}

func TestNewDefaultsToNoMultiplexer(t *testing.T) {
	t.Setenv("UNISHELL_RUNTIME_DIR", "")
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if application.MultiplexerName != "" {
		t.Fatalf(
			"multiplexer name = %q, want empty",
			application.MultiplexerName,
		)
	}
}

func TestNewUsesEnvironmentRuntimeRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")
	t.Setenv("UNISHELL_RUNTIME_DIR", root)
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if application.Paths.Root != root {
		t.Fatalf("runtime root = %q, want %q", application.Paths.Root, root)
	}
}

func TestNewUsesExplicitRuntimeRoot(t *testing.T) {
	explicitRoot := filepath.Join(t.TempDir(), "explicit")
	envRoot := filepath.Join(t.TempDir(), "environment")

	t.Setenv("UNISHELL_RUNTIME_DIR", envRoot)
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
		Root:    explicitRoot,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if application.Paths.Root != explicitRoot {
		t.Fatalf("runtime root = %q, want %q", application.Paths.Root, explicitRoot)
	}
}

func TestNewBuildsVersionedRuntimePaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	application, err := New(Options{
		Version: "1.2.3",
		Commit:  "test",
		Root:    root,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	wantRuntime := filepath.Join(root, "runtime", "1.2.3")
	wantBin := filepath.Join(wantRuntime, "bin")
	wantConfig := filepath.Join(wantRuntime, "config")

	if application.Paths.Runtime != wantRuntime {
		t.Fatalf("runtime path = %q, want %q", application.Paths.Runtime, wantRuntime)
	}

	if application.Paths.Bin != wantBin {
		t.Fatalf("bin path = %q, want %q", application.Paths.Bin, wantBin)
	}

	if application.Paths.Config != wantConfig {
		t.Fatalf("config path = %q, want %q", application.Paths.Config, wantConfig)
	}
}

func TestNewRejectsEmptyVersion(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")
	_, err := New(Options{
		Version: "",
		Commit:  "test",
	})
	if err == nil {
		t.Fatal("New() returned nil error, want error")
	}
}

func TestNewRequiresAuthentication(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "")

	_, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})

	if err == nil {
		t.Fatal("New() returned nil error, want authentication error")
	}
}

func TestNewUsesEnvironmentAuthenticationToken(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "environment-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if application.AuthToken != "environment-token" {
		t.Fatalf(
			"auth token = %q, want %q",
			application.AuthToken,
			"environment-token",
		)
	}
}

func TestStartSessionCreatesIsolatedRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	t.Setenv("UNISHELL_AUTH_TOKEN", "test-fixture-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
		Root:    root,
		Bundle:  testBundleSource(t),
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	session, err := application.StartSession()
	if err != nil {
		t.Fatalf("StartSession() returned error: %v", err)
	}

	if session.ID == "" {
		t.Fatal("session ID is empty")
	}

	if session.Paths.Runtime == application.Paths.Runtime {
		t.Fatal("session runtime equals version runtime")
	}

	info, err := os.Stat(session.Paths.Runtime)
	if err != nil {
		t.Fatalf("stat session runtime: %v", err)
	}

	if !info.IsDir() {
		t.Fatal("session runtime is not a directory")
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}
}

func TestStartSessionSupportsConcurrentSessions(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	t.Setenv("UNISHELL_AUTH_TOKEN", "test-fixture-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
		Root:    root,
		Bundle:  testBundleSource(t),
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	first, err := application.StartSession()
	if err != nil {
		t.Fatalf("StartSession(first) returned error: %v", err)
	}

	second, err := application.StartSession()
	if err != nil {
		t.Fatalf("StartSession(second) returned error: %v", err)
	}

	if first.ID == second.ID {
		t.Fatal("concurrent sessions have identical IDs")
	}

	if first.Paths.Runtime == second.Paths.Runtime {
		t.Fatal("concurrent sessions share the same runtime")
	}

	if _, err := os.Stat(first.Paths.Runtime); err != nil {
		t.Fatalf("first session disappeared: %v", err)
	}

	if _, err := os.Stat(second.Paths.Runtime); err != nil {
		t.Fatalf("second session disappeared: %v", err)
	}

	if err := first.Cleanup(); err != nil {
		t.Fatalf("first Cleanup() returned error: %v", err)
	}

	if _, err := os.Stat(second.Paths.Runtime); err != nil {
		t.Fatalf(
			"second session was affected by first cleanup: %v",
			err,
		)
	}

	if err := second.Cleanup(); err != nil {
		t.Fatalf("second Cleanup() returned error: %v", err)
	}
}

func TestStartSessionRequiresAuthentication(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "")

	_, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
		Root:    t.TempDir(),
	})

	if err == nil {
		t.Fatal("New() returned nil error without authentication")
	}
}

func testBundleSource(t *testing.T) BundleSource {
	t.Helper()

	sourceDir := t.TempDir()

	if err := os.MkdirAll(
		filepath.Join(sourceDir, "config", "shell", "shared"),
		0o755,
	); err != nil {
		t.Fatalf("create test bundle shell configuration directory: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sourceDir, "config", "shell", "shared", "config.toml"),
		[]byte("[environment]\n"),
		0o644,
	); err != nil {
		t.Fatalf("write test bundle shell configuration: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sourceDir, "test-tool"),
		[]byte("test runtime payload\n"),
		0o755,
	); err != nil {
		t.Fatalf("write test runtime payload: %v", err)
	}

	data, err := bundle.Create(sourceDir, "test-fixture-token")
	if err != nil {
		t.Fatalf("create test bundle: %v", err)
	}

	return func() ([]byte, error) {
		return data, nil
	}
}

func TestStartSessionExtractsAuthenticatedBundle(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-fixture-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
		Root:    t.TempDir(),
		Bundle:  testBundleSource(t),
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	session, err := application.StartSession()
	if err != nil {
		t.Fatalf("StartSession() returned error: %v", err)
	}
	defer session.Cleanup()

	payload, err := os.ReadFile(
		filepath.Join(session.Paths.Runtime, "test-tool"),
	)
	if err != nil {
		t.Fatalf("read extracted test tool: %v", err)
	}

	if string(payload) != "test runtime payload\n" {
		t.Fatalf(
			"extracted payload = %q, want %q",
			string(payload),
			"test runtime payload\n",
		)
	}
}

func TestStartSessionRejectsWrongTokenAndCleansRuntime(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "wrong-test-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
		Root:    t.TempDir(),
		Bundle:  testBundleSource(t),
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	session, err := application.StartSession()
	if err == nil {
		if session != nil {
			_ = session.Cleanup()
		}

		t.Fatal("StartSession() returned nil error with wrong token")
	}

	if !errors.Is(err, credentials.ErrAuthenticationFailed) {
		t.Fatalf(
			"StartSession() error = %v, want authentication failure",
			err,
		)
	}
}

func TestStartSessionFailureLeavesNoSessionRuntime(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "wrong-test-token")

	root := t.TempDir()

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
		Root:    root,
		Bundle:  testBundleSource(t),
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	_, err = application.StartSession()
	if err == nil {
		t.Fatal("StartSession() returned nil error")
	}

	entries, err := os.ReadDir(application.Paths.Runtime)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}

		t.Fatalf("read runtime directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf(
				"failed session left runtime directory: %q",
				entry.Name(),
			)
		}
	}
}

func TestNewUsesProvidedMultiplexerManager(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	backend := &appTestBackend{}

	manager := multiplexer.NewManager(
		multiplexer.NewRegistry(backend),
	)

	application, err := New(Options{
		Version:     "1.0.0",
		Commit:      "test",
		Root:        t.TempDir(),
		Multiplexer: manager,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if application.Multiplexer != manager {
		t.Fatal("application did not retain provided multiplexer manager")
	}
}

type appTestBackend struct {
	created bool
}

func (appTestBackend) Name() string {
	return "test"
}

func (appTestBackend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (appTestBackend) Available() bool {
	return true
}

func (b *appTestBackend) Create(multiplexer.Session) error {
	b.created = true
	return nil
}

func (appTestBackend) Attach(multiplexer.Session) error {
	return nil
}

func (appTestBackend) Detach(multiplexer.Session) error {
	return nil
}

func (appTestBackend) IsAlive(multiplexer.Session) bool {
	return true
}

func (appTestBackend) Destroy(multiplexer.Session) error {
	return nil
}

func TestStartMultiplexerSessionCreatesManagedSession(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	t.Setenv("UNISHELL_AUTH_TOKEN", "test-fixture-token")

	backend := &appTestBackend{
		created: true,
	}

	manager := multiplexer.NewManager(
		multiplexer.NewRegistry(backend),
	)

	application, err := New(Options{
		Version:         "1.0.0",
		Commit:          "test",
		Root:            root,
		Bundle:          testBundleSource(t),
		Multiplexer:     manager,
		MultiplexerName: "test",
		SessionName:     "default",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	session, err := application.StartMultiplexerSession()
	if err != nil {
		t.Fatalf(
			"StartMultiplexerSession() returned error: %v",
			err,
		)
	}

	if session == nil {
		t.Fatal("StartMultiplexerSession() returned nil session")
	}

	if session.Runtime == nil {
		t.Fatal("runtime session is nil")
	}

	if session.Multiplexer == nil {
		t.Fatal("multiplexer session is nil")
	}

	if session.Runtime.Mode != runtime.SessionModeMultiplexer {
		t.Fatalf(
			"runtime session mode = %q, want %q",
			session.Runtime.Mode,
			runtime.SessionModeMultiplexer,
		)
	}

	if session.Multiplexer.Metadata.Name != "default" {
		t.Fatalf(
			"multiplexer session name = %q, want %q",
			session.Multiplexer.Metadata.Name,
			"default",
		)
	}

	if session.Multiplexer.Metadata.Multiplexer != "test" {
		t.Fatalf(
			"multiplexer = %q, want %q",
			session.Multiplexer.Metadata.Multiplexer,
			"test",
		)
	}

	if session.Multiplexer.Session.Runtime != session.Runtime.Paths.Runtime {
		t.Fatalf(
			"multiplexer runtime = %q, want %q",
			session.Multiplexer.Session.Runtime,
			session.Runtime.Paths.Runtime,
		)
	}

	if _, err := os.Stat(
		filepath.Join(
			session.Runtime.Paths.Runtime,
			"test-tool",
		),
	); err != nil {
		t.Fatalf(
			"extracted runtime payload is missing: %v",
			err,
		)
	}

	if !backend.created {
		t.Fatal("multiplexer backend Create() was not called")
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}
}

func TestCreateMultiplexerSessionUsesProvidedMultiplexer(
	t *testing.T,
) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-fixture-token")

	backend := &appTestBackend{}

	manager := multiplexer.NewManager(
		multiplexer.NewRegistry(backend),
	)

	application, err := New(Options{
		Version:     "1.0.0",
		Commit:      "test",
		Root:        t.TempDir(),
		Bundle:      testBundleSource(t),
		Multiplexer: manager,
		Shell:       "bash",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	runtimeSession := &runtime.Session{
		Paths: runtime.Paths{
			Bin:     t.TempDir(),
			Runtime: t.TempDir(),
		},
	}

	session, err := application.CreateMultiplexerSession(
		runtimeSession,
		"test",
		"bash",
		shell.Startup{},
	)
	if err != nil {
		t.Fatalf(
			"CreateMultiplexerSession() returned error: %v",
			err,
		)
	}

	if session == nil {
		t.Fatal("CreateMultiplexerSession() returned nil session")
	}

	if session.Multiplexer == nil {
		t.Fatal("multiplexer session is nil")
	}

	if session.Multiplexer.Metadata.Multiplexer != "test" {
		t.Fatalf(
			"multiplexer = %q, want %q",
			session.Multiplexer.Metadata.Multiplexer,
			"test",
		)
	}

	if !backend.created {
		t.Fatal("multiplexer backend Create() was not called")
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}
}

func TestCreateMultiplexerSessionPassesShellStartup(
	t *testing.T,
) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &appTestBackend{}

	manager := multiplexer.NewManager(
		multiplexer.NewRegistry(backend),
	)

	application, err := New(Options{
		Version:         "1.0.0",
		Commit:          "test",
		Root:            filepath.Dir(runtimePath),
		Multiplexer:     manager,
		MultiplexerName: "test",
		SessionName:     "default",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	runtimeSession := &runtime.Session{
		Paths: runtime.Paths{
			Runtime: runtimePath,
			Bin:     filepath.Join(runtimePath, "bin"),
		},
	}

	startup := shell.Startup{
		Args: []string{
			"-d",
		},
		Env: map[string]string{
			"ZDOTDIR": filepath.Join(
				runtimePath,
				"config",
				"shell",
				"zsh",
			),
		},
	}

	session, err := application.CreateMultiplexerSession(
		runtimeSession,
		"test",
		"zsh",
		startup,
	)
	if err != nil {
		t.Fatalf(
			"CreateMultiplexerSession() returned error: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		session.Multiplexer.Session.ShellArgs,
		startup.Args,
	) {
		t.Fatalf(
			"shell args = %#v, want %#v",
			session.Multiplexer.Session.ShellArgs,
			startup.Args,
		)
	}

	wantEnvironment := ""
	for _, entry := range session.Multiplexer.Session.Env {
		if strings.HasPrefix(entry, "ZDOTDIR=") {
			wantEnvironment = entry
			break
		}
	}

	want := "ZDOTDIR=" + startup.Env["ZDOTDIR"]

	if wantEnvironment != want {
		t.Fatalf(
			"ZDOTDIR environment = %q, want %q",
			wantEnvironment,
			want,
		)
	}
}

func TestNewLoadsMultiplexerOptionsFromEnvironment(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")
	t.Setenv(
		"UNISHELL_TMUX_OPTS",
		`-f "/tmp/my config" -L work`,
	)
	t.Setenv(
		"UNISHELL_ZELLIJ_OPTS",
		`--layout "compact layout.kdl"`,
	)

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	want := api.Options{
		Tmux: api.TmuxOptions{
			CreateArgs: []string{
				"-f",
				"/tmp/my config",
				"-L",
				"work",
			},
		},
		Zellij: api.ZellijOptions{
			CreateArgs: []string{
				"--layout",
				"compact layout.kdl",
			},
		},
	}

	if !reflect.DeepEqual(
		application.MultiplexerOptions,
		want,
	) {
		t.Fatalf(
			"multiplexer options = %#v, want %#v",
			application.MultiplexerOptions,
			want,
		)
	}
}

func TestNewRejectsMalformedMultiplexerOptions(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")
	t.Setenv(
		"UNISHELL_TMUX_OPTS",
		`--name "unterminated`,
	)
	t.Setenv(
		"UNISHELL_ZELLIJ_OPTS",
		"",
	)

	_, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})

	if err == nil {
		t.Fatal(
			"New() returned nil error, want multiplexer option error",
		)
	}
}

func TestNewUsesExplicitMultiplexerOptionsOverEnvironment(
	t *testing.T,
) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")
	t.Setenv(
		"UNISHELL_TMUX_OPTS",
		`-L environment`,
	)
	t.Setenv(
		"UNISHELL_ZELLIJ_OPTS",
		`--layout environment.kdl`,
	)

	want := api.Options{
		Tmux: api.TmuxOptions{
			CreateArgs: []string{
				"-L",
				"explicit",
			},
		},
	}

	application, err := New(Options{
		Version:            "1.0.0",
		Commit:             "test",
		MultiplexerOptions: want,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if !reflect.DeepEqual(
		application.MultiplexerOptions,
		want,
	) {
		t.Fatalf(
			"multiplexer options = %#v, want %#v",
			application.MultiplexerOptions,
			want,
		)
	}
}
