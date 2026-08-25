package multiplexer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

type Manager struct {
	registry *Registry
}

func NewManager(registry *Registry) *Manager {
	return &Manager{
		registry: registry,
	}
}

type ManagedSession struct {
	Metadata Metadata
	Backend  Backend
	Session  Session
}

func (m *Manager) Create(
	backendName string,
	sessionName string,
	runtimePath string,
	endpoint string,
) (*ManagedSession, error) {
	backend, ok := m.registry.Get(backendName)
	if !ok {
		return nil, fmt.Errorf(
			"multiplexer %q: %w",
			backendName,
			ErrUnavailable,
		)
	}

	if !backend.Available() {
		return nil, fmt.Errorf(
			"multiplexer %q: %w",
			backendName,
			ErrUnavailable,
		)
	}

	id, err := generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("generate multiplexer session ID: %w", err)
	}

	session := Session{
		Name:     sessionName,
		Runtime:  runtimePath,
		Endpoint: endpoint,
	}

	if err := backend.Create(session); err != nil {
		return nil, fmt.Errorf(
			"create %s session: %w",
			backendName,
			err,
		)
	}

	metadata := Metadata{
		ID:          id,
		Name:        sessionName,
		Multiplexer: backendName,
		Endpoint:    endpoint,
		CreatedAt:   time.Now().UTC(),
	}

	if err := WriteMetadata(runtimePath, metadata); err != nil {
		// The backend session was created but metadata failed.
		// Attempt to avoid leaving an unmanaged session behind.
		_ = backend.Destroy(session)

		return nil, err
	}

	return &ManagedSession{
		Metadata: metadata,
		Backend:  backend,
		Session:  session,
	}, nil
}

func (m *Manager) Attach(
	runtimePath string,
) (*ManagedSession, error) {
	metadata, err := ReadMetadata(runtimePath)
	if err != nil {
		return nil, err
	}

	backend, ok := m.registry.Get(metadata.Multiplexer)
	if !ok {
		return nil, fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	session := Session{
		Name:     metadata.Name,
		Runtime:  runtimePath,
		Endpoint: metadata.Endpoint,
	}

	if !backend.IsAlive(session) {
		return nil, ErrSessionNotFound
	}

	return &ManagedSession{
		Metadata: metadata,
		Backend:  backend,
		Session:  session,
	}, nil
}

func (m *Manager) Destroy(
	runtimePath string,
) error {
	metadata, err := ReadMetadata(runtimePath)
	if err != nil {
		return err
	}

	backend, ok := m.registry.Get(metadata.Multiplexer)
	if !ok {
		return fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	session := Session{
		Name:     metadata.Name,
		Runtime:  runtimePath,
		Endpoint: metadata.Endpoint,
	}

	if backend.IsAlive(session) {
		if err := backend.Destroy(session); err != nil {
			return fmt.Errorf(
				"destroy %s session: %w",
				metadata.Multiplexer,
				err,
			)
		}
	}

	return RemoveMetadata(runtimePath)
}

func generateSessionID() (string, error) {
	data := make([]byte, 16)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	return hex.EncodeToString(data), nil
}
