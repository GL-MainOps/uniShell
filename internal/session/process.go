package session

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
