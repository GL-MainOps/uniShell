package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
	"gitlab.com/mainops/uniShell/internal/shell"
)

func TestNewApplicationPropagatesShellConfigurationOptions(
	t *testing.T,
) {
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

	runtimeDir := t.TempDir()
	sharedDir := filepath.Join(runtimeDir, "config", "shell", "shared")

	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatalf("create shared shell config directory: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sharedDir, "config.toml"),
		[]byte("[environment]\n"),
		0o644,
	); err != nil {
		t.Fatalf("write shared shell configuration: %v", err)
	}

	return &runtime.Session{
		Paths: runtime.Paths{
			Runtime: runtimeDir,
		},
	}
}

type shellTestApplication struct {
	discoverSession      *app.Session
	discoverErr          error
	startSession         *app.Session
	startErr             error
	preparedSession      *runtime.Session
	preparedErr          error
	createdSession       *app.Session
	createdErr           error
	requestedShell       string
	requestedMultiplexer string
	runtimeSession       *runtime.Session
	runtimeSessionErr    error
	authErr              error
	createdMultiplexer   string
	createdStartup       shell.Startup
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

func (b *shellTestBackend) Create(multiplexer.Session) error {
	return nil
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

	if err := runShell(application, nil); err != nil {
		t.Fatalf("runShell() returned error: %v", err)
	}

	if !backend.attached {
		t.Fatal("runShell() did not attach newly created session")
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

func TestRunCleanDestroysExistingSession(t *testing.T) {
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

	if err := runClean(application, nil); err != nil {
		t.Fatalf("runClean() returned error: %v", err)
	}

	if !backend.destroyed {
		t.Fatal("runClean() did not destroy session")
	}
}

func TestRunCleanRejectsNonexistentTarget(t *testing.T) {
	application := &shellTestApplication{
		discoverSession: &app.Session{
			Multiplexer: &multiplexer.ManagedSession{
				Session: multiplexer.Session{
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

func TestRunCleanConfirmsExplicitTarget(t *testing.T) {
	application := &shellTestApplication{
		discoverSession: &app.Session{
			Multiplexer: &multiplexer.ManagedSession{
				Session: multiplexer.Session{
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

	if _, err := writer.WriteString("y\n"); err != nil {
		t.Fatalf("writer.WriteString() returned error: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() returned error: %v", err)
	}

	os.Stdin = reader

	output := captureStdout(t, func() {
		err = runClean(
			application,
			[]string{"--target", "development"},
		)
	})

	if err == nil {
		t.Fatal("runClean() returned nil error")
	}

	wantErr := `clean confirmation accepted for "development", cleanup is not implemented yet`

	if err.Error() != wantErr {
		t.Fatalf(
			"runClean() error = %q, want %q",
			err.Error(),
			wantErr,
		)
	}

	wantOutput := `Are you sure you want to clean session "development"? [y/N]: `

	if output != wantOutput {
		t.Fatalf(
			"runClean() output = %q, want %q",
			output,
			wantOutput,
		)
	}
}

func TestRunCleanDoesNotCleanWhenConfirmationIsRejected(t *testing.T) {
	backend := &shellTestBackend{}

	application := &shellTestApplication{
		discoverSession: &app.Session{
			Multiplexer: &multiplexer.ManagedSession{
				Backend: backend,
				Session: multiplexer.Session{
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

	if backend.destroyed {
		t.Fatal("runClean() destroyed session after rejected confirmation")
	}
}

func TestRunCleanAcceptsCaseInsensitiveConfirmation(t *testing.T) {
	application := &shellTestApplication{
		discoverSession: &app.Session{
			Multiplexer: &multiplexer.ManagedSession{
				Session: multiplexer.Session{
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

	if _, err := writer.WriteString("YeS\n"); err != nil {
		t.Fatalf("writer.WriteString() returned error: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() returned error: %v", err)
	}

	os.Stdin = reader

	err = runClean(
		application,
		[]string{"--target", "development"},
	)

	if err == nil {
		t.Fatal("runClean() returned nil error")
	}

	want := `clean confirmation accepted for "development", cleanup is not implemented yet`

	if err.Error() != want {
		t.Fatalf(
			"runClean() error = %q, want %q",
			err.Error(),
			want,
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
