package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	sessionMarkerName = ".unishell-session"
	sessionIDBytes    = 16
)

type sessionMarker struct {
	SessionID         string `json:"session_id"`
	PID               int    `json:"pid"`
	ProcessStartTicks uint64 `json:"process_start_ticks"`
	StartedAt         int64  `json:"started_at"`
	Version           string `json:"version"`
}

// Session represents one isolated temporary uniShell runtime session.
type Session struct {
	Paths Paths
	ID    string
}

// NewSession creates a new isolated runtime session.
//
// The filesystem is not modified until Prepare is called.
func NewSession(paths Paths) (*Session, error) {
	id, err := newSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate runtime session ID: %w", err)
	}

	sessionPaths := paths
	sessionPaths.Runtime = filepath.Join(paths.Runtime, id)
	sessionPaths.Bin = filepath.Join(sessionPaths.Runtime, "bin")
	sessionPaths.Config = filepath.Join(sessionPaths.Runtime, "config")

	return &Session{
		Paths: sessionPaths,
		ID:    id,
	}, nil
}

// Prepare creates the private directories and session marker required by
// the session.
func (s *Session) Prepare() error {
	if s.ID == "" {
		return errors.New("runtime session ID cannot be empty")
	}

	if err := os.MkdirAll(s.Paths.Root, 0700); err != nil {
		return runtimeFilesystemError(
			s.Paths.Root,
			"create runtime root",
			err,
		)
	}

	if err := os.MkdirAll(
		filepath.Dir(s.Paths.Runtime),
		0700,
	); err != nil {
		return runtimeFilesystemError(
			filepath.Dir(s.Paths.Runtime),
			"create runtime version directory",
			err,
		)
	}

	if err := os.Mkdir(s.Paths.Runtime, 0700); err != nil {
		return runtimeFilesystemError(
			s.Paths.Runtime,
			"create runtime session directory",
			err,
		)
	}

	if err := os.Mkdir(s.Paths.Bin, 0700); err != nil {
		return runtimeFilesystemError(
			s.Paths.Bin,
			"create runtime bin directory",
			err,
		)
	}

	if err := os.Mkdir(s.Paths.Config, 0700); err != nil {
		return runtimeFilesystemError(
			s.Paths.Config,
			"create runtime config directory",
			err,
		)
	}

	marker := sessionMarker{
		SessionID:         s.ID,
		PID:               os.Getpid(),
		ProcessStartTicks: currentProcessStartTicks(),
		StartedAt:         time.Now().UnixNano(),
		Version:           filepath.Base(filepath.Dir(s.Paths.Runtime)),
	}

	if err := writeSessionMarker(
		filepath.Join(s.Paths.Runtime, sessionMarkerName),
		marker,
	); err != nil {
		_ = os.RemoveAll(s.Paths.Runtime)
		return err
	}

	return nil
}

// Cleanup removes this session's ephemeral runtime.
//
// Cleanup is idempotent.
func (s *Session) Cleanup() error {
	if err := os.RemoveAll(s.Paths.Runtime); err != nil {
		return runtimeFilesystemError(
			s.Paths.Runtime,
			"remove temporary runtime",
			err,
		)
	}

	return nil
}

// CleanupStale removes stale sessions beneath the supplied version
// directory.
//
// Only directories containing a valid uniShell session marker are
// considered. Unknown directories are left untouched.
func CleanupStale(paths Paths) error {
	entries, err := os.ReadDir(paths.Runtime)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return runtimeFilesystemError(
			paths.Runtime,
			"inspect stale runtime",
			err,
		)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		sessionRuntime := filepath.Join(paths.Runtime, entry.Name())
		markerPath := filepath.Join(sessionRuntime, sessionMarkerName)

		marker, err := readSessionMarker(markerPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return fmt.Errorf(
				"inspect runtime session %q: %w",
				sessionRuntime,
				err,
			)
		}

		if sessionIsAlive(marker) {
			continue
		}

		if err := os.RemoveAll(sessionRuntime); err != nil {
			return runtimeFilesystemError(
				sessionRuntime,
				"remove stale runtime",
				err,
			)
		}
	}

	return nil
}

// IsWithinRoot verifies that a path belongs to the configured runtime
// root.
func IsWithinRoot(paths Paths, target string) bool {
	root, err := filepath.Abs(paths.Root)
	if err != nil {
		return false
	}

	target, err = filepath.Abs(target)
	if err != nil {
		return false
	}

	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}

	return relative != ".." &&
		len(relative) >= 3 &&
		relative[:3] != ".."+string(filepath.Separator)
}

func newSessionID() (string, error) {
	buffer := make([]byte, sessionIDBytes)

	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	return hex.EncodeToString(buffer), nil
}

func writeSessionMarker(path string, marker sessionMarker) error {
	data, err := json.Marshal(marker)
	if err != nil {
		return fmt.Errorf("encode runtime session marker: %w", err)
	}

	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0600,
	)
	if err != nil {
		return runtimeFilesystemError(
			path,
			"create runtime session marker",
			err,
		)
	}

	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write runtime session marker: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close runtime session marker: %w", err)
	}

	return nil
}

func readSessionMarker(path string) (sessionMarker, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return sessionMarker{}, err
	}

	var marker sessionMarker

	if err := json.Unmarshal(data, &marker); err != nil {
		return sessionMarker{}, fmt.Errorf(
			"decode runtime session marker: %w",
			err,
		)
	}

	if marker.SessionID == "" ||
		marker.PID <= 0 ||
		marker.ProcessStartTicks == 0 ||
		marker.StartedAt == 0 ||
		marker.Version == "" {
		return sessionMarker{}, errors.New(
			"invalid runtime session marker",
		)
	}

	return marker, nil
}

func sessionIsAlive(marker sessionMarker) bool {
	startTicks, err := processStartTicks(marker.PID)
	if err != nil {
		return false
	}

	return startTicks == marker.ProcessStartTicks
}

func currentProcessStartTicks() uint64 {
	ticks, err := processStartTicks(os.Getpid())
	if err != nil {
		return 0
	}

	return ticks
}

func processStartTicks(pid int) (uint64, error) {
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

func runtimeFilesystemError(
	path string,
	action string,
	err error,
) error {
	if errors.Is(err, os.ErrPermission) {
		return &PermissionError{
			Path:   path,
			Action: action,
		}
	}

	return fmt.Errorf("%s %q: %w", action, path, err)
}
