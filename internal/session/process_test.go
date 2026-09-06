package session

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func startProcessGroupTestHelper(t *testing.T) *exec.Cmd {
	t.Helper()

	if os.Getenv("UNISHELL_PROCESS_GROUP_HELPER") == "1" {
		for {
			time.Sleep(time.Second)
		}
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestProcessGroupHelper$")
	cmd.Env = append(
		os.Environ(),
		"UNISHELL_PROCESS_GROUP_HELPER=1",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("start process-group helper: %v", err)
	}

	return cmd
}

func waitForProcessGroupEmpty(
	t *testing.T,
	processGroupID int,
) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)

	for time.Now().Before(deadline) {
		members, err := ProcessGroupMembers(processGroupID)
		if err != nil {
			t.Fatalf(
				"ProcessGroupMembers(%d) returned error: %v",
				processGroupID,
				err,
			)
		}

		if len(members) == 0 {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	members, err := ProcessGroupMembers(processGroupID)
	if err != nil {
		t.Fatalf(
			"ProcessGroupMembers(%d) returned error after timeout: %v",
			processGroupID,
			err,
		)
	}

	t.Fatalf(
		"process group %d still contains processes after timeout: %v",
		processGroupID,
		members,
	)
}

func TestProcessGroupHelper(t *testing.T) {
	if os.Getenv("UNISHELL_PROCESS_GROUP_HELPER") != "1" {
		return
	}

	for {
		time.Sleep(time.Second)
	}
}

func TestProcessGroupMembersFindsProcess(t *testing.T) {
	cmd := startProcessGroupTestHelper(t)

	pgid, err := ProcessGroupID(cmd.Process.Pid)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessGroupID(%d) returned error: %v",
			cmd.Process.Pid,
			err,
		)
	}

	members, err := ProcessGroupMembers(pgid)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessGroupMembers(%d) returned error: %v",
			pgid,
			err,
		)
	}

	found := false

	for _, pid := range members {
		if pid == cmd.Process.Pid {
			found = true
			break
		}
	}

	if !found {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"process group %d does not contain helper pid %d; members = %v",
			pgid,
			cmd.Process.Pid,
			members,
		)
	}

	if err := cmd.Process.Kill(); err != nil && !errors.Is(
		err,
		os.ErrProcessDone,
	) {
		t.Fatalf("kill helper: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			t.Fatalf("wait for helper: %v", err)
		}
	}
}

func processIdentityForTestProcess(pid int) (ProcessIdentity, error) {
	startTicks, err := ProcessStartTicks(pid)
	if err != nil {
		return ProcessIdentity{}, err
	}

	processGroupID, err := ProcessGroupID(pid)
	if err != nil {
		return ProcessIdentity{}, err
	}

	return ProcessIdentity{
		PID:               pid,
		ProcessStartTicks: startTicks,
		ProcessGroupID:    processGroupID,
	}, nil
}

func TestTerminateProcessGroupTerminatesOwnedGroup(t *testing.T) {
	cmd := startProcessGroupTestHelper(t)

	identity, err := processIdentityForTestProcess(cmd.Process.Pid)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessIdentityForTestProcess(%d) returned error: %v",
			cmd.Process.Pid,
			err,
		)
	}

	if identity.ProcessGroupID == CurrentProcessGroupID() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"test helper unexpectedly shares test process group %d",
			identity.ProcessGroupID,
		)
	}

	if err := TerminateProcessGroup(identity); err != nil {
		t.Fatalf(
			"TerminateProcessGroup() returned error: %v",
			err,
		)
	}

	if err := cmd.Wait(); err == nil {
		t.Fatal(
			"helper process exited successfully after SIGKILL",
		)
	}

	waitForProcessGroupEmpty(
		t,
		identity.ProcessGroupID,
	)
}

func TestProcessGroupLiveMembersIgnoresZombieMembers(t *testing.T) {
	cmd := startProcessGroupTestHelper(t)

	identity, err := processIdentityForTestProcess(cmd.Process.Pid)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"processIdentityForTestProcess(%d) returned error: %v",
			cmd.Process.Pid,
			err,
		)
	}

	if err := unix.Kill(
		cmd.Process.Pid,
		unix.SIGKILL,
	); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()

		t.Fatalf(
			"kill helper process: %v",
			err,
		)
	}

	members, err := ProcessGroupLiveMembers(
		identity.ProcessGroupID,
	)
	if err != nil {
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessGroupLiveMembers() returned error: %v",
			err,
		)
	}

	if len(members) != 0 {
		_ = cmd.Wait()

		t.Fatalf(
			"ProcessGroupLiveMembers() = %v, want no live members",
			members,
		)
	}

	if err := cmd.Wait(); err == nil {
		t.Fatal(
			"helper process exited successfully after SIGKILL",
		)
	}
}

func TestTerminateProcessGroupRejectsStartTimeMismatch(t *testing.T) {
	cmd := startProcessGroupTestHelper(t)
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	identity, err := processIdentityForTestProcess(cmd.Process.Pid)
	if err != nil {
		t.Fatalf(
			"processIdentityForTestProcess(%d) returned error: %v",
			cmd.Process.Pid,
			err,
		)
	}

	identity.ProcessStartTicks++

	err = TerminateProcessGroup(identity)
	if err == nil {
		t.Fatal(
			"TerminateProcessGroup() returned nil for mismatched start time",
		)
	}

	if !strings.Contains(err.Error(), "process identity mismatch") {
		t.Fatalf(
			"TerminateProcessGroup() error = %q, want process identity mismatch",
			err,
		)
	}
}

func TestTerminateProcessGroupRejectsProcessGroupMismatch(t *testing.T) {
	cmd := startProcessGroupTestHelper(t)
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	identity, err := processIdentityForTestProcess(cmd.Process.Pid)
	if err != nil {
		t.Fatalf(
			"processIdentityForTestProcess(%d) returned error: %v",
			cmd.Process.Pid,
			err,
		)
	}

	identity.ProcessGroupID++

	err = TerminateProcessGroup(identity)
	if err == nil {
		t.Fatal(
			"TerminateProcessGroup() returned nil for mismatched process group",
		)
	}

	if !strings.Contains(
		err.Error(),
		"process group identity mismatch",
	) {
		t.Fatalf(
			"TerminateProcessGroup() error = %q, want process group identity mismatch",
			err,
		)
	}
}

func TestTerminateProcessGroupRejectsInvalidIdentity(t *testing.T) {
	tests := []struct {
		name     string
		identity ProcessIdentity
		want     string
	}{
		{
			name: "invalid pid",
			identity: ProcessIdentity{
				PID:               0,
				ProcessStartTicks: 1,
				ProcessGroupID:    1,
			},
			want: "invalid process ID",
		},
		{
			name: "invalid start time",
			identity: ProcessIdentity{
				PID:               1,
				ProcessStartTicks: 0,
				ProcessGroupID:    1,
			},
			want: "invalid process start time",
		},
		{
			name: "invalid process group",
			identity: ProcessIdentity{
				PID:               1,
				ProcessStartTicks: 1,
				ProcessGroupID:    0,
			},
			want: "invalid process group ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := TerminateProcessGroup(tt.identity)
			if err == nil {
				t.Fatal(
					"TerminateProcessGroup() returned nil",
				)
			}

			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf(
					"TerminateProcessGroup() error = %q, want %q",
					err,
					tt.want,
				)
			}
		})
	}
}

func TestTerminateProcessGroupRejectsCurrentProcessGroup(t *testing.T) {
	identity := ProcessIdentity{
		PID:               os.Getpid(),
		ProcessStartTicks: CurrentProcessStartTicks(),
		ProcessGroupID:    CurrentProcessGroupID(),
	}

	err := TerminateProcessGroup(identity)
	if err == nil {
		t.Fatal(
			"TerminateProcessGroup() returned nil for current process group",
		)
	}

	want := "refusing to terminate current process group"

	if !strings.Contains(err.Error(), want) {
		t.Fatalf(
			"TerminateProcessGroup() error = %q, want %q",
			err,
			want,
		)
	}
}

func TestProcessGroupTestHelper(t *testing.T) {
	if os.Getenv("UNISHELL_PROCESS_GROUP_TEST_HELPER") != "1" {
		return
	}

	for {
		time.Sleep(time.Second)
	}
}

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

func TestProcessIdentityForCurrentProcess(t *testing.T) {
	pid := os.Getpid()

	startTicks, err := ProcessStartTicks(pid)
	if err != nil {
		t.Fatalf(
			"ProcessStartTicks() returned error: %v",
			err,
		)
	}

	pgid, err := ProcessGroupID(pid)
	if err != nil {
		t.Fatalf(
			"ProcessGroupID() returned error: %v",
			err,
		)
	}

	identity := ProcessIdentity{
		PID:               pid,
		ProcessStartTicks: startTicks,
		ProcessGroupID:    pgid,
	}

	if identity.PID != pid {
		t.Fatalf(
			"identity PID = %d, want %d",
			identity.PID,
			pid,
		)
	}

	if identity.ProcessStartTicks == 0 {
		t.Fatal("identity process start ticks = 0")
	}

	if identity.ProcessGroupID <= 0 {
		t.Fatalf(
			"identity process group ID = %d, want positive value",
			identity.ProcessGroupID,
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
