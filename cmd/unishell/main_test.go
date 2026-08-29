package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/runtime"
	"gitlab.com/mainops/uniShell/internal/shell"
)

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

type shellTestApplication struct {
	discoverSession       *app.Session
	discoverErr           error
	startSession          *app.Session
	startErr              error
	preparedSession       *runtime.Session
	preparedErr           error
	createdSession        *app.Session
	createdErr            error
	requestedShell        string
	requestedShellProfile string
	requestedNoSharedRC   bool
	requestedMultiplexer  string
	runtimeSession        *runtime.Session
	runtimeSessionErr     error
	authErr               error
	createdMultiplexer    string
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

func (a *shellTestApplication) RequestedShell() string {
	return a.requestedShell
}

func (a *shellTestApplication) RequestedShellProfile() string {
	return a.requestedShellProfile
}

func (a *shellTestApplication) RequestedNoSharedRC() bool {
	return a.requestedNoSharedRC
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
	return a.createdSession, a.createdErr
}

type shellTestBackend struct {
	attached  bool
	detached  bool
	destroyed bool
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
	return true
}

func (b *shellTestBackend) Destroy(multiplexer.Session) error {
	b.destroyed = true
	return nil
}

func TestPrepareShellStartupWithoutProfileReturnsEmptyStartup(
	t *testing.T,
) {
	application := &shellTestApplication{}

	startup, err := prepareShellStartup(
		application,
		shell.Shell{
			Name: "bash",
		},
		t.TempDir(),
	)
	if err != nil {
		t.Fatalf(
			"prepareShellStartup() returned error: %v",
			err,
		)
	}

	if len(startup.Args) != 0 {
		t.Fatalf(
			"startup args = %#v, want empty",
			startup.Args,
		)
	}

	if len(startup.Env) != 0 {
		t.Fatalf(
			"startup env = %#v, want empty",
			startup.Env,
		)
	}
}

func TestPrepareShellStartupLoadsProfileAndSharedConfig(
	t *testing.T,
) {
	runtimeDir := t.TempDir()

	profileRoot := filepath.Join(
		runtimeDir,
		"config",
		"shell",
	)

	if err := os.MkdirAll(
		filepath.Join(profileRoot, "bash"),
		0700,
	); err != nil {
		t.Fatalf(
			"create profile directory: %v",
			err,
		)
	}

	if err := os.MkdirAll(
		filepath.Join(profileRoot, "shared"),
		0700,
	); err != nil {
		t.Fatalf(
			"create shared directory: %v",
			err,
		)
	}

	if err := os.WriteFile(
		filepath.Join(
			profileRoot,
			"shared",
			"config.toml",
		),
		[]byte(`
[environment]
EDITOR = "vim"
`),
		0600,
	); err != nil {
		t.Fatalf(
			"write shared config: %v",
			err,
		)
	}

	if err := os.WriteFile(
		filepath.Join(
			profileRoot,
			"bash",
			"server.bash",
		),
		[]byte("echo profile\n"),
		0600,
	); err != nil {
		t.Fatalf(
			"write profile: %v",
			err,
		)
	}

	application := &shellTestApplication{
		requestedShellProfile: "server",
	}

	startup, err := prepareShellStartup(
		application,
		shell.Shell{
			Name: "bash",
		},
		runtimeDir,
	)
	if err != nil {
		t.Fatalf(
			"prepareShellStartup() returned error: %v",
			err,
		)
	}

	wantPath := filepath.Join(
		runtimeDir,
		"config",
		"shell-generated",
		"server.bash",
	)

	wantArgs := []string{
		"--noprofile",
		"--rcfile",
		wantPath,
	}

	if !reflect.DeepEqual(startup.Args, wantArgs) {
		t.Fatalf(
			"startup args = %#v, want %#v",
			startup.Args,
			wantArgs,
		)
	}

	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf(
			"read generated startup: %v",
			err,
		)
	}

	want := `# Generated by uniShell for bash. Do not edit.

export EDITOR='vim'
echo profile
`

	if string(data) != want {
		t.Fatalf(
			"generated startup = %q, want %q",
			string(data),
			want,
		)
	}
}

func TestPrepareShellStartupExcludesSharedConfig(
	t *testing.T,
) {
	runtimeDir := t.TempDir()

	profileRoot := filepath.Join(
		runtimeDir,
		"config",
		"shell",
	)

	if err := os.MkdirAll(
		filepath.Join(profileRoot, "bash"),
		0700,
	); err != nil {
		t.Fatalf(
			"create profile directory: %v",
			err,
		)
	}

	if err := os.WriteFile(
		filepath.Join(
			profileRoot,
			"bash",
			"server.bash",
		),
		[]byte("echo profile\n"),
		0600,
	); err != nil {
		t.Fatalf(
			"write profile: %v",
			err,
		)
	}

	application := &shellTestApplication{
		requestedShellProfile: "server",
		requestedNoSharedRC:   true,
	}

	_, err := prepareShellStartup(
		application,
		shell.Shell{
			Name: "bash",
		},
		runtimeDir,
	)
	if err != nil {
		t.Fatalf(
			"prepareShellStartup() returned error: %v",
			err,
		)
	}

	path := filepath.Join(
		runtimeDir,
		"config",
		"shell-generated",
		"server.bash",
	)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf(
			"read generated startup: %v",
			err,
		)
	}

	want := `echo profile
`

	if string(data) != want {
		t.Fatalf(
			"generated startup = %q, want %q",
			string(data),
			want,
		)
	}
}

func TestRunShellAttachesExistingSession(t *testing.T) {
	backend := &shellTestBackend{}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Metadata: multiplexer.Metadata{
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
		discoverErr:          multiplexer.ErrSessionNotFound,
		preparedSession:      &runtime.Session{},
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
		preparedSession:      &runtime.Session{},
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

func TestRunCleanSucceedsWhenSessionDoesNotExist(t *testing.T) {
	application := &shellTestApplication{
		discoverErr: multiplexer.ErrSessionNotFound,
	}

	if err := runClean(application, nil); err != nil {
		t.Fatalf(
			"runClean() returned error: %v",
			err,
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
