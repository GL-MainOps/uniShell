package app

import (
	"fmt"
	"io"
	"os"
	"time"
)

const startupTimingEnv = "UNISHELL_STARTUP_TIMING"

var startupTimingEnabled = os.Getenv(startupTimingEnv) == "1"

func traceStartup(stage string, started time.Time) {
	if !startupTimingEnabled {
		return
	}
	fmt.Fprintf(os.Stderr, "uniShell startup %s: %s\n", stage, time.Since(started))
}

func traceStartupDuration(stage string, elapsed time.Duration) {
	if !startupTimingEnabled {
		return
	}
	fmt.Fprintf(os.Stderr, "uniShell startup %s: %s\n", stage, elapsed)
}

type startupTimedReader struct {
	reader  io.Reader
	elapsed time.Duration
}

func (reader *startupTimedReader) Read(buffer []byte) (int, error) {
	started := time.Now()
	n, err := reader.reader.Read(buffer)
	reader.elapsed += time.Since(started)
	return n, err
}
