package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"gitlab.com/mainops/uniShell/internal/bundle"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	"gitlab.com/mainops/uniShell/internal/runtime"
	"gitlab.com/mainops/uniShell/internal/shell"

	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
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
	created         bool
	processIdentity sessionmeta.ProcessIdentity
}

func startAppProcessGroupHelper(t *testing.T) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(
		os.Args[0],
		"-test.run=^TestAppProcessGroupHelper$",
	)
	cmd.Env = append(
		os.Environ(),
		"UNISHELL_APP_PROCESS_GROUP_HELPER=1",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf(
			"failed to start app process-group helper: %v",
			err,
		)
	}

	return cmd
}

func TestAppProcessGroupHelper(t *testing.T) {
	if os.Getenv("UNISHELL_APP_PROCESS_GROUP_HELPER") != "1" {
		return
	}

	for {
		time.Sleep(time.Second)
	}
}

func appProcessIdentity(
	t *testing.T,
	cmd *exec.Cmd,
) sessionmeta.ProcessIdentity {
	t.Helper()

	startTicks, err := sessionmeta.ProcessStartTicks(cmd.Process.Pid)
	if err != nil {
		t.Fatalf(
			"ProcessStartTicks() returned error: %v",
			err,
		)
	}

	processGroupID, err := sessionmeta.ProcessGroupID(
		cmd.Process.Pid,
	)
	if err != nil {
		t.Fatalf(
			"ProcessGroupID() returned error: %v",
			err,
		)
	}

	return sessionmeta.ProcessIdentity{
		PID:               cmd.Process.Pid,
		ProcessStartTicks: startTicks,
		ProcessGroupID:    processGroupID,
	}
}

func (b *appTestBackend) ProcessIdentity(
	multiplexer.Session,
) (sessionmeta.ProcessIdentity, error) {
	return b.processIdentity, nil
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

func (appTestBackend) AvailableForSession(
	multiplexer.Session,
) bool {
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

	helper := startAppProcessGroupHelper(t)
	defer func() {
		_ = helper.Process.Kill()
		_ = helper.Wait()
	}()

	backend := &appTestBackend{
		created:         true,
		processIdentity: appProcessIdentity(t, helper),
	}

	manager := multiplexer.NewManager(
		multiplexer.NewRegistry(backend),
	)

	application, err := New(Options{
		Version:              "1.0.0",
		Commit:               "test",
		Root:                 root,
		Bundle:               testBundleSource(t),
		Multiplexer:          manager,
		MultiplexerName:      "test",
		SessionName:          "default",
		SessionNameSpecified: true,
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

	wantSessionName := "default@" + session.Runtime.ID[:7]
	if session.Multiplexer.Metadata.Name != wantSessionName {
		t.Fatalf(
			"multiplexer session name = %q, want %q",
			session.Multiplexer.Metadata.Name,
			wantSessionName,
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

	helper := startAppProcessGroupHelper(t)
	defer func() {
		_ = helper.Process.Kill()
		_ = helper.Wait()
	}()

	backend := &appTestBackend{
		processIdentity: appProcessIdentity(t, helper),
	}

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

	helper := startAppProcessGroupHelper(t)
	defer func() {
		_ = helper.Process.Kill()
		_ = helper.Wait()
	}()

	backend := &appTestBackend{
		processIdentity: appProcessIdentity(t, helper),
	}

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

func TestStartSessionUsesGeneratedSessionName(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-fixture-token")

	application, err := New(Options{
		Version:              "1.0.0",
		Commit:               "test",
		Root:                 t.TempDir(),
		Bundle:               testBundleSource(t),
		SessionName:          "direct-shell",
		SessionNameSpecified: true,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	session, err := application.StartSession()
	if err != nil {
		t.Fatalf("StartSession() returned error: %v", err)
	}
	defer session.Cleanup()

	metadata, err := sessionmeta.ReadMetadata(
		session.Paths.Runtime,
	)
	if err != nil {
		t.Fatalf("ReadMetadata() returned error: %v", err)
	}

	wantPrefix := "direct-shell@"
	if !strings.HasPrefix(metadata.Name, wantPrefix) {
		t.Fatalf(
			"metadata name = %q, want prefix %q",
			metadata.Name,
			wantPrefix,
		)
	}

	wantSuffix := session.ID[:7]
	if metadata.Name != "direct-shell@"+wantSuffix {
		t.Fatalf(
			"metadata name = %q, want %q",
			metadata.Name,
			"direct-shell@"+wantSuffix,
		)
	}

	if metadata.ID != session.ID {
		t.Fatalf(
			"metadata ID = %q, want %q",
			metadata.ID,
			session.ID,
		)
	}
}

func TestStartSessionUsesUnnamedGeneratedSessionName(t *testing.T) {
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

	metadata, err := sessionmeta.ReadMetadata(
		session.Paths.Runtime,
	)
	if err != nil {
		t.Fatalf("ReadMetadata() returned error: %v", err)
	}

	want := "unnamed@" + session.ID[:7]
	if metadata.Name != want {
		t.Fatalf(
			"metadata name = %q, want %q",
			metadata.Name,
			want,
		)
	}
}

func TestSessionNameForRuntime(t *testing.T) {
	runtimeSession := &runtime.Session{
		ID: "81cac8cbf90a225223b66c0898c8d54f",
	}

	tests := []struct {
		name          string
		specifiedName string
		specified     bool
		want          string
	}{
		{
			name:      "unspecified",
			specified: false,
			want:      "unnamed@81cac8c",
		},
		{
			name:          "specified",
			specifiedName: "unishell_dev_session",
			specified:     true,
			want:          "unishell_dev_session@81cac8c",
		},
		{
			name:          "specified without explicit flag for compatibility",
			specifiedName: "default",
			specified:     false,
			want:          "unnamed@81cac8c",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := sessionNameForRuntime(
				runtimeSession,
				test.specifiedName,
				test.specified,
			)
			if err != nil {
				t.Fatalf(
					"sessionNameForRuntime() returned error: %v",
					err,
				)
			}

			if got != test.want {
				t.Fatalf(
					"sessionNameForRuntime() = %q, want %q",
					got,
					test.want,
				)
			}
		})
	}
}
