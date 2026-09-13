package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"gitlab.com/mainops/uniShell/internal/app"
	"gitlab.com/mainops/uniShell/internal/bundle"
	"gitlab.com/mainops/uniShell/internal/multiplexer"

	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

func TestRunShellCreatesThenReattachesExistingMultiplexerSession(
	t *testing.T,
) {
	root := filepath.Join(t.TempDir(), "unishell")

	t.Setenv("UNISHELL_AUTH_TOKEN", "test-fixture-token")

	helper := startLifecycleProcessGroupHelper(t)
	defer func() {
		_ = helper.Process.Kill()
		_ = helper.Wait()
	}()

	backend := &lifecycleTestBackend{
		processIdentity: lifecycleProcessIdentity(t, helper),
	}

	manager := multiplexer.NewManager(
		multiplexer.NewRegistry(backend),
	)

	application, err := app.New(app.Options{
		Version:         "1.0.0",
		Commit:          "test",
		Root:            root,
		Bundle:          lifecycleTestBundleSource(t),
		Multiplexer:     manager,
		MultiplexerName: "tmux",
		SessionName:     "default",
	})
	if err != nil {
		t.Fatalf("app.New() returned error: %v", err)
	}

	// First invocation must create and attach a new session.
	if err := runShell(application, nil); err != nil {
		t.Fatalf(
			"first runShell() returned error: %v",
			err,
		)
	}

	if backend.createCount != 1 {
		t.Fatalf(
			"backend Create() count = %d, want 1",
			backend.createCount,
		)
	}

	if backend.attachCount != 1 {
		t.Fatalf(
			"backend Attach() count after first invocation = %d, want 1",
			backend.attachCount,
		)
	}

	if backend.destroyCount != 0 {
		t.Fatalf(
			"backend Destroy() count after first invocation = %d, want 0",
			backend.destroyCount,
		)
	}

	// The runtime must remain available after a successful shell start.
	entries, err := os.ReadDir(application.Paths.Runtime)
	if err != nil {
		t.Fatalf(
			"read version runtime after first invocation: %v",
			err,
		)
	}

	if len(entries) != 1 {
		t.Fatalf(
			"version runtime entry count = %d, want 1",
			len(entries),
		)
	}

	// Second invocation must discover the existing session instead
	// of creating another one.
	if err := runShell(application, nil); err != nil {
		t.Fatalf(
			"second runShell() returned error: %v",
			err,
		)
	}

	if backend.createCount != 1 {
		t.Fatalf(
			"backend Create() count after second invocation = %d, want 1",
			backend.createCount,
		)
	}

	if backend.attachCount != 2 {
		t.Fatalf(
			"backend Attach() count after second invocation = %d, want 2",
			backend.attachCount,
		)
	}

	if backend.destroyCount != 0 {
		t.Fatalf(
			"backend Destroy() count after second invocation = %d, want 0",
			backend.destroyCount,
		)
	}

	// Explicit cleanup must destroy the multiplexer and remove
	// the runtime session.
	session, err := application.DiscoverMultiplexerSession()
	if err != nil {
		t.Fatalf(
			"DiscoverMultiplexerSession() returned error: %v",
			err,
		)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf(
			"session.Cleanup() returned error: %v",
			err,
		)
	}

	if backend.destroyCount != 1 {
		t.Fatalf(
			"backend Destroy() count after cleanup = %d, want 1",
			backend.destroyCount,
		)
	}

	entries, err = os.ReadDir(application.Paths.Runtime)
	if err == nil && len(entries) != 0 {
		t.Fatalf(
			"version runtime still contains %d entries after cleanup",
			len(entries),
		)
	}

	if err != nil && !os.IsNotExist(err) {
		t.Fatalf(
			"read version runtime after cleanup: %v",
			err,
		)
	}
}

func startLifecycleProcessGroupHelper(t *testing.T) *exec.Cmd {
	t.Helper()

	cmd := exec.Command(
		os.Args[0],
		"-test.run=^TestLifecycleProcessGroupHelper$",
	)
	cmd.Env = append(
		os.Environ(),
		"UNISHELL_LIFECYCLE_PROCESS_GROUP_HELPER=1",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf(
			"failed to start lifecycle process-group helper: %v",
			err,
		)
	}

	return cmd
}

func TestLifecycleProcessGroupHelper(t *testing.T) {
	if os.Getenv("UNISHELL_LIFECYCLE_PROCESS_GROUP_HELPER") != "1" {
		return
	}

	for {
		time.Sleep(time.Second)
	}
}

func lifecycleProcessIdentity(
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

type lifecycleTestBackend struct {
	createCount  int
	attachCount  int
	destroyCount int

	alive           bool
	processIdentity sessionmeta.ProcessIdentity
}

func (b *lifecycleTestBackend) Name() string {
	return "tmux"
}

func (b *lifecycleTestBackend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *lifecycleTestBackend) Available() bool {
	return true
}

func (b *lifecycleTestBackend) AvailableForSession(
	multiplexer.Session,
) bool {
	return true
}

func (b *lifecycleTestBackend) Create(
	session multiplexer.Session,
) error {
	b.createCount++
	b.alive = true

	return nil
}

func (b *lifecycleTestBackend) ProcessIdentity(
	multiplexer.Session,
) (sessionmeta.ProcessIdentity, error) {
	return b.processIdentity, nil
}

func (b *lifecycleTestBackend) Attach(
	session multiplexer.Session,
) error {
	b.attachCount++

	return nil
}

func (b *lifecycleTestBackend) Detach(
	session multiplexer.Session,
) error {
	return nil
}

func (b *lifecycleTestBackend) IsAlive(
	session multiplexer.Session,
) bool {
	return b.alive
}

func (b *lifecycleTestBackend) Destroy(
	session multiplexer.Session,
) error {
	b.destroyCount++
	b.alive = false

	return nil
}

func lifecycleTestBundleSource(t *testing.T) app.BundleSource {
	t.Helper()

	sourceDir := t.TempDir()

	if err := os.MkdirAll(
		filepath.Join(sourceDir, "config", "shell", "shared"),
		0o755,
	); err != nil {
		t.Fatalf("create lifecycle test shell configuration directory: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sourceDir, "config", "shell", "shared", "config.toml"),
		[]byte("[environment]\n"),
		0o644,
	); err != nil {
		t.Fatalf("write lifecycle test shell configuration: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sourceDir, "test-tool"),
		[]byte("test runtime payload\n"),
		0o755,
	); err != nil {
		t.Fatalf("write lifecycle test runtime payload: %v", err)
	}

	data, err := bundle.Create(sourceDir, "test-fixture-token")
	if err != nil {
		t.Fatalf("create lifecycle test bundle: %v", err)
	}

	return func() ([]byte, error) {
		return data, nil
	}
}
