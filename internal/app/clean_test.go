package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"gitlab.com/mainops/uniShell/internal/runtime"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

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

	if err != nil {
		t.Fatalf(
			"TerminateNormalSession() returned error: %v",
			err,
		)
	}
}
