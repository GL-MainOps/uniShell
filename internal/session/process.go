package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// ProcessIdentity identifies one live process instance and its process group.
type ProcessIdentity struct {
	PID               int
	ProcessStartTicks uint64
	ProcessGroupID    int
}

// CurrentProcessStartTicks returns the kernel start-time tick count of the
// current process.
//
// A zero value indicates that the process start time could not be determined.
func CurrentProcessStartTicks() uint64 {
	ticks, err := ProcessStartTicks(os.Getpid())
	if err != nil {
		return 0
	}

	return ticks
}

// CurrentProcessGroupID returns the process group ID of the current process.
//
// A zero value indicates that the process group ID could not be determined.
func CurrentProcessGroupID() int {
	pgid, err := ProcessGroupID(os.Getpid())
	if err != nil {
		return 0
	}

	return pgid
}

// ProcessStartTicks returns the kernel start-time tick count for pid.
func ProcessStartTicks(pid int) (uint64, error) {
	data, err := os.ReadFile(
		filepath.Join("/proc", strconv.Itoa(pid), "stat"),
	)
	if err != nil {
		return 0, err
	}

	line := string(data)

	endComm := strings.LastIndex(line, ") ")
	if endComm == -1 || endComm+2 >= len(line) {
		return 0, errors.New("invalid process stat")
	}

	fields := strings.Fields(line[endComm+2:])
	if len(fields) <= 19 {
		return 0, errors.New("invalid process stat fields")
	}

	return strconv.ParseUint(fields[19], 10, 64)
}

// ProcessGroupID returns the process group ID for pid.
func ProcessGroupID(pid int) (int, error) {
	if pid <= 0 {
		return 0, fmt.Errorf("invalid process ID")
	}

	data, err := os.ReadFile(
		filepath.Join("/proc", strconv.Itoa(pid), "stat"),
	)
	if err != nil {
		return 0, err
	}

	line := string(data)

	endComm := strings.LastIndex(line, ") ")
	if endComm == -1 || endComm+2 >= len(line) {
		return 0, errors.New("invalid process stat")
	}

	fields := strings.Fields(line[endComm+2:])
	if len(fields) <= 2 {
		return 0, errors.New("invalid process stat fields")
	}

	pgid, err := strconv.Atoi(fields[2])
	if err != nil {
		return 0, fmt.Errorf(
			"parse process group ID: %w",
			err,
		)
	}

	if pgid <= 0 {
		return 0, errors.New("invalid process group ID")
	}

	return pgid, nil
}

func TerminateProcess(
	pid int,
	processStartTicks uint64,
) error {
	if pid <= 0 {
		return fmt.Errorf("invalid process ID")
	}

	if processStartTicks == 0 {
		return fmt.Errorf("invalid process start time")
	}

	pidfd, err := unix.PidfdOpen(pid, 0)
	if err != nil {
		if errors.Is(err, unix.ESRCH) {
			return os.ErrProcessDone
		}

		return fmt.Errorf(
			"open process descriptor: %w",
			err,
		)
	}

	defer unix.Close(pidfd)

	currentStartTicks, err := ProcessStartTicks(pid)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return os.ErrProcessDone
		}

		return fmt.Errorf(
			"read process identity: %w",
			err,
		)
	}

	if currentStartTicks != processStartTicks {
		return fmt.Errorf(
			"process identity mismatch for pid %d",
			pid,
		)
	}

	if err := unix.PidfdSendSignal(
		pidfd,
		unix.SIGKILL,
		nil,
		0,
	); err != nil {
		if errors.Is(err, unix.ESRCH) {
			return os.ErrProcessDone
		}

		return fmt.Errorf(
			"terminate process %d: %w",
			pid,
			err,
		)
	}

	return nil
}

// ProcessGroupMembers returns the PIDs currently belonging to processGroupID.
func ProcessGroupMembers(processGroupID int) ([]int, error) {
	if processGroupID <= 0 {
		return nil, fmt.Errorf("invalid process group ID")
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read proc filesystem: %w", err)
	}

	members := make([]int, 0)

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		pgid, err := ProcessGroupID(pid)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			continue
		}

		if pgid == processGroupID {
			members = append(members, pid)
		}
	}

	return members, nil
}

func ProcessGroupLiveMembers(processGroupID int) ([]int, error) {
	if processGroupID <= 0 {
		return nil, fmt.Errorf("invalid process group ID")
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read proc filesystem: %w", err)
	}

	members := make([]int, 0)

	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		pgid, err := ProcessGroupID(pid)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			continue
		}

		if pgid != processGroupID {
			continue
		}

		state, err := processState(pid)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			continue
		}

		if state != 'Z' {
			members = append(members, pid)
		}
	}

	return members, nil
}

func processState(pid int) (byte, error) {
	data, err := os.ReadFile(
		filepath.Join("/proc", strconv.Itoa(pid), "stat"),
	)
	if err != nil {
		return 0, err
	}

	line := string(data)

	endComm := strings.LastIndex(line, ") ")
	if endComm == -1 || endComm+2 >= len(line) {
		return 0, errors.New("invalid process stat")
	}

	fields := strings.Fields(line[endComm+2:])
	if len(fields) == 0 || len(fields[0]) == 0 {
		return 0, errors.New("invalid process stat fields")
	}

	return fields[0][0], nil
}

// TerminateProcessGroup terminates the process group identified by identity.
func TerminateProcessGroup(identity ProcessIdentity) error {
	if identity.PID <= 0 {
		return fmt.Errorf("invalid process ID")
	}

	if identity.ProcessStartTicks == 0 {
		return fmt.Errorf("invalid process start time")
	}

	if identity.ProcessGroupID <= 0 {
		return fmt.Errorf("invalid process group ID")
	}

	if identity.ProcessGroupID == CurrentProcessGroupID() {
		return fmt.Errorf(
			"refusing to terminate current process group %d",
			identity.ProcessGroupID,
		)
	}

	currentStartTicks, err := ProcessStartTicks(identity.PID)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return verifyProcessGroupTerminated(identity.ProcessGroupID)
		}

		return fmt.Errorf(
			"read process identity: %w",
			err,
		)
	}

	if currentStartTicks != identity.ProcessStartTicks {
		return fmt.Errorf(
			"process identity mismatch for pid %d",
			identity.PID,
		)
	}

	currentProcessGroupID, err := ProcessGroupID(identity.PID)
	if err != nil {
		return fmt.Errorf(
			"read process group identity: %w",
			err,
		)
	}

	if currentProcessGroupID != identity.ProcessGroupID {
		return fmt.Errorf(
			"process group identity mismatch for pid %d",
			identity.PID,
		)
	}

	if err := unix.Kill(
		-identity.ProcessGroupID,
		unix.SIGKILL,
	); err != nil {
		if !errors.Is(err, unix.ESRCH) {
			return fmt.Errorf(
				"terminate process group %d: %w",
				identity.ProcessGroupID,
				err,
			)
		}
	}

	return verifyProcessGroupTerminated(identity.ProcessGroupID)
}

func verifyProcessGroupTerminated(processGroupID int) error {
	members, err := ProcessGroupLiveMembers(processGroupID)
	if err != nil {
		return fmt.Errorf(
			"verify process group %d termination: %w",
			processGroupID,
			err,
		)
	}

	if len(members) != 0 {
		return fmt.Errorf(
			"process group %d still contains processes: %v",
			processGroupID,
			members,
		)
	}

	return nil
}
