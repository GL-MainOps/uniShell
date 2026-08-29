package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/shell"
)

type shellHandoffBackend struct {
	attached  bool
	created   bool
	destroyed bool
}

func (b *shellHandoffBackend) Name() string {
	return "test"
}

func (b *shellHandoffBackend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *shellHandoffBackend) Available() bool {
	return true
}

func (b *shellHandoffBackend) Create(multiplexer.Session) error {
	b.created = true
	return nil
}

func (b *shellHandoffBackend) Attach(multiplexer.Session) error {
	b.attached = true
	return nil
}

func (b *shellHandoffBackend) Detach(multiplexer.Session) error {
	return nil
}

func (b *shellHandoffBackend) IsAlive(multiplexer.Session) bool {
	return true
}

func (b *shellHandoffBackend) Destroy(multiplexer.Session) error {
	b.destroyed = true
	return nil
}

func TestRunShellUsesMultiplexerAsInteractiveEntryPoint(
	t *testing.T,
) {
	backend := &shellHandoffBackend{}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/runtime/multiplexer/tmux.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverSession:      session,
		requestedMultiplexer: "tmux",
	}

	if err := runShell(application, nil); err != nil {
		t.Fatalf(
			"runShell() returned error: %v",
			err,
		)
	}

	if !backend.attached {
		t.Fatal(
			"runShell() did not enter the existing multiplexer session",
		)
	}

	if backend.created {
		t.Fatal(
			"runShell() created a second multiplexer session",
		)
	}

	if backend.destroyed {
		t.Fatal(
			"runShell() destroyed the existing multiplexer session",
		)
	}
}

func TestRunShellDoesNotFallbackToStandaloneShell(
	t *testing.T,
) {
	backend := &shellHandoffBackend{}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/runtime/multiplexer/tmux.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverSession: session,
	}

	err := runShell(application, []string{"unexpected"})
	if err == nil {
		t.Fatal("runShell() accepted an unexpected shell argument")
	}

	if backend.attached {
		t.Fatal(
			"runShell() attached a session after rejecting the command",
		)
	}
}

func TestRunShellPropagatesMultiplexerAttachFailure(
	t *testing.T,
) {
	attachErr := errors.New("multiplexer attach failed")

	backend := &shellHandoffBackendWithError{
		attachErr: attachErr,
	}

	session := &app.Session{
		Multiplexer: &multiplexer.ManagedSession{
			Backend: backend,
			Session: multiplexer.Session{
				Name:     "default",
				Endpoint: "/runtime/multiplexer/tmux.sock",
			},
		},
	}

	application := &shellTestApplication{
		discoverSession:      session,
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
}

func TestDirectShellProfileStartupUsesSelectedProfile(
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
		[]byte("echo server\n"),
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

	if len(startup.Args) != len(wantArgs) {
		t.Fatalf(
			"startup args = %#v, want %#v",
			startup.Args,
			wantArgs,
		)
	}

	for i := range wantArgs {
		if startup.Args[i] != wantArgs[i] {
			t.Fatalf(
				"startup args = %#v, want %#v",
				startup.Args,
				wantArgs,
			)
		}
	}

	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf(
			"read generated startup: %v",
			err,
		)
	}

	wantContent := `# Generated by uniShell for bash. Do not edit.

export EDITOR='vim'
echo server
`

	if string(data) != wantContent {
		t.Fatalf(
			"startup content = %q, want %q",
			string(data),
			wantContent,
		)
	}
}

func TestDirectShellProfileStartupHonorsNoSharedRC(
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
		[]byte("echo server\n"),
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

	want := "echo server\n"

	if string(data) != want {
		t.Fatalf(
			"startup content = %q, want %q",
			string(data),
			want,
		)
	}
}

func TestDirectShellProfileStartupRequiresProfileFile(
	t *testing.T,
) {
	runtimeDir := t.TempDir()

	application := &shellTestApplication{
		requestedShellProfile: "missing",
	}

	_, err := prepareShellStartup(
		application,
		shell.Shell{
			Name: "bash",
		},
		runtimeDir,
	)
	if err == nil {
		t.Fatal(
			"prepareShellStartup() returned nil error for missing profile",
		)
	}

	want := `load shell profile "missing": shell profile "missing" not found for shell "bash"`

	if err.Error() != want {
		t.Fatalf(
			"error = %q, want %q",
			err.Error(),
			want,
		)
	}
}

type shellHandoffBackendWithError struct {
	attachErr error
}

func (b *shellHandoffBackendWithError) Name() string {
	return "test"
}

func (b *shellHandoffBackendWithError) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *shellHandoffBackendWithError) Available() bool {
	return true
}

func (b *shellHandoffBackendWithError) Create(multiplexer.Session) error {
	return nil
}

func (b *shellHandoffBackendWithError) Attach(multiplexer.Session) error {
	return b.attachErr
}

func (b *shellHandoffBackendWithError) Detach(multiplexer.Session) error {
	return nil
}

func (b *shellHandoffBackendWithError) IsAlive(multiplexer.Session) bool {
	return true
}

func (b *shellHandoffBackendWithError) Destroy(multiplexer.Session) error {
	return nil
}
