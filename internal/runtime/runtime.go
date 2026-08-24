package runtime

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Session represents one temporary uniShell runtime session.
type Session struct {
	Paths Paths
}

// NewSession creates a runtime session for the supplied paths.
//
// The filesystem is not modified until Prepare is called.
func NewSession(paths Paths) *Session {
	return &Session{
		Paths: paths,
	}
}

// Prepare creates the directories required by the session.
func (s *Session) Prepare() error {
	directories := []string{
		s.Paths.Root,
		s.Paths.Runtime,
		s.Paths.Bin,
		s.Paths.Config,
	}

	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0700); err != nil {
			if errors.Is(err, os.ErrPermission) {
				return &PermissionError{
					Path:   directory,
					Action: "create runtime directory",
				}
			}

			return fmt.Errorf("create runtime directory %q: %w", directory, err)
		}
	}

	return nil
}

// Cleanup removes the session's ephemeral runtime.
//
// Cleanup is intentionally best-effort. If the runtime cannot be removed,
// the error is returned so the caller can report it to the user.
func (s *Session) Cleanup() error {
	if err := os.RemoveAll(s.Paths.Runtime); err != nil {
		if errors.Is(err, os.ErrPermission) {
			return &PermissionError{
				Path:   s.Paths.Runtime,
				Action: "remove temporary runtime",
			}
		}

		return fmt.Errorf(
			"remove temporary runtime %q: %w",
			s.Paths.Runtime,
			err,
		)
	}

	return nil
}

// CleanupStale removes an existing versioned runtime before starting
// a new session.
//
// This protects against runtimes left behind by abnormal termination.
func CleanupStale(paths Paths) error {
	if _, err := os.Stat(paths.Runtime); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		if errors.Is(err, os.ErrPermission) {
			return &PermissionError{
				Path:   paths.Runtime,
				Action: "inspect stale runtime",
			}
		}

		return fmt.Errorf(
			"inspect stale runtime %q: %w",
			paths.Runtime,
			err,
		)
	}

	session := NewSession(paths)

	return session.Cleanup()
}

// IsWithinRoot verifies that a path belongs to the configured runtime root.
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
