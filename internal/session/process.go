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
