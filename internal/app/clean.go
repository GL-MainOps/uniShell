package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

type CleanSession struct {
	Metadata   sessionmeta.Metadata
	RuntimeDir string
}

func (a *App) DiscoverCleanSessions() ([]*CleanSession, error) {
	entries, err := os.ReadDir(a.Paths.Runtime)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"inspect clean sessions: %w",
			err,
		)
	}

	sessions := make([]*CleanSession, 0)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		runtimeDir := filepath.Join(
			a.Paths.Runtime,
			entry.Name(),
		)

		metadata, err := sessionmeta.ReadMetadata(runtimeDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, fmt.Errorf(
				"read session metadata %q: %w",
				runtimeDir,
				err,
			)
		}

		cleanSession := &CleanSession{
			Metadata:   metadata,
			RuntimeDir: runtimeDir,
		}

		sessions = append(sessions, cleanSession)
	}

	return sessions, nil
}

// TerminateNormalSession verifies and terminates the process associated
// with a normal uniShell session.
func (a *App) TerminateNormalSession(
	cleanSession *CleanSession,
) error {
	if cleanSession == nil {
		return fmt.Errorf("clean session cannot be nil")
	}

	if cleanSession.Metadata.Mode != sessionmeta.ModeNormal {
		return fmt.Errorf(
			"session %q is not a normal session",
			cleanSession.Metadata.Name,
		)
	}

	err := sessionmeta.TerminateProcess(
		cleanSession.Metadata.PID,
		cleanSession.Metadata.ProcessStartTicks,
	)

	if err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf(
			"terminate normal session %q: %w",
			cleanSession.Metadata.Name,
			err,
		)
	}

	runtimePath := cleanSession.RuntimeDir
	if runtimePath == "" {
		return fmt.Errorf(
			"normal session %q runtime path is empty",
			cleanSession.Metadata.Name,
		)
	}

	if err := os.RemoveAll(runtimePath); err != nil {
		return fmt.Errorf(
			"cleanup normal session %q: %w",
			cleanSession.Metadata.Name,
			err,
		)
	}

	return nil
}

func (a *App) CleanupMultiplexerSession(
	cleanSession *CleanSession,
) error {
	if cleanSession == nil {
		return fmt.Errorf("clean session cannot be nil")
	}

	if cleanSession.Metadata.Mode != sessionmeta.ModeMultiplexer {
		return fmt.Errorf(
			"session %q is not a multiplexer session",
			cleanSession.Metadata.Name,
		)
	}

	if a.Multiplexer == nil {
		return fmt.Errorf(
			"multiplexer manager is unavailable",
		)
	}

	runtimePath := cleanSession.RuntimeDir
	if runtimePath == "" {
		return fmt.Errorf(
			"multiplexer session %q runtime path is empty",
			cleanSession.Metadata.Name,
		)
	}

	if err := a.Multiplexer.Cleanup(runtimePath); err != nil {
		return fmt.Errorf(
			"cleanup multiplexer session %q: %w",
			cleanSession.Metadata.Name,
			err,
		)
	}

	return nil
}

// CleanPersistentRuntime intentionally removes all uniShell-managed runtime
// versions, sessions, cache data, and local settings below the configured root.
// The installed executable in ~/.local/bin is outside this directory and is
// preserved.
func (a *App) CleanPersistentRuntime() error {
	allSessions, err := a.DiscoverPersistentCleanSessions()
	if err != nil {
		return err
	}
	var cleanupErrors []error
	for _, session := range allSessions {
		switch session.Metadata.Mode {
		case sessionmeta.ModeNormal:
			if err := a.TerminateNormalSession(session); err != nil {
				cleanupErrors = append(cleanupErrors, err)
			}
		case sessionmeta.ModeMultiplexer:
			if err := a.CleanupMultiplexerSession(session); err != nil {
				cleanupErrors = append(cleanupErrors, err)
			}
		default:
			cleanupErrors = append(cleanupErrors, fmt.Errorf(
				"session %q uses unsupported termination mode %q",
				session.Metadata.Name,
				session.Metadata.Mode,
			))
		}
	}
	if len(cleanupErrors) > 0 {
		return errors.Join(cleanupErrors...)
	}

	if err := os.RemoveAll(filepath.Join(a.Paths.Root, "runtime")); err != nil {
		return fmt.Errorf("remove persistent runtime data: %w", err)
	}
	for _, name := range []string{".token", ".token.key", ".unishell-config.toml"} {
		path := filepath.Join(a.Paths.Root, name)
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove persistent file %q: %w", path, err)
		}
	}
	return nil
}

func (a *App) DiscoverPersistentCleanSessions() ([]*CleanSession, error) {
	runtimeRoot := filepath.Join(a.Paths.Root, "runtime")
	versions, err := os.ReadDir(runtimeRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("inspect persistent runtime versions: %w", err)
	}

	var sessions []*CleanSession
	for _, version := range versions {
		if !version.IsDir() || strings.HasPrefix(version.Name(), ".") {
			continue
		}
		versionPath := filepath.Join(runtimeRoot, version.Name())
		entries, err := os.ReadDir(versionPath)
		if err != nil {
			return nil, fmt.Errorf("inspect persistent runtime %q: %w", version.Name(), err)
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			path := filepath.Join(versionPath, entry.Name())
			metadata, err := sessionmeta.ReadMetadata(path)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue
				}
				return nil, fmt.Errorf("read persistent session metadata %q: %w", path, err)
			}
			sessions = append(sessions, &CleanSession{
				Metadata:   metadata,
				RuntimeDir: path,
			})
		}
	}
	return sessions, nil
}

func (a *App) IsPersistent() bool {
	return a.Persistent
}

func (a *App) PersistentRoot() string {
	return a.Paths.Root
}
