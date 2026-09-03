package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

const sessionIDBytes = 16

type SessionMode string

const (
	SessionModeNormal      SessionMode = "normal"
	SessionModeMultiplexer SessionMode = "multiplexer"
)

// Session represents one isolated temporary uniShell runtime session.
type Session struct {
	Paths Paths
	ID    string
	Mode  SessionMode
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
		Mode:  SessionModeNormal,
	}, nil
}

// Prepare creates the private directories and session metadata required by
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

	metadata := sessionmeta.Metadata{
		ID:                s.ID,
		PID:               os.Getpid(),
		ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
		CreatedAt:         time.Now().UTC(),
		Version:           filepath.Base(filepath.Dir(s.Paths.Runtime)),
		Mode:              sessionmeta.Mode(s.Mode),
	}

	if metadata.ProcessStartTicks == 0 {
		_ = os.RemoveAll(s.Paths.Runtime)
		return errors.New("unable to determine current process start time")
	}

	if err := sessionmeta.WriteMetadata(
		s.Paths.Runtime,
		metadata,
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
// Only directories containing valid uniShell session metadata are
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

		metadata, err := sessionmeta.ReadMetadata(
			sessionRuntime,
		)
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

		if metadata.Mode == sessionmeta.ModeMultiplexer {
			continue
		}

		alive, err := sessionIsAlive(metadata)

		if err != nil {
			return fmt.Errorf(
				"check runtime session %q: %w",
				sessionRuntime,
				err,
			)
		}

		if alive {
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

func sessionIsAlive(metadata sessionmeta.Metadata) (bool, error) {
	switch metadata.Mode {
	case sessionmeta.ModeNormal:
		startTicks, err := sessionmeta.ProcessStartTicks(metadata.PID)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return false, nil
			}

			return false, err
		}

		return startTicks == metadata.ProcessStartTicks, nil

	case sessionmeta.ModeMultiplexer:
		return true, nil

	default:
		return false, fmt.Errorf(
			"unsupported runtime session mode %q",
			metadata.Mode,
		)
	}
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

func (s *Session) SetMode(mode SessionMode) error {
	switch mode {
	case SessionModeNormal, SessionModeMultiplexer:
		s.Mode = mode
		return nil

	default:
		return fmt.Errorf("unsupported runtime session mode %q", mode)
	}
}

func NewSessionWithMode(paths Paths, mode SessionMode) (*Session, error) {
	if mode != SessionModeNormal &&
		mode != SessionModeMultiplexer {
		return nil, fmt.Errorf(
			"unsupported runtime session mode %q",
			mode,
		)
	}

	session, err := NewSession(paths)
	if err != nil {
		return nil, err
	}

	session.Mode = mode

	return session, nil
}
