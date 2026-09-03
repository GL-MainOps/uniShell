package app

import (
	"fmt"
	"os"
	"path/filepath"

	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
)

type CleanSession struct {
	Metadata    sessionmeta.Metadata
	RuntimeDir  string
	Multiplexer *Session
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

		if metadata.Mode == sessionmeta.ModeMultiplexer {
			managed, err := a.Multiplexer.Discover(
				runtimeDir,
				metadata.Name,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"discover multiplexer session %q: %w",
					metadata.Name,
					err,
				)
			}

			cleanSession.Multiplexer = &Session{
				Multiplexer: managed,
			}
		}

		sessions = append(sessions, cleanSession)
	}

	return sessions, nil
}
