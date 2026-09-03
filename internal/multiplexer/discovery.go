package multiplexer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type DiscoveryState string

const (
	DiscoveryMissing DiscoveryState = "missing"
	DiscoveryLive    DiscoveryState = "live"
)

type DiscoveryResult struct {
	State   DiscoveryState
	Session *ManagedSession
}

func (m *Manager) DiscoverState(
	runtimePath string,
	sessionName string,
) (DiscoveryResult, error) {
	session, err := m.Discover(runtimePath, sessionName)
	if err == nil {
		return DiscoveryResult{
			State:   DiscoveryLive,
			Session: session,
		}, nil
	}

	if errors.Is(err, ErrSessionNotFound) {
		return DiscoveryResult{
			State: DiscoveryMissing,
		}, nil
	}

	return DiscoveryResult{}, err
}

// DiscoverAll returns every managed multiplexer session beneath the
// supplied version runtime directory.
//
// A managed session is identified by valid multiplexer metadata. Session
// liveness is intentionally not checked here because clean must also be
// able to discover runtimes belonging to sessions that have already
// exited.
func (m *Manager) DiscoverAll(
	versionRuntime string,
) ([]*ManagedSession, error) {
	entries, err := os.ReadDir(versionRuntime)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"inspect runtime sessions: %w",
			err,
		)
	}

	sessions := make([]*ManagedSession, 0)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		runtimePath := filepath.Join(
			versionRuntime,
			entry.Name(),
		)

		metadata, err := ReadMetadata(runtimePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf(
				"read session metadata %q: %w",
				runtimePath,
				err,
			)
		}

		backend, ok := m.registry.Get(metadata.Multiplexer)
		if !ok {
			continue
		}

		sessions = append(
			sessions,
			&ManagedSession{
				Metadata: metadata,
				Backend:  backend,
				Session: Session{
					Name:       metadata.Name,
					NativeName: metadata.NativeName,
					Runtime:    runtimePath,
					Endpoint:   metadata.Endpoint,
					ShellName:  metadata.ShellName,
					ShellPath:  metadata.ShellPath,
				},
			},
		)
	}

	return sessions, nil
}
