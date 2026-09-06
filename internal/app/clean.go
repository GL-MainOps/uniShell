package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

	if errors.Is(err, os.ErrProcessDone) {
		return nil
	}

	if err != nil {
		return fmt.Errorf(
			"terminate normal session %q: %w",
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
