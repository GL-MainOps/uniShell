package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	"gitlab.com/mainops/uniShell/internal/persistence"
	"gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
	"gitlab.com/mainops/uniShell/internal/shell"
)

func writeShellTestSharedConfig(
	t *testing.T,
	runtimeSession *runtime.Session,
	content string,
) {
	t.Helper()

	sharedDir := filepath.Join(
		runtimeSession.Paths.Runtime,
		"config",
		"shell",
		"shared",
	)

	if err := os.WriteFile(
		filepath.Join(sharedDir, "config.toml"),
		[]byte(content),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}
}

func TestNewApplicationPropagatesShellConfigurationOptions(
	t *testing.T,
) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")
	options := cliOptions{
		RuntimeDir:             t.TempDir(),
		Shell:                  "bash",
		ShellProfile:           "work",
		NoSharedRC:             true,
		Multiplexer:            "tmux",
		SessionName:            "test-session",
		MultiplexerSessionName: "test-multiplexer-session",
	}

	application, err := newApplication(options)
	if err != nil {
		t.Fatalf("newApplication() returned error: %v", err)
	}

	if got := application.RequestedShellProfile(); got != "work" {
		t.Fatalf(
			"RequestedShellProfile() = %q, want %q",
			got,
			"work",
		)
	}

	if got := application.RequestedNoSharedRC(); !got {
		t.Fatal("RequestedNoSharedRC() = false, want true")
	}
}

func TestPrintErrorAuthenticationFailed(t *testing.T) {
	originalStderr := os.Stderr

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stderr = writer

	printError(credentials.ErrAuthenticationFailed)

	_ = writer.Close()
	os.Stderr = originalStderr

	var output bytes.Buffer

	_, err = output.ReadFrom(reader)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}

	want := "Authentication Failed. Aborting...\n"

	if output.String() != want {
		t.Fatalf(
			"output = %q, want %q",
			output.String(),
			want,
		)
	}
}

func TestPrintErrorGenericError(t *testing.T) {
	originalStderr := os.Stderr

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stderr = writer

	printError(errors.New("test error"))

	_ = writer.Close()
	os.Stderr = originalStderr

	var output bytes.Buffer

	_, err = output.ReadFrom(reader)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}

	want := "uniShell: test error\n"

	if output.String() != want {
		t.Fatalf(
			"output = %q, want %q",
			output.String(),
			want,
		)
	}
}

func TestExitCodeReturnsZeroForNil(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Fatalf("exitCode(nil) = %d, want 0", got)
	}
}

func TestExitCodeSuppressesSIGINTStatus(t *testing.T) {
	err := exec.Command("sh", "-c", "exit 130").Run()
	if err == nil {
		t.Fatal("command returned nil error")
	}

	if got := exitCode(err); got != 0 {
		t.Fatalf("exitCode(130) = %d, want 0", got)
	}
}

func TestExitCodePreservesNonSIGINTStatus(t *testing.T) {
	err := exec.Command("sh", "-c", "exit 42").Run()
	if err == nil {
		t.Fatal("command returned nil error")
	}

	if got := exitCode(err); got != 42 {
		t.Fatalf("exitCode(42) = %d, want 42", got)
	}
}

func TestPrintErrorSuppressesSIGINTStatus(t *testing.T) {
	err := exec.Command("sh", "-c", "exit 130").Run()
	if err == nil {
		t.Fatal("command returned nil error")
	}

	originalStderr := os.Stderr

	reader, writer, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("create stderr pipe: %v", pipeErr)
	}

	os.Stderr = writer

	printError(err)

	_ = writer.Close()
	os.Stderr = originalStderr

	var output bytes.Buffer
	if _, err := output.ReadFrom(reader); err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}

	if output.Len() != 0 {
		t.Fatalf(
			"printError(130) output = %q, want empty output",
			output.String(),
		)
	}
}

func shellTestRuntime(t *testing.T) *runtime.Session {
	t.Helper()

	root := t.TempDir()

	paths, err := runtime.NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session, err := runtime.NewSession(paths)
	if err != nil {
		t.Fatalf("NewSession() returned error: %v", err)
	}

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	sharedDir := filepath.Join(
		session.Paths.Runtime,
		"config",
		"shell",
		"shared",
	)

	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sharedDir, "config.toml"),
		[]byte("[environment]\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	return session
}

func TestSetSessionEnvironmentInitializesNilEnvironment(
	t *testing.T,
) {
	startup := shell.Startup{}

	sessionEnvironment := map[string]string{
		"UNISHELL_SESSION_ID":         "session-id",
		"UNISHELL_SESSION_MODE":       "normal",
		"UNISHELL_SESSION_SHELL_NAME": "bash",
	}

	startup = setSessionEnvironment(
		startup,
		sessionEnvironment,
	)

	for key, want := range sessionEnvironment {
		if got := startup.Env[key]; got != want {
			t.Fatalf(
				"%s = %q, want %q",
				key,
				got,
				want,
			)
		}
	}
}

type shellTestApplication struct {
	discoverSession              *app.Session
	discoverSessions             []*multiplexer.ManagedSession
	discoverCleanSessions        []*app.CleanSession
	discoverCleanSessionCalls    int
	discoverCleanSessionResult   [][]*app.CleanSession
	terminateNormalSessionErr    error
	terminatedCleanSession       *app.CleanSession
	terminatedCleanSessions      []*app.CleanSession
	cleanedMultiplexerSession    *app.CleanSession
	cleanedMultiplexerSessions   []*app.CleanSession
	cleanupMultiplexerSessionErr error
	discoverErr                  error
	startSession                 *app.Session
	startErr                     error
	preparedSession              *runtime.Session
	preparedErr                  error
	createdSession               *app.Session
	createdErr                   error
	requestedShell               string
	requestedMultiplexer         string
	runtimeSession               *runtime.Session
	runtimeSessionErr            error
	authErr                      error
	createdMultiplexer           string
	createdStartup               shell.Startup
}

func (a *shellTestApplication) StartMultiplexerSession() (*app.Session, error) {
	return a.startSession, a.startErr
}

func (a *shellTestApplication) ValidateAuthentication() error {
	return a.authErr
}

func (a *shellTestApplication) StartSession() (
	*runtime.Session,
	error,
) {
	return a.runtimeSession, a.runtimeSessionErr
}

func (a *shellTestApplication) RequestedMultiplexer() string {
	return a.requestedMultiplexer
}

func (a *shellTestApplication) DiscoverMultiplexerSession() (*app.Session, error) {
	return a.discoverSession, a.discoverErr
}

func (a *shellTestApplication) DiscoverMultiplexerSessions() (
	[]*multiplexer.ManagedSession,
	error,
) {
	if errors.Is(a.discoverErr, multiplexer.ErrSessionNotFound) {
		return nil, nil
	}
	if a.discoverErr != nil {
		return nil, a.discoverErr
	}

	if a.discoverSessions != nil {
		return a.discoverSessions, nil
	}

	if a.discoverSession == nil ||
		a.discoverSession.Multiplexer == nil {
		return nil, nil
	}

	return []*multiplexer.ManagedSession{
		a.discoverSession.Multiplexer,
	}, nil
}

func (a *shellTestApplication) RequestedShell() string {
	return a.requestedShell
}

func (a *shellTestApplication) RequestedShellProfile() string {
	return ""
}

func (a *shellTestApplication) RequestedNoSharedRC() bool {
	return false
}

func (a *shellTestApplication) PrepareMultiplexerSession() (
	*runtime.Session,
	error,
) {
	return a.preparedSession, a.preparedErr
}

func (a *shellTestApplication) CreateMultiplexerSessionResolved(
	runtimeSession *runtime.Session,
	multiplexerName string,
	selectedShell shell.Shell,
	startup shell.Startup,
) (*app.Session, error) {
	a.createdMultiplexer = multiplexerName
	a.createdStartup = startup
	return a.createdSession, a.createdErr
}

func (a *shellTestApplication) CreateMultiplexerSession(
	runtimeSession *runtime.Session,
	multiplexerName string,
	shellName string,
	startup shell.Startup,
) (*app.Session, error) {
	a.createdMultiplexer = multiplexerName
	a.createdStartup = startup
	return a.createdSession, a.createdErr
}

func (a *shellTestApplication) DiscoverCleanSessions() (
	[]*app.CleanSession,
	error,
) {
	if a.discoverErr != nil {
		return nil, a.discoverErr
	}

	if a.discoverCleanSessionCalls <
		len(a.discoverCleanSessionResult) {
		result := a.discoverCleanSessionResult[a.discoverCleanSessionCalls]
		a.discoverCleanSessionCalls++

		return result, nil
	}

	a.discoverCleanSessionCalls++

	return a.discoverCleanSessions, nil
}

func (a *shellTestApplication) TerminateNormalSession(
	session *app.CleanSession,
) error {
	a.terminatedCleanSession = session
	a.terminatedCleanSessions = append(
		a.terminatedCleanSessions,
		session,
	)

	return a.terminateNormalSessionErr
}

func (a *shellTestApplication) CleanupMultiplexerSession(
	session *app.CleanSession,
) error {
	a.cleanedMultiplexerSession = session
	a.cleanedMultiplexerSessions = append(
		a.cleanedMultiplexerSessions,
		session,
	)

	return a.cleanupMultiplexerSessionErr
}

type shellTestBackend struct {
	attached  bool
	detached  bool
	destroyed bool
	alive     bool
	attachErr error
	detachErr error
}

func (b *shellTestBackend) Name() string {
	return "test"
}

func (b *shellTestBackend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *shellTestBackend) Available() bool {
	return true
}

func (b *shellTestBackend) AvailableForSession(
	multiplexer.Session,
) bool {
	return true
}

func (b *shellTestBackend) ResolveEndpoint(
	_ string,
	_ api.Options,
) (string, error) {
	return "", nil
}

func (b *shellTestBackend) Create(multiplexer.Session) error {
	return nil
}

func (b *shellTestBackend) ProcessIdentity(
	multiplexer.Session,
) (sessionmeta.ProcessIdentity, error) {
	return sessionmeta.ProcessIdentity{
		PID:               os.Getpid(),
		ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
		ProcessGroupID:    sessionmeta.CurrentProcessGroupID(),
	}, nil
}

func (b *shellTestBackend) Attach(multiplexer.Session) error {
	b.attached = true
	return b.attachErr
}

func (b *shellTestBackend) Detach(multiplexer.Session) error {
	b.detached = true
	return b.detachErr
}

func (b *shellTestBackend) IsAlive(multiplexer.Session) bool {
	return b.alive
}

func (b *shellTestBackend) Destroy(multiplexer.Session) error {
	b.destroyed = true
	return nil
}

type cleanLifecycleBackend struct {
	alive     bool
	destroyed bool
	identity  sessionmeta.ProcessIdentity
}

func (b *cleanLifecycleBackend) Name() string {
	return "test"
}

func (b *cleanLifecycleBackend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *cleanLifecycleBackend) Available() bool {
	return true
}

func (b *cleanLifecycleBackend) AvailableForSession(
	multiplexer.Session,
) bool {
	return true
}

func (b *cleanLifecycleBackend) ResolveEndpoint(
	_ string,
	_ api.Options,
) (string, error) {
	return "", nil
}

func (b *cleanLifecycleBackend) Create(
	multiplexer.Session,
) error {
	b.alive = true
	return nil
}

func (b *cleanLifecycleBackend) ProcessIdentity(
	multiplexer.Session,
) (sessionmeta.ProcessIdentity, error) {
	return b.identity, nil
}

func (b *cleanLifecycleBackend) Attach(
	multiplexer.Session,
) error {
	return nil
}

func (b *cleanLifecycleBackend) Detach(
	multiplexer.Session,
) error {
	return nil
}

func (b *cleanLifecycleBackend) IsAlive(
	multiplexer.Session,
) bool {
	return b.alive
}

func (b *cleanLifecycleBackend) Destroy(
	multiplexer.Session,
) error {
	b.destroyed = true
	b.alive = false
	return nil
}

func startCleanLifecycleProcessGroupHelper(
	t *testing.T,
) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(
		os.Args[0],
		"-test.run=^TestCleanLifecycleProcessGroupHelper$",
	)

	cmd.Env = append(
		os.Environ(),
		"UNISHELL_CLEAN_LIFECYCLE_HELPER=1",
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf(
			"start clean lifecycle process-group helper: %v",
			err,
		)
	}

	return cmd
}

func TestCleanLifecycleProcessGroupHelper(
	t *testing.T,
) {
	if os.Getenv("UNISHELL_CLEAN_LIFECYCLE_HELPER") != "1" {
		return
	}

	for {
		time.Sleep(time.Second)
	}
}

func cleanLifecycleProcessIdentity(
	t *testing.T,
	cmd *exec.Cmd,
) sessionmeta.ProcessIdentity {
	t.Helper()

	startTicks, err := sessionmeta.ProcessStartTicks(
		cmd.Process.Pid,
	)
	if err != nil {
		t.Fatalf(
			"ProcessStartTicks(%d) returned error: %v",
			cmd.Process.Pid,
			err,
		)
	}

	processGroupID, err := sessionmeta.ProcessGroupID(
		cmd.Process.Pid,
	)
	if err != nil {
		t.Fatalf(
			"ProcessGroupID(%d) returned error: %v",
			cmd.Process.Pid,
			err,
		)
	}

	return sessionmeta.ProcessIdentity{
		PID:               cmd.Process.Pid,
		ProcessStartTicks: startTicks,
		ProcessGroupID:    processGroupID,
	}
}

func TestRunCleanSelectsAllSessionsWhenRequested(
	t *testing.T,
) {
	first := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:   "first-id",
			Name: "first",
			Mode: sessionmeta.ModeNormal,
		},
		RuntimeDir: "/tmp/first",
	}

	second := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:   "second-id",
			Name: "second",
			Mode: sessionmeta.ModeNormal,
		},
		RuntimeDir: "/tmp/second",
	}

	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			first,
			second,
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}
	defer reader.Close()

	if _, err := writer.WriteString("a\ny\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	if err := runClean(application, nil); err != nil {
		t.Fatalf(
			"runClean() returned error: %v",
			err,
		)
	}

	if len(application.terminatedCleanSessions) != 2 {
		t.Fatalf(
			"terminated clean sessions = %d, want 2",
			len(application.terminatedCleanSessions),
		)
	}

	if application.terminatedCleanSessions[0] != first {
		t.Fatal(
			"runClean() did not clean the first selected session",
		)
	}

	if application.terminatedCleanSessions[1] != second {
		t.Fatal(
			"runClean() did not clean the second selected session",
		)
	}
}

func TestRunCleanDoesNotAcceptAllSelectionForSingleSession(
	t *testing.T,
) {
	target := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:   "development-id",
			Name: "development",
			Mode: sessionmeta.ModeNormal,
		},
		RuntimeDir: "/tmp/development",
	}

	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			target,
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}
	defer reader.Close()

	if _, err := writer.WriteString("a\nq\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	if err := runClean(application, nil); err != nil {
		t.Fatalf(
			"runClean() returned error: %v",
			err,
		)
	}

	if len(application.terminatedCleanSessions) != 0 {
		t.Fatalf(
			"terminated clean sessions = %d, want 0",
			len(application.terminatedCleanSessions),
		)
	}
}

func TestRunCleanRevalidatesAllSessionsBeforeCleanup(
	t *testing.T,
) {
	first := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:   "first-id",
			Name: "first",
			Mode: sessionmeta.ModeNormal,
		},
		RuntimeDir: "/tmp/first",
	}

	second := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:   "second-id",
			Name: "second",
			Mode: sessionmeta.ModeNormal,
		},
		RuntimeDir: "/tmp/second",
	}

	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			first,
			second,
		},
		discoverCleanSessionResult: [][]*app.CleanSession{
			{
				first,
				second,
			},
			{
				first,
			},
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}
	defer reader.Close()

	if _, err := writer.WriteString("a\ny\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	if err := runClean(application, nil); err == nil {
		t.Fatal(
			"runClean() returned nil after all-session revalidation failure",
		)
	}

	if len(application.terminatedCleanSessions) != 0 {
		t.Fatalf(
			"cleanup began before all sessions were revalidated: %d sessions",
			len(application.terminatedCleanSessions),
		)
	}
}

func TestRunCleanContinuesAfterAllSessionCleanupFailure(
	t *testing.T,
) {
	first := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:   "first-id",
			Name: "first",
			Mode: sessionmeta.ModeNormal,
		},
		RuntimeDir: "/tmp/first",
	}

	second := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:   "second-id",
			Name: "second",
			Mode: sessionmeta.ModeNormal,
		},
		RuntimeDir: "/tmp/second",
	}

	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			first,
			second,
		},
		terminateNormalSessionErr: errors.New(
			"normal cleanup failed",
		),
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}
	defer reader.Close()

	if _, err := writer.WriteString("a\ny\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	runErr := runClean(application, nil)
	if runErr == nil {
		t.Fatal(
			"runClean() returned nil after cleanup failure",
		)
	}

	if len(application.terminatedCleanSessions) != 2 {
		t.Fatalf(
			"terminated clean sessions = %d, want 2",
			len(application.terminatedCleanSessions),
		)
	}

	if !strings.Contains(
		runErr.Error(),
		"normal cleanup failed",
	) {
		t.Fatalf(
			"runClean() error = %q, want cleanup failure",
			runErr,
		)
	}
}

func TestRunCleanTerminatesConfirmedMultiplexerSessionEndToEnd(
	t *testing.T,
) {
	root := t.TempDir()
	runtimeRoot := filepath.Join(
		root,
		"runtime",
		"1.0.0",
	)

	runtimePath := filepath.Join(
		runtimeRoot,
		"session",
	)

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf(
			"create runtime path %q: %v",
			runtimePath,
			err,
		)
	}

	cmd := startCleanLifecycleProcessGroupHelper(t)
	defer func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
		}

		_ = cmd.Wait()
	}()

	identity := cleanLifecycleProcessIdentity(t, cmd)

	backend := &cleanLifecycleBackend{
		identity: identity,
		alive:    true,
	}

	manager := multiplexer.NewManager(
		multiplexer.NewRegistry(backend),
	)

	t.Setenv(
		"UNISHELL_AUTH_TOKEN",
		"test-fixture-token",
	)

	application, err := app.New(app.Options{
		Version:     "1.0.0",
		Commit:      "test",
		Root:        root,
		Multiplexer: manager,
	})
	if err != nil {
		t.Fatalf(
			"app.New() returned error: %v",
			err,
		)
	}

	_, err = manager.Create(
		"test",
		"development",
		"",
		runtimePath,
		"",
		"",
		nil,
		nil,
		api.Options{},
	)
	if err != nil {
		t.Fatalf(
			"manager.Create() returned error: %v",
			err,
		)
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf(
			"os.Pipe() returned error: %v",
			err,
		)
	}

	defer reader.Close()

	if _, err := writer.WriteString("y\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	if err := runClean(
		application,
		[]string{"--target", "development"},
	); err != nil {
		t.Fatalf(
			"runClean() returned error: %v",
			err,
		)
	}

	if !backend.destroyed {
		t.Fatal(
			"runClean() did not destroy the multiplexer backend",
		)
	}

	if _, err := os.Stat(runtimePath); !os.IsNotExist(err) {
		t.Fatalf(
			"runtime path still exists after clean: %v",
			err,
		)
	}

	if err := cmd.Wait(); err == nil {
		t.Fatal(
			"managed process-group helper remained alive after clean",
		)
	}
}

func TestRunCleanTerminatesConfirmedMultiplexerSession(
	t *testing.T,
) {
	target := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:                "development-id",
			PID:               os.Getpid(),
			ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
			ProcessGroupID:    sessionmeta.CurrentProcessGroupID(),
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeMultiplexer,
			Name:              "development",
			Multiplexer:       "test",
		},
		RuntimeDir: "/tmp/development",
	}

	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			target,
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf(
			"os.Pipe() returned error: %v",
			err,
		)
	}

	defer reader.Close()

	if _, err := writer.WriteString("y\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	if err := runClean(
		application,
		[]string{"--target", "development"},
	); err != nil {
		t.Fatalf(
			"runClean() returned error: %v",
			err,
		)
	}

	if application.cleanedMultiplexerSession != target {
		t.Fatal(
			"runClean() did not clean the confirmed multiplexer session",
		)
	}

	if application.terminatedCleanSession != nil {
		t.Fatal(
			"runClean() incorrectly used normal-session termination",
		)
	}
}

func TestRunCleanReportsMultiplexerCleanupError(
	t *testing.T,
) {
	target := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:                "development-id",
			PID:               os.Getpid(),
			ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
			ProcessGroupID:    sessionmeta.CurrentProcessGroupID(),
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeMultiplexer,
			Name:              "development",
			Multiplexer:       "test",
		},
		RuntimeDir: "/tmp/development",
	}

	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			target,
		},
		cleanupMultiplexerSessionErr: errors.New(
			"multiplexer cleanup failed",
		),
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf(
			"os.Pipe() returned error: %v",
			err,
		)
	}

	defer reader.Close()

	if _, err := writer.WriteString("y\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	err = runClean(
		application,
		[]string{"--target", "development"},
	)

	if err == nil {
		t.Fatal(
			"runClean() returned nil after multiplexer cleanup failure",
		)
	}

	if !strings.Contains(
		err.Error(),
		"multiplexer cleanup failed",
	) {
		t.Fatalf(
			"runClean() error = %q, want multiplexer cleanup failure",
			err,
		)
	}
}

func TestRunShellAttachesExistingSession(t *testing.T) {
	backend := &shellTestBackend{
		alive: true,
	}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Metadata: sessionmeta.Metadata{
				ShellName: "bash",
			},
			Backend: backend,
			Session: multiplexer.Session{
				Name:      "default",
				Endpoint:  "/tmp/test.sock",
				ShellName: "bash",
			},
		},
	}

	application := &shellTestApplication{
		discoverSession:      session,
		requestedShell:       "zsh",
		requestedMultiplexer: "tmux",
	}

	if err := runShell(application, nil); err != nil {
		t.Fatalf("runShell() returned error: %v", err)
	}

	if !backend.attached {
		t.Fatal("runShell() did not attach existing session")
	}

	if backend.destroyed {
		t.Fatal("runShell() destroyed existing session")
	}
}

func TestRunShellCreatesAndAttachesWhenSessionDoesNotExist(
	t *testing.T,
) {
	backend := &shellTestBackend{
		alive: true,
	}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/tmp/test.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverErr:          multiplexer.ErrSessionNotFound,
		preparedSession:      shellTestRuntime(t),
		createdSession:       session,
		requestedShell:       "bash",
		requestedMultiplexer: "tmux",
	}

	writeShellTestSharedConfig(
		t,
		application.preparedSession,
		"[environment]\n"+
			"TEST_SESSION_ID = \"${UNISHELL_SESSION_ID}\"\n"+
			"TEST_RUNTIME_DIR = \"${UNISHELL_SESSION_RUNTIME_DIR}\"\n",
	)

	if err := runShell(application, nil); err != nil {
		t.Fatalf("runShell() returned error: %v", err)
	}

	if !backend.attached {
		t.Fatal("runShell() did not attach newly created session")
	}

	startupPath := application.createdStartup.Args[2]

	data, err := os.ReadFile(startupPath)
	if err != nil {
		t.Fatalf(
			"ReadFile(%q) returned error: %v",
			startupPath,
			err,
		)
	}

	wantSessionID := application.preparedSession.ID

	if !strings.Contains(
		string(data),
		"export TEST_SESSION_ID='"+wantSessionID+"'",
	) {
		t.Fatalf(
			"startup file does not contain resolved session ID: %q",
			string(data),
		)
	}

	if !strings.Contains(
		string(data),
		"export TEST_RUNTIME_DIR='"+application.preparedSession.Paths.Runtime+"'",
	) {
		t.Fatalf(
			"startup file does not contain resolved runtime directory: %q",
			string(data),
		)
	}

	if got := application.createdStartup.Env["UNISHELL_SESSION_ID"]; got != wantSessionID {
		t.Fatalf(
			"startup session ID = %q, want %q",
			got,
			wantSessionID,
		)
	}

	if got := application.createdStartup.Env[shell.SessionRuntimeDirEnvName]; got != application.preparedSession.Paths.Runtime {
		t.Fatalf(
			"startup runtime directory = %q, want %q",
			got,
			application.preparedSession.Paths.Runtime,
		)
	}

	if backend.destroyed {
		t.Fatal("runShell() destroyed successfully attached session")
	}
}

func TestRunShellCleansNewSessionWhenAttachFails(t *testing.T) {
	attachErr := errors.New("attach failed")

	backend := &shellTestBackend{
		attachErr: attachErr,
	}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/tmp/test.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverErr:          multiplexer.ErrSessionNotFound,
		preparedSession:      shellTestRuntime(t),
		createdSession:       session,
		requestedShell:       "bash",
		requestedMultiplexer: "tmux",
	}

	err := runShell(application, nil)

	if !errors.Is(err, attachErr) {
		t.Fatalf(
			"runShell() error = %v, want %v",
			err,
			attachErr,
		)
	}

	if !backend.attached {
		t.Fatal("runShell() did not attempt attachment")
	}

	if !backend.destroyed {
		t.Fatal(
			"runShell() did not clean up newly created session",
		)
	}
}

func TestRunShellDoesNotCreateWhenDiscoveryFailsUnexpectedly(
	t *testing.T,
) {
	discoverErr := errors.New("discovery failed")

	application := &shellTestApplication{
		discoverErr:          discoverErr,
		requestedMultiplexer: "tmux",
	}

	err := runShell(application, nil)

	if !errors.Is(err, discoverErr) {
		t.Fatalf(
			"runShell() error = %v, want %v",
			err,
			discoverErr,
		)
	}
}

func TestRunShellRejectsArguments(t *testing.T) {
	application := &shellTestApplication{}

	err := runShell(application, []string{"unexpected"})

	if err == nil {
		t.Fatal("runShell() returned nil error")
	}
}

func TestRunDetachDetachesExistingSession(t *testing.T) {
	backend := &shellTestBackend{}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/tmp/test.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverSession: session,
	}

	if err := runDetach(application, nil); err != nil {
		t.Fatalf("runDetach() returned error: %v", err)
	}

	if !backend.detached {
		t.Fatal("runDetach() did not detach session")
	}

	if backend.destroyed {
		t.Fatal("runDetach() destroyed session")
	}
}

func TestRunDetachSucceedsWhenSessionDoesNotExist(t *testing.T) {
	application := &shellTestApplication{
		discoverErr: multiplexer.ErrSessionNotFound,
	}

	if err := runDetach(application, nil); err != nil {
		t.Fatalf(
			"runDetach() returned error: %v",
			err,
		)
	}
}

func TestRunDetachReturnsDetachError(t *testing.T) {
	detachErr := errors.New("detach failed")

	backend := &shellTestBackend{
		detachErr: detachErr,
	}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/tmp/test.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverSession: session,
	}

	err := runDetach(application, nil)

	if !errors.Is(err, detachErr) {
		t.Fatalf(
			"runDetach() error = %v, want %v",
			err,
			detachErr,
		)
	}

	if !backend.detached {
		t.Fatal("runDetach() did not attempt detachment")
	}

	if backend.destroyed {
		t.Fatal("runDetach() destroyed session after detach failure")
	}
}

func TestRunDetachDoesNotDetachWhenDiscoveryFailsUnexpectedly(
	t *testing.T,
) {
	discoverErr := errors.New("discovery failed")

	application := &shellTestApplication{
		discoverErr: discoverErr,
	}

	err := runDetach(application, nil)

	if !errors.Is(err, discoverErr) {
		t.Fatalf(
			"runDetach() error = %v, want %v",
			err,
			discoverErr,
		)
	}
}

func TestRunDetachRejectsArguments(t *testing.T) {
	application := &shellTestApplication{}

	err := runDetach(
		application,
		[]string{"unexpected"},
	)

	if err == nil {
		t.Fatal("runDetach() returned nil error")
	}
}

func TestFormatCleanSessionLabel(t *testing.T) {
	tests := []struct {
		name     string
		metadata sessionmeta.Metadata
		want     string
	}{
		{
			name: "direct shell",
			metadata: sessionmeta.Metadata{
				Mode: sessionmeta.ModeNormal,
				Name: "default",
			},
			want: "direct-shell: default",
		},
		{
			name: "tmux",
			metadata: sessionmeta.Metadata{
				Mode:        sessionmeta.ModeMultiplexer,
				Multiplexer: "tmux",
				Name:        "default",
			},
			want: "Multiplexer-tmux: default",
		},
		{
			name: "zellij",
			metadata: sessionmeta.Metadata{
				Mode:        sessionmeta.ModeMultiplexer,
				Multiplexer: "zellij",
				Name:        "default",
			},
			want: "Multiplexer-zellij: default",
		},
		{
			name: "future multiplexer",
			metadata: sessionmeta.Metadata{
				Mode:        sessionmeta.ModeMultiplexer,
				Multiplexer: "herder",
				Name:        "default",
			},
			want: "Multiplexer-herder: default",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := formatCleanSessionLabel(test.metadata)
			if got != test.want {
				t.Fatalf(
					"formatCleanSessionLabel() = %q, want %q",
					got,
					test.want,
				)
			}
		})
	}
}

func TestSelectCleanSessionUsesNumericIndex(t *testing.T) {
	sessions := []*app.CleanSession{
		{
			Metadata: sessionmeta.Metadata{
				Name: "development",
			},
		},
		{
			Metadata: sessionmeta.Metadata{
				Name: "production",
			},
		},
		{
			Metadata: sessionmeta.Metadata{
				Name: "testing",
			},
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}

	defer reader.Close()

	if _, err := writer.WriteString("2\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	session, err := selectCleanSession(
		sessions,
		bufio.NewReader(os.Stdin),
	)
	if err != nil {
		t.Fatalf(
			"selectCleanSession() returned error: %v",
			err,
		)
	}

	if session != sessions[1] {
		t.Fatalf(
			"selected session = %p, want %p",
			session,
			sessions[1],
		)
	}
}

func TestSelectCleanSessionRepromptsAfterInvalidSelection(t *testing.T) {
	sessions := []*app.CleanSession{
		{
			Metadata: sessionmeta.Metadata{
				Name: "development",
			},
		},
		{
			Metadata: sessionmeta.Metadata{
				Name: "production",
			},
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}

	defer reader.Close()

	if _, err := writer.WriteString("production\n2\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	session, err := selectCleanSession(
		sessions,
		bufio.NewReader(os.Stdin),
	)
	if err != nil {
		t.Fatalf(
			"selectCleanSession() returned error: %v",
			err,
		)
	}

	if session != sessions[1] {
		t.Fatalf(
			"selected session = %p, want %p",
			session,
			sessions[1],
		)
	}
}

func TestSelectCleanSessionCanCancel(t *testing.T) {
	sessions := []*app.CleanSession{
		{
			Metadata: sessionmeta.Metadata{
				Name: "development",
			},
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}

	defer reader.Close()

	if _, err := writer.WriteString("q\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	_, err = selectCleanSession(
		sessions,
		bufio.NewReader(os.Stdin),
	)

	if !errors.Is(err, errCleanSelectionCancelled) {
		t.Fatalf(
			"selectCleanSession() error = %v, want %v",
			err,
			errCleanSelectionCancelled,
		)
	}
}

func TestRunCleanSelectsSingleSession(t *testing.T) {
	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			{
				Metadata: sessionmeta.Metadata{
					Mode: sessionmeta.ModeNormal,
					Name: "default",
				},
			},
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}

	defer reader.Close()

	if _, err := writer.WriteString("1\nn\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	output := captureStdout(t, func() {
		if err := runClean(application, nil); err != nil {
			t.Fatalf(
				"runClean() returned error: %v",
				err,
			)
		}
	})

	wantOutput := `Managed uniShell sessions:

1) direct-shell: default
q) quit

Enter session number: Are you sure you want to clean session "default"? [y/N]: `

	if output != wantOutput {
		t.Fatalf(
			"runClean() output = %q, want %q",
			output,
			wantOutput,
		)
	}
}

func TestRunCleanSelectsMultipleSessions(t *testing.T) {
	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			{
				Metadata: sessionmeta.Metadata{
					Mode: sessionmeta.ModeNormal,
					Name: "development",
				},
			},
			{
				Metadata: sessionmeta.Metadata{
					Mode: sessionmeta.ModeNormal,
					Name: "production",
				},
			},
			{
				Metadata: sessionmeta.Metadata{
					Mode: sessionmeta.ModeNormal,
					Name: "testing",
				},
			},
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}

	defer reader.Close()

	if _, err := writer.WriteString("2\nn\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	output := captureStdout(t, func() {
		if err := runClean(application, nil); err != nil {
			t.Fatalf(
				"runClean() returned error: %v",
				err,
			)
		}
	})

	wantOutput := `Managed uniShell sessions:

1) direct-shell: development
2) direct-shell: production
3) direct-shell: testing
a) ALL
q) quit

Enter session number: Are you sure you want to clean session "production"? [y/N]: `

	if output != wantOutput {
		t.Fatalf(
			"runClean() output = %q, want %q",
			output,
			wantOutput,
		)
	}
}

func TestRunCleanCanCancelSessionSelection(t *testing.T) {
	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			{
				Metadata: sessionmeta.Metadata{
					Mode: sessionmeta.ModeNormal,
					Name: "development",
				},
			},
			{
				Metadata: sessionmeta.Metadata{
					Mode: sessionmeta.ModeNormal,
					Name: "production",
				},
			},
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}

	defer reader.Close()

	if _, err := writer.WriteString("q\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	output := captureStdout(t, func() {
		if err := runClean(application, nil); err != nil {
			t.Fatalf(
				"runClean() returned error: %v",
				err,
			)
		}
	})

	wantOutput := `Managed uniShell sessions:

1) direct-shell: development
2) direct-shell: production
a) ALL
q) quit

Enter session number: `

	if output != wantOutput {
		t.Fatalf(
			"runClean() output = %q, want %q",
			output,
			wantOutput,
		)
	}
}

func TestRunCleanRejectsNonexistentTarget(t *testing.T) {
	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			{
				Metadata: sessionmeta.Metadata{
					Name: "development",
				},
			},
		},
	}

	err := runClean(
		application,
		[]string{"--target", "production"},
	)

	if err == nil {
		t.Fatal("runClean() returned nil error")
	}

	want := `managed session "production" not found`

	if err.Error() != want {
		t.Fatalf(
			"runClean() error = %q, want %q",
			err.Error(),
			want,
		)
	}
}

func TestRunCleanTerminatesConfirmedNormalSession(
	t *testing.T,
) {
	target := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:                "development-id",
			PID:               os.Getpid(),
			ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeNormal,
			Name:              "development",
		},
		RuntimeDir: "/tmp/development",
	}

	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			target,
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf(
			"os.Pipe() returned error: %v",
			err,
		)
	}

	defer reader.Close()

	if _, err := writer.WriteString("y\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	if err := runClean(
		application,
		[]string{"--target", "development"},
	); err != nil {
		t.Fatalf(
			"runClean() returned error: %v",
			err,
		)
	}

	if application.terminatedCleanSession != target {
		t.Fatal(
			"runClean() did not terminate the confirmed session",
		)
	}
}

func TestRunCleanRejectsTargetThatDisappearsAfterConfirmation(
	t *testing.T,
) {
	target := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:                "development-id",
			PID:               os.Getpid(),
			ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeNormal,
			Name:              "development",
		},
		RuntimeDir: "/tmp/development",
	}

	application := &shellTestApplication{
		discoverCleanSessionResult: [][]*app.CleanSession{
			{target},
			nil,
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf(
			"os.Pipe() returned error: %v",
			err,
		)
	}

	defer reader.Close()

	if _, err := writer.WriteString("y\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	err = runClean(
		application,
		[]string{"--target", "development"},
	)

	if err == nil {
		t.Fatal(
			"runClean() returned nil error after target disappeared",
		)
	}

	if application.terminatedCleanSession != nil {
		t.Fatal(
			"runClean() terminated a session that disappeared",
		)
	}
}

func TestRunCleanRejectsReplacementWithDifferentIdentity(
	t *testing.T,
) {
	target := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:                "development-id",
			PID:               os.Getpid(),
			ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeNormal,
			Name:              "development",
		},
		RuntimeDir: "/tmp/development",
	}

	replacement := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:                "replacement-id",
			PID:               os.Getpid(),
			ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeNormal,
			Name:              "development",
		},
		RuntimeDir: "/tmp/replacement",
	}

	application := &shellTestApplication{
		discoverCleanSessionResult: [][]*app.CleanSession{
			{target},
			{replacement},
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf(
			"os.Pipe() returned error: %v",
			err,
		)
	}

	defer reader.Close()

	if _, err := writer.WriteString("y\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	err = runClean(
		application,
		[]string{"--target", "development"},
	)

	if err == nil {
		t.Fatal(
			"runClean() returned nil error for replacement session",
		)
	}

	if application.terminatedCleanSession != nil {
		t.Fatal(
			"runClean() terminated the replacement session",
		)
	}
}

func TestRunCleanDoesNotCleanWhenConfirmationIsRejected(t *testing.T) {
	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			{
				Metadata: sessionmeta.Metadata{
					Name: "development",
				},
			},
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}

	defer reader.Close()

	if _, err := writer.WriteString("n\n"); err != nil {
		t.Fatalf("writer.WriteString() returned error: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() returned error: %v", err)
	}

	os.Stdin = reader

	output := captureStdout(t, func() {
		if err := runClean(
			application,
			[]string{"--target", "development"},
		); err != nil {
			t.Fatalf("runClean() returned error: %v", err)
		}
	})

	wantOutput := `Are you sure you want to clean session "development"? [y/N]: `

	if output != wantOutput {
		t.Fatalf(
			"runClean() output = %q, want %q",
			output,
			wantOutput,
		)
	}
}

func TestRunCleanAcceptsCaseInsensitiveConfirmation(t *testing.T) {
	target := &app.CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:                "development-id",
			PID:               os.Getpid(),
			ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeNormal,
			Name:              "development",
		},
		RuntimeDir: "/tmp/development",
	}

	application := &shellTestApplication{
		discoverCleanSessions: []*app.CleanSession{
			target,
		},
	}

	originalStdin := os.Stdin
	defer func() {
		os.Stdin = originalStdin
	}()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf(
			"os.Pipe() returned error: %v",
			err,
		)
	}

	defer reader.Close()

	if _, err := writer.WriteString("Y\n"); err != nil {
		t.Fatalf(
			"writer.WriteString() returned error: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"writer.Close() returned error: %v",
			err,
		)
	}

	os.Stdin = reader

	if err := runClean(
		application,
		[]string{"--target", "development"},
	); err != nil {
		t.Fatalf(
			"runClean() returned error: %v",
			err,
		)
	}

	if application.terminatedCleanSession != target {
		t.Fatal(
			"runClean() did not terminate the confirmed session",
		)
	}
}

func TestRunCleanReportsWhenNoManagedSessionsExist(t *testing.T) {
	application := &shellTestApplication{}

	output := captureStdout(t, func() {
		if err := runClean(application, nil); err != nil {
			t.Fatalf(
				"runClean() returned error: %v",
				err,
			)
		}
	})

	if output != "No managed uniShell sessions found.\n" {
		t.Fatalf(
			"runClean() output = %q, want %q",
			output,
			"No managed uniShell sessions found.\n",
		)
	}
}

func TestRunCleanRejectsArguments(t *testing.T) {
	application := &shellTestApplication{}

	err := runClean(
		application,
		[]string{"unexpected"},
	)

	if err == nil {
		t.Fatal("runClean() returned nil error")
	}
}

func TestParseCleanArgsUsesInstalled(t *testing.T) {
	options, err := parseCleanArgs([]string{"--installed"})
	if err != nil {
		t.Fatalf("parseCleanArgs() returned error: %v", err)
	}
	if !options.Installed {
		t.Fatal("--installed was not recorded")
	}
}

func TestParseCleanArgsRejectsInstalledWithTarget(t *testing.T) {
	_, err := parseCleanArgs([]string{"--installed", "--target", "development"})
	if err == nil {
		t.Fatal("parseCleanArgs() accepted --installed with --target")
	}
}

func TestConfirmCleanPersistentRuntimeRequiresPathConfirmation(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "home", "user", ".local", "unishell")

	confirmed, err := confirmCleanPersistentRuntime(bufio.NewReader(strings.NewReader("yes\n"+root+"\n")), root)
	if err != nil {
		t.Fatalf("confirmCleanPersistentRuntime() returned error: %v", err)
	}
	if !confirmed {
		t.Fatal("confirmCleanPersistentRuntime() rejected matching path confirmation")
	}

	confirmed, err = confirmCleanPersistentRuntime(bufio.NewReader(strings.NewReader("yes\nwrong\n")), root)
	if err != nil {
		t.Fatalf("confirmCleanPersistentRuntime() returned error: %v", err)
	}
	if confirmed {
		t.Fatal("confirmCleanPersistentRuntime() accepted a different path")
	}
}

func TestParseCleanArgsUsesTarget(t *testing.T) {
	options, err := parseCleanArgs([]string{
		"--target",
		"development",
	})
	if err != nil {
		t.Fatalf(
			"parseCleanArgs() returned error: %v",
			err,
		)
	}

	if options.Target != "development" {
		t.Fatalf(
			"target = %q, want %q",
			options.Target,
			"development",
		)
	}
}

func TestParseCleanArgsUsesTargetEqualsSyntax(t *testing.T) {
	options, err := parseCleanArgs([]string{
		"--target=development",
	})
	if err != nil {
		t.Fatalf(
			"parseCleanArgs() returned error: %v",
			err,
		)
	}

	if options.Target != "development" {
		t.Fatalf(
			"target = %q, want %q",
			options.Target,
			"development",
		)
	}
}

func TestParseCleanArgsRejectsMissingTarget(t *testing.T) {
	_, err := parseCleanArgs([]string{
		"--target",
	})
	if err == nil {
		t.Fatal("parseCleanArgs() returned nil error")
	}
}

func TestParseCleanArgsRejectsEmptyTarget(t *testing.T) {
	_, err := parseCleanArgs([]string{
		"--target=",
	})
	if err == nil {
		t.Fatal("parseCleanArgs() returned nil error")
	}
}

func TestParseCleanArgsRejectsExtraArguments(t *testing.T) {
	_, err := parseCleanArgs([]string{
		"unexpected",
	})
	if err == nil {
		t.Fatal("parseCleanArgs() returned nil error")
	}
}

func TestParseCleanArgsRejectsExtraArgumentAfterTarget(t *testing.T) {
	_, err := parseCleanArgs([]string{
		"--target",
		"development",
		"unexpected",
	})
	if err == nil {
		t.Fatal("parseCleanArgs() returned nil error")
	}
}

func TestRunShellAuthenticatesBeforeMultiplexerSelection(
	t *testing.T,
) {
	authErr := errors.New("invalid auth token")

	application := &shellTestApplication{
		authErr: authErr,
	}

	err := runShell(
		application,
		nil,
	)
	if err == nil {
		t.Fatal("runShell() returned nil error")
	}

	if !errors.Is(err, authErr) {
		t.Fatalf(
			"runShell() error = %v, want auth error",
			err,
		)
	}

	if application.createdMultiplexer != "" {
		t.Fatalf(
			"created multiplexer = %q, want empty",
			application.createdMultiplexer,
		)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() returned error: %v", err)
	}

	os.Stdout = writer

	defer func() {
		os.Stdout = original
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() returned error: %v", err)
	}

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll() returned error: %v", err)
	}

	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close() returned error: %v", err)
	}

	return string(output)
}

func TestEnvironmentMap(t *testing.T) {
	got := environmentMap([]string{
		"HOME=/home/test",
		"EMPTY=",
		"INVALID",
		"PATH=/usr/bin",
	})

	want := map[string]string{
		"HOME":  "/home/test",
		"EMPTY": "",
		"PATH":  "/usr/bin",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"environmentMap() = %#v, want %#v",
			got,
			want,
		)
	}
}

func TestInterpolationEnvironmentIncludesRuntimeDirectory(
	t *testing.T,
) {
	runtimeDir := filepath.Join(t.TempDir(), "session")

	sessionEnvironment, systemEnvironment :=
		interpolationEnvironment(
			map[string]string{
				"UNISHELL_SESSION_ID": "test-session",
			},
			runtimeDir,
		)

	if got := sessionEnvironment[shell.SessionRuntimeDirEnvName]; got != runtimeDir {
		t.Fatalf(
			"session runtime directory = %q, want %q",
			got,
			runtimeDir,
		)
	}

	if systemEnvironment == nil {
		t.Fatal("system environment is nil")
	}
}

func TestPrepareShellStartupPropagatesInterpolationEnvironment(
	t *testing.T,
) {
	runtimeDir := t.TempDir()

	sharedDir := filepath.Join(
		runtimeDir,
		"config",
		"shell",
		"shared",
	)

	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sharedDir, "config.toml"),
		[]byte(
			"[environment]\n"+
				"TEST_SESSION_ID = \"${UNISHELL_SESSION_ID}\"\n"+
				"TEST_RUNTIME_DIR = \"${UNISHELL_SESSION_RUNTIME_DIR}\"\n"+
				"TEST_HOME = \"${HOME}\"\n",
		),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	application := &shellTestApplication{
		requestedShell: "bash",
	}

	startup, err := prepareShellStartup(
		application,
		shell.Shell{
			Name: "bash",
			Path: "/bin/bash",
		},
		runtimeDir,
		map[string]string{
			"UNISHELL_SESSION_ID":          "test-session",
			shell.SessionRuntimeDirEnvName: runtimeDir,
		},
		map[string]string{
			"HOME": "/home/test",
		},
	)
	if err != nil {
		t.Fatalf("prepareShellStartup() returned error: %v", err)
	}

	if len(startup.Args) != 3 {
		t.Fatalf(
			"startup.Args length = %d, want 3",
			len(startup.Args),
		)
	}

	startupPath := startup.Args[2]

	data, err := os.ReadFile(startupPath)
	if err != nil {
		t.Fatalf(
			"ReadFile(%q) returned error: %v",
			startupPath,
			err,
		)
	}

	want := `# Generated by uniShell for bash. Do not edit.

export TEST_HOME='/home/test'
export TEST_RUNTIME_DIR='` + runtimeDir + `'
export TEST_SESSION_ID='test-session'
`

	if string(data) != want {
		t.Fatalf(
			"startup file = %q, want %q",
			string(data),
			want,
		)
	}
}

func TestChooseMultiplexerSessionAttachesMatchingTypeWithoutPrompt(t *testing.T) {
	backend := &shellTestBackend{alive: true}
	session := &app.Session{Multiplexer: &multiplexer.ManagedSession{
		Metadata: sessionmeta.Metadata{
			Multiplexer:            "tmux",
			MultiplexerSessionName: "work@abc",
			ID:                     "session-id",
		},
		Backend: backend,
	}}
	var output bytes.Buffer

	selected, err := chooseMultiplexerSession(
		t.Context(), "tmux", []*app.Session{session}, strings.NewReader(""), &output,
	)
	if err != nil {
		t.Fatalf("chooseMultiplexerSession() returned error: %v", err)
	}
	if selected != session {
		t.Fatal("chooseMultiplexerSession() did not return matching session")
	}
	if output.Len() != 0 {
		t.Fatalf("prompt output = %q, want empty", output.String())
	}
}

func TestChooseMultiplexerSessionPromptsForOtherType(t *testing.T) {
	backend := &shellTestBackend{alive: true}
	session := &app.Session{Multiplexer: &multiplexer.ManagedSession{
		Metadata: sessionmeta.Metadata{
			Multiplexer:            "tmux",
			MultiplexerSessionName: "work@abc",
			ID:                     "metadata-id",
		},
		Backend: backend,
		Session: multiplexer.Session{
			Runtime: "/tmp/unishell/runtime/development/runtime-directory-id",
		},
	}}
	var output bytes.Buffer

	selected, err := chooseMultiplexerSession(
		t.Context(), "zellij", []*app.Session{session}, strings.NewReader("1\n"), &output,
	)
	if err != nil {
		t.Fatalf("chooseMultiplexerSession() returned error: %v", err)
	}
	if selected != session {
		t.Fatal("chooseMultiplexerSession() did not return selected session")
	}
	for _, want := range []string{"tmux", "work@abc", "runtime-directory-id"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("prompt output %q does not contain %q", output.String(), want)
		}
	}
}

func TestChooseMultiplexerSessionCanStartNew(t *testing.T) {
	backend := &shellTestBackend{alive: true}
	session := &app.Session{Multiplexer: &multiplexer.ManagedSession{
		Metadata: sessionmeta.Metadata{Multiplexer: "tmux"},
		Backend:  backend,
	}}

	selected, err := chooseMultiplexerSession(
		t.Context(), "zellij", []*app.Session{session}, strings.NewReader("n\n"), io.Discard,
	)
	if err != nil {
		t.Fatalf("chooseMultiplexerSession() returned error: %v", err)
	}
	if selected != nil {
		t.Fatal("chooseMultiplexerSession() selected existing session, want new session")
	}
}

type listTestApplication struct {
	sessions []*app.CleanSession
}

func (a *listTestApplication) ListSessions() ([]*app.CleanSession, error) {
	return a.sessions, nil
}

func TestRunListPrintsSessionIDNameAndType(t *testing.T) {
	application := &listTestApplication{sessions: []*app.CleanSession{
		{Metadata: sessionmeta.Metadata{ID: "direct-id", Name: "local", Mode: sessionmeta.ModeNormal}},
		{Metadata: sessionmeta.Metadata{ID: "mux-id", Name: "work", Mode: sessionmeta.ModeMultiplexer}},
	}}

	output := captureStdout(t, func() {
		if err := runList(application, nil); err != nil {
			t.Fatalf("runList() returned error: %v", err)
		}
	})
	want := "SESSION ID\tSESSION NAME\tSESSION TYPE\n" +
		"direct-id\tlocal\tdirect\n" +
		"mux-id\twork\tmultiplexer\n"
	if output != want {
		t.Fatalf("runList() output = %q, want %q", output, want)
	}
}

func TestRunListRejectsArguments(t *testing.T) {
	if err := runList(&listTestApplication{}, []string{"unexpected"}); err == nil {
		t.Fatal("runList() accepted arguments")
	}
}

func TestApplyPersistentConfigAllowsExplicitBooleanOverride(t *testing.T) {
	root := t.TempDir()
	if _, err := persistence.Create(root); err != nil {
		t.Fatalf("persistence.Create() returned error: %v", err)
	}
	if err := persistence.SaveFirstLaunch(root, persistence.LaunchConfig{
		Shell:      "bash",
		NoSharedRC: true,
	}); err != nil {
		t.Fatalf("SaveFirstLaunch() returned error: %v", err)
	}

	defaults, err := applyPersistentConfig(cliOptions{}, root)
	if err != nil {
		t.Fatalf("applyPersistentConfig() returned error: %v", err)
	}
	if !defaults.NoSharedRC {
		t.Fatal("config no_shared_rc=true was not applied")
	}

	override, err := applyPersistentConfig(cliOptions{NoSharedRCSpecified: true}, root)
	if err != nil {
		t.Fatalf("applyPersistentConfig() with override returned error: %v", err)
	}
	if override.NoSharedRC {
		t.Fatal("explicit --shared-rc did not override config")
	}
}

func TestRollbackRefreshedRuntimeRemovesOnlyNewInstalledBundle(t *testing.T) {
	root := t.TempDir()
	fingerprint := strings.Repeat("a", 24)
	directory := filepath.Join(root, "runtime", "v2", "installed-"+fingerprint)
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(directory, ".unishell-installed"), []byte("bundle="+fingerprint+"\n"), 0600); err != nil {
		t.Fatalf("WriteFile(marker) returned error: %v", err)
	}

	if err := rollbackRefreshedRuntime(root, runtimeRefreshReceipt{Directory: directory, Created: true}); err != nil {
		t.Fatalf("rollbackRefreshedRuntime() returned error: %v", err)
	}
	if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("refreshed bundle still exists after rollback: stat error = %v", err)
	}
}

func TestRollbackRefreshedRuntimePreservesExistingBundle(t *testing.T) {
	root := t.TempDir()
	fingerprint := strings.Repeat("b", 24)
	directory := filepath.Join(root, "runtime", "v2", "installed-"+fingerprint)
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}
	if err := rollbackRefreshedRuntime(root, runtimeRefreshReceipt{Directory: directory, Created: false}); err != nil {
		t.Fatalf("rollbackRefreshedRuntime() returned error: %v", err)
	}
	if _, err := os.Stat(directory); err != nil {
		t.Fatalf("existing bundle was removed: %v", err)
	}
}

func TestRollbackRefreshedRuntimeRejectsPathOutsideRuntime(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "installed-"+strings.Repeat("c", 24))
	if err := os.MkdirAll(outside, 0700); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}
	if err := rollbackRefreshedRuntime(root, runtimeRefreshReceipt{Directory: outside, Created: true}); err == nil {
		t.Fatal("rollbackRefreshedRuntime() accepted a path outside runtime")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside path was removed: %v", err)
	}
}

func TestRollbackRefreshedRuntimeRemovesIncompleteNewBundle(t *testing.T) {
	root := t.TempDir()
	fingerprint := strings.Repeat("d", 24)
	directory := filepath.Join(root, "runtime", "v2", "installed-"+fingerprint)
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatalf("MkdirAll() returned error: %v", err)
	}

	if err := rollbackRefreshedRuntime(root, runtimeRefreshReceipt{Directory: directory, Created: true}); err != nil {
		t.Fatalf("rollbackRefreshedRuntime() returned error: %v", err)
	}
	if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("incomplete refreshed bundle remains: stat error = %v", err)
	}
}

func TestRunCompletionPrintsEmbeddedScripts(t *testing.T) {
	for _, shellName := range []string{"bash", "zsh", "fish"} {
		t.Run(shellName, func(t *testing.T) {
			output := captureStdout(t, func() {
				if err := runCompletion([]string{shellName}); err != nil {
					t.Fatalf("runCompletion() returned error: %v", err)
				}
			})
			for _, expected := range []string{"completion", "multiplexer", "tmux", "zellij", "shared-rc"} {
				if !strings.Contains(output, expected) {
					t.Errorf("%s completions do not contain %q", shellName, expected)
				}
			}
		})
	}
}

func TestRunCompletionRejectsUnsupportedShell(t *testing.T) {
	if err := runCompletion([]string{"powershell"}); err == nil {
		t.Fatal("runCompletion() accepted an unsupported shell")
	}
}
