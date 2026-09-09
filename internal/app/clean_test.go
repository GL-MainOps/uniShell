package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"gitlab.com/mainops/uniShell/internal/multiplexer"
	"gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

type cleanTestBackend struct {
	alive     bool
	destroyed bool
}

func (b *cleanTestBackend) Name() string {
	return "test"
}

func (b *cleanTestBackend) Capabilities() map[multiplexer.Capability]bool {
	return map[multiplexer.Capability]bool{
		multiplexer.CapabilitySessions: true,
		multiplexer.CapabilityAttach:   true,
		multiplexer.CapabilityDetach:   true,
		multiplexer.CapabilityDestroy:  true,
	}
}

func (b *cleanTestBackend) Available() bool {
	return true
}

func (b *cleanTestBackend) Create(multiplexer.Session) error {
	b.alive = true
	return nil
}

func (b *cleanTestBackend) Attach(multiplexer.Session) error {
	return nil
}

func (b *cleanTestBackend) Detach(multiplexer.Session) error {
	return nil
}

func (b *cleanTestBackend) IsAlive(multiplexer.Session) bool {
	return b.alive
}

func (b *cleanTestBackend) Destroy(multiplexer.Session) error {
	b.destroyed = true
	b.alive = false
	return nil
}

func writeCleanTestMetadata(
	t *testing.T,
	runtimeDir string,
	mode sessionmeta.Mode,
	name string,
) {
	t.Helper()

	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		t.Fatalf("create runtime directory: %v", err)
	}

	metadata := sessionmeta.Metadata{
		ID:                name + "-id",
		PID:               os.Getpid(),
		ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
		CreatedAt:         time.Now().UTC(),
		Version:           "development",
		Mode:              mode,
		Name:              name,
	}

	if mode == sessionmeta.ModeMultiplexer {
		metadata.Multiplexer = "test"
		metadata.NativeName = "native-" + name
		metadata.Endpoint = filepath.Join(
			runtimeDir,
			"multiplexer",
			"test.sock",
		)
	}

	if err := sessionmeta.WriteMetadata(
		runtimeDir,
		metadata,
	); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
}

func TestCleanupMultiplexerSessionTerminatesOwnedGroup(
	t *testing.T,
) {
	root := t.TempDir()
	runtimeDir := filepath.Join(
		root,
		"multiplexer-session",
	)

	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		t.Fatalf(
			"create runtime directory: %v",
			err,
		)
	}

	cmd := exec.Command(
		os.Args[0],
		"-test.run=^TestCleanupMultiplexerSessionHelper$",
	)

	cmd.Env = append(
		os.Environ(),
		"UNISHELL_CLEAN_MULTIPLEXER_HELPER=1",
	)

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf(
			"start multiplexer helper: %v",
			err,
		)
	}

	startTicks, err := sessionmeta.ProcessStartTicks(
		cmd.Process.Pid,
	)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessStartTicks() returned error: %v",
			err,
		)
	}

	processGroupID, err := sessionmeta.ProcessGroupID(
		cmd.Process.Pid,
	)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessGroupID() returned error: %v",
			err,
		)
	}

	backend := &cleanTestBackend{
		alive: true,
	}

	if err := sessionmeta.WriteMetadata(
		runtimeDir,
		sessionmeta.Metadata{
			ID:                "multiplexer-test-id",
			PID:               cmd.Process.Pid,
			ProcessStartTicks: startTicks,
			ProcessGroupID:    processGroupID,
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeMultiplexer,
			Name:              "development",
			Multiplexer:       "test",
			NativeName:        "native-development",
			Endpoint: filepath.Join(
				runtimeDir,
				"multiplexer",
				"test.sock",
			),
		},
	); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"WriteMetadata() returned error: %v",
			err,
		)
	}

	application := &App{
		Paths: runtime.Paths{
			Runtime: root,
		},
		Multiplexer: multiplexer.NewManager(
			multiplexer.NewRegistry(backend),
		),
	}

	cleanSession := &CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:                "multiplexer-test-id",
			PID:               cmd.Process.Pid,
			ProcessStartTicks: startTicks,
			ProcessGroupID:    processGroupID,
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeMultiplexer,
			Name:              "development",
			Multiplexer:       "test",
		},
		RuntimeDir: runtimeDir,
	}

	if err := application.CleanupMultiplexerSession(
		cleanSession,
	); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"CleanupMultiplexerSession() returned error: %v",
			err,
		)
	}

	if !backend.destroyed {
		t.Fatal(
			"CleanupMultiplexerSession() did not destroy backend session",
		)
	}

	if _, err := os.Stat(runtimeDir); !os.IsNotExist(err) {
		t.Fatalf(
			"runtime directory still exists, stat error = %v",
			err,
		)
	}

	if err := cmd.Wait(); err == nil {
		t.Fatal(
			"multiplexer helper exited successfully after cleanup",
		)
	}
}

func TestCleanupMultiplexerSessionHelper(t *testing.T) {
	if os.Getenv("UNISHELL_CLEAN_MULTIPLEXER_HELPER") != "1" {
		return
	}

	for {
		time.Sleep(time.Second)
	}
}

func TestCleanupMultiplexerSessionRejectsNormalSession(
	t *testing.T,
) {
	err := (&App{}).CleanupMultiplexerSession(
		&CleanSession{
			Metadata: sessionmeta.Metadata{
				Name: "development",
				Mode: sessionmeta.ModeNormal,
			},
		},
	)

	if err == nil {
		t.Fatal(
			"CleanupMultiplexerSession() returned nil for normal session",
		)
	}

	if !strings.Contains(
		err.Error(),
		"not a multiplexer session",
	) {
		t.Fatalf(
			"CleanupMultiplexerSession() error = %q, want multiplexer-session error",
			err,
		)
	}
}

func TestCleanupMultiplexerSessionRejectsMissingMultiplexer(
	t *testing.T,
) {
	err := (&App{}).CleanupMultiplexerSession(
		&CleanSession{
			Metadata: sessionmeta.Metadata{
				Name: "development",
				Mode: sessionmeta.ModeMultiplexer,
			},
		},
	)

	if err == nil {
		t.Fatal(
			"CleanupMultiplexerSession() returned nil for missing multiplexer",
		)
	}

	if !strings.Contains(
		err.Error(),
		"multiplexer manager is unavailable",
	) {
		t.Fatalf(
			"CleanupMultiplexerSession() error = %q, want unavailable-manager error",
			err,
		)
	}
}

func TestDiscoverCleanSessionsIncludesNormalSession(t *testing.T) {
	root := t.TempDir()

	runtimeDir := filepath.Join(
		root,
		"normal-session",
	)

	writeCleanTestMetadata(
		t,
		runtimeDir,
		sessionmeta.ModeNormal,
		"development",
	)

	application := &App{
		Paths: runtime.Paths{
			Runtime: root,
		},
	}

	sessions, err := application.DiscoverCleanSessions()
	if err != nil {
		t.Fatalf(
			"DiscoverCleanSessions() returned error: %v",
			err,
		)
	}

	if len(sessions) != 1 {
		t.Fatalf(
			"session count = %d, want %d",
			len(sessions),
			1,
		)
	}

	if sessions[0].Metadata.Mode != sessionmeta.ModeNormal {
		t.Fatalf(
			"session mode = %q, want %q",
			sessions[0].Metadata.Mode,
			sessionmeta.ModeNormal,
		)
	}
}

func TestDiscoverCleanSessionsReturnsEmptyWhenRuntimeMissing(
	t *testing.T,
) {
	application := &App{
		Paths: runtime.Paths{
			Runtime: filepath.Join(
				t.TempDir(),
				"missing",
			),
		},
	}

	sessions, err := application.DiscoverCleanSessions()
	if err != nil {
		t.Fatalf(
			"DiscoverCleanSessions() returned error: %v",
			err,
		)
	}

	if len(sessions) != 0 {
		t.Fatalf(
			"session count = %d, want %d",
			len(sessions),
			0,
		)
	}
}

func TestTerminateNormalSessionKillsMatchingProcess(
	t *testing.T,
) {
	if os.Getenv("UNISHELL_CLEAN_PROCESS_TEST_HELPER") == "1" {
		select {}
	}

	cmd := exec.Command(
		os.Args[0],
		"-test.run=TestTerminateNormalSessionKillsMatchingProcess",
	)

	cmd.Env = append(
		os.Environ(),
		"UNISHELL_CLEAN_PROCESS_TEST_HELPER=1",
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf(
			"start helper process: %v",
			err,
		)
	}

	startTicks, err := sessionmeta.ProcessStartTicks(
		cmd.Process.Pid,
	)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessStartTicks() returned error: %v",
			err,
		)
	}

	runtimeDir := filepath.Join(
		t.TempDir(),
		"normal-session",
	)
	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"create runtime directory: %v",
			err,
		)
	}

	cleanSession := &CleanSession{
		Metadata: sessionmeta.Metadata{
			ID:                "test-session",
			PID:               cmd.Process.Pid,
			ProcessStartTicks: startTicks,
			CreatedAt:         time.Now().UTC(),
			Version:           "development",
			Mode:              sessionmeta.ModeNormal,
			Name:              "development",
		},
		RuntimeDir: runtimeDir,
	}

	application := &App{}

	if err := application.TerminateNormalSession(
		cleanSession,
	); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"TerminateNormalSession() returned error: %v",
			err,
		)
	}

	err = cmd.Wait()
	if err == nil {
		t.Fatal("helper process exited without an error")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf(
			"helper process error = %T %v, want *exec.ExitError",
			err,
			err,
		)
	}

	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok {
		t.Fatalf(
			"helper process status = %T, want syscall.WaitStatus",
			exitErr.Sys(),
		)
	}

	if !status.Signaled() {
		t.Fatalf(
			"helper process was not signaled: %v",
			status,
		)
	}

	if status.Signal() != syscall.SIGKILL {
		t.Fatalf(
			"helper process signal = %v, want %v",
			status.Signal(),
			syscall.SIGKILL,
		)
	}

	if _, err := os.Stat(runtimeDir); !os.IsNotExist(err) {
		t.Fatalf(
			"runtime directory still exists, stat error = %v",
			err,
		)
	}
}

func TestTerminateNormalSessionRejectsMultiplexerSession(
	t *testing.T,
) {
	application := &App{}

	err := application.TerminateNormalSession(
		&CleanSession{
			Metadata: sessionmeta.Metadata{
				ID:                "test-session",
				PID:               os.Getpid(),
				ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
				CreatedAt:         time.Now().UTC(),
				Version:           "development",
				Mode:              sessionmeta.ModeMultiplexer,
				Name:              "development",
			},
		},
	)

	if err == nil {
		t.Fatal(
			"TerminateNormalSession() returned nil error for multiplexer session",
		)
	}

	if !strings.Contains(
		err.Error(),
		"not a normal session",
	) {
		t.Fatalf(
			"TerminateNormalSession() error = %q, want normal-session error",
			err.Error(),
		)
	}
}

func TestTerminateNormalSessionRejectsEmptyRuntimePath(
	t *testing.T,
) {
	err := (&App{}).TerminateNormalSession(
		&CleanSession{
			Metadata: sessionmeta.Metadata{
				ID:                "test-session",
				PID:               999999,
				ProcessStartTicks: 1,
				CreatedAt:         time.Now().UTC(),
				Version:           "development",
				Mode:              sessionmeta.ModeNormal,
				Name:              "development",
			},
		},
	)

	if err == nil {
		t.Fatal(
			"TerminateNormalSession() returned nil error for empty runtime path",
		)
	}

	if !strings.Contains(
		err.Error(),
		"runtime path is empty",
	) {
		t.Fatalf(
			"TerminateNormalSession() error = %q, want runtime-path error",
			err.Error(),
		)
	}
}

func TestTerminateNormalSessionRejectsMismatchedIdentity(
	t *testing.T,
) {
	if os.Getenv("UNISHELL_CLEAN_PROCESS_TEST_HELPER") == "1" {
		select {}
	}

	cmd := exec.Command(
		os.Args[0],
		"-test.run=TestTerminateNormalSessionRejectsMismatchedIdentity",
	)

	cmd.Env = append(
		os.Environ(),
		"UNISHELL_CLEAN_PROCESS_TEST_HELPER=1",
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf(
			"start helper process: %v",
			err,
		)
	}

	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	startTicks, err := sessionmeta.ProcessStartTicks(
		cmd.Process.Pid,
	)
	if err != nil {
		t.Fatalf(
			"ProcessStartTicks() returned error: %v",
			err,
		)
	}

	err = (&App{}).TerminateNormalSession(
		&CleanSession{
			Metadata: sessionmeta.Metadata{
				ID:                "test-session",
				PID:               cmd.Process.Pid,
				ProcessStartTicks: startTicks + 1,
				CreatedAt:         time.Now().UTC(),
				Version:           "development",
				Mode:              sessionmeta.ModeNormal,
				Name:              "development",
			},
		},
	)

	if err == nil {
		t.Fatal(
			"TerminateNormalSession() returned nil error for mismatched identity",
		)
	}

	if !strings.Contains(
		err.Error(),
		"process identity mismatch",
	) {
		t.Fatalf(
			"TerminateNormalSession() error = %q, want identity mismatch",
			err.Error(),
		)
	}

	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatalf(
			"helper process was terminated after identity mismatch: %v",
			err,
		)
	}
}

func TestTerminateNormalSessionTreatsMissingProcessAsComplete(
	t *testing.T,
) {
	runtimeDir := filepath.Join(
		t.TempDir(),
		"normal-session",
	)
	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		t.Fatalf(
			"create runtime directory: %v",
			err,
		)
	}

	err := (&App{}).TerminateNormalSession(
		&CleanSession{
			Metadata: sessionmeta.Metadata{
				ID:                "test-session",
				PID:               999999,
				ProcessStartTicks: 1,
				CreatedAt:         time.Now().UTC(),
				Version:           "development",
				Mode:              sessionmeta.ModeNormal,
				Name:              "development",
			},
			RuntimeDir: runtimeDir,
		},
	)

	if err != nil {
		t.Fatalf(
			"TerminateNormalSession() returned error: %v",
			err,
		)
	}

	if _, err := os.Stat(runtimeDir); !os.IsNotExist(err) {
		t.Fatalf(
			"runtime directory still exists, stat error = %v",
			err,
		)
	}
}
