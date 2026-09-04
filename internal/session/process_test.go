package session

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

func TestProcessStartTicksReturnsCurrentProcessStartTicks(
	t *testing.T,
) {
	got, err := ProcessStartTicks(os.Getpid())
	if err != nil {
		t.Fatalf(
			"ProcessStartTicks() returned error: %v",
			err,
		)
	}

	want := CurrentProcessStartTicks()

	if got != want {
		t.Fatalf(
			"ProcessStartTicks() = %d, want %d",
			got,
			want,
		)
	}
}

func TestProcessGroupIDReturnsCurrentProcessGroupID(
	t *testing.T,
) {
	got, err := ProcessGroupID(os.Getpid())
	if err != nil {
		t.Fatalf(
			"ProcessGroupID() returned error: %v",
			err,
		)
	}

	want := CurrentProcessGroupID()

	if got != want {
		t.Fatalf(
			"ProcessGroupID() = %d, want %d",
			got,
			want,
		)
	}

	if got <= 0 {
		t.Fatalf(
			"ProcessGroupID() = %d, want positive process group ID",
			got,
		)
	}
}

func TestProcessGroupIDRejectsInvalidPID(t *testing.T) {
	if _, err := ProcessGroupID(0); err == nil {
		t.Fatal(
			"ProcessGroupID() returned nil error for invalid PID",
		)
	}
}

func TestProcessGroupIDReturnsErrorForMissingProcess(
	t *testing.T,
) {
	_, err := ProcessGroupID(999999)

	if err == nil {
		t.Fatal(
			"ProcessGroupID() returned nil error for missing process",
		)
	}
}

func TestTerminateProcessKillsMatchingProcess(
	t *testing.T,
) {
	if os.Getenv("UNISHELL_PROCESS_TEST_HELPER") == "1" {
		select {}
	}

	cmd := exec.Command(
		os.Args[0],
		"-test.run=TestTerminateProcessKillsMatchingProcess",
	)

	cmd.Env = append(
		os.Environ(),
		"UNISHELL_PROCESS_TEST_HELPER=1",
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf(
			"start helper process: %v",
			err,
		)
	}

	pid := cmd.Process.Pid

	startTicks, err := ProcessStartTicks(pid)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessStartTicks() returned error: %v",
			err,
		)
	}

	if err := TerminateProcess(pid, startTicks); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"TerminateProcess() returned error: %v",
			err,
		)
	}

	err = cmd.Wait()
	if err == nil {
		t.Fatal("helper process exited without an error")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
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

func TestTerminateProcessRejectsMismatchedIdentity(
	t *testing.T,
) {
	if os.Getenv("UNISHELL_PROCESS_TEST_HELPER") == "1" {
		select {}
	}

	cmd := exec.Command(
		os.Args[0],
		"-test.run=TestTerminateProcessRejectsMismatchedIdentity",
	)

	cmd.Env = append(
		os.Environ(),
		"UNISHELL_PROCESS_TEST_HELPER=1",
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

	startTicks, err := ProcessStartTicks(cmd.Process.Pid)
	if err != nil {
		t.Fatalf(
			"ProcessStartTicks() returned error: %v",
			err,
		)
	}

	err = TerminateProcess(
		cmd.Process.Pid,
		startTicks+1,
	)
	if err == nil {
		t.Fatal(
			"TerminateProcess() returned nil error for mismatched identity",
		)
	}

	if !strings.Contains(
		err.Error(),
		"process identity mismatch",
	) {
		t.Fatalf(
			"TerminateProcess() error = %q, want identity mismatch",
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

func TestTerminateProcessTreatsMissingProcessAsComplete(
	t *testing.T,
) {
	cmd := exec.Command(
		os.Args[0],
		"-test.run=TestTerminateProcessTreatsMissingProcessAsComplete",
	)

	cmd.Env = append(
		os.Environ(),
		"UNISHELL_PROCESS_TEST_HELPER=1",
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf(
			"start helper process: %v",
			err,
		)
	}

	pid := cmd.Process.Pid

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf(
			"kill helper process: %v",
			err,
		)
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError

		if !errors.As(err, &exitErr) {
			t.Fatalf(
				"helper process wait error = %v",
				err,
			)
		}
	}

	err := TerminateProcess(pid, 1)
	if !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf(
			"TerminateProcess() error = %v, want os.ErrProcessDone",
			err,
		)
	}
}

func TestTerminateProcessRejectsInvalidPID(t *testing.T) {
	err := TerminateProcess(0, 1)

	if err == nil {
		t.Fatal(
			"TerminateProcess() returned nil error for invalid PID",
		)
	}
}

func TestTerminateProcessRejectsInvalidStartTime(
	t *testing.T,
) {
	err := TerminateProcess(os.Getpid(), 0)

	if err == nil {
		t.Fatal(
			"TerminateProcess() returned nil error for invalid start time",
		)
	}
}

func TestTerminateProcessDoesNotKillAfterProcessHasExited(
	t *testing.T,
) {
	cmd := exec.Command(
		os.Args[0],
		"-test.run=TestTerminateProcessDoesNotKillAfterProcessHasExited",
	)

	cmd.Env = append(
		os.Environ(),
		"UNISHELL_PROCESS_TEST_HELPER=1",
	)

	if err := cmd.Start(); err != nil {
		t.Fatalf(
			"start helper process: %v",
			err,
		)
	}

	startTicks, err := ProcessStartTicks(cmd.Process.Pid)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessStartTicks() returned error: %v",
			err,
		)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf(
			"kill helper process: %v",
			err,
		)
	}

	if err := cmd.Wait(); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf(
				"helper process wait error = %v",
				err,
			)
		}
	}

	err = TerminateProcess(cmd.Process.Pid, startTicks)
	if err == nil {
		t.Fatal(
			"TerminateProcess() returned nil error for exited process",
		)
	}

	if !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf(
			"TerminateProcess() error = %v, want os.ErrProcessDone",
			err,
		)
	}
}
