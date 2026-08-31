package multiplexer

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
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
	nativeName string,
	runtimePath string,
	endpoint string,
	shellName string,
	shellPath string,
	shellArgs []string,
	env []string,
	options api.Options,
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
		return nil, fmt.Errorf(
			"generate multiplexer session ID: %w",
			err,
		)
	}

	session := Session{
		Name:       sessionName,
		NativeName: nativeName,
		Runtime:    runtimePath,
		Endpoint:   endpoint,
		ShellName:  shellName,
		ShellPath:  shellPath,
		ShellArgs:  append([]string(nil), shellArgs...),
		Env:        append([]string(nil), env...),
		Options:    options,
	}

	var createdNativeName = nativeName

	if creator, ok := backend.(api.NativeNameCreator); ok {
		createdNativeName, err = creator.CreateWithNativeName(
			session,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"create %s session: %w",
				backendName,
				err,
			)
		}
	} else if err := backend.Create(session); err != nil {
		return nil, fmt.Errorf(
			"create %s session: %w",
			backendName,
			err,
		)
	}

	session.NativeName = createdNativeName

	metadata := Metadata{
		ID:          id,
		Name:        sessionName,
		NativeName:  createdNativeName,
		Multiplexer: backendName,
		Endpoint:    endpoint,
		ShellName:   shellName,
		ShellPath:   shellPath,
		CreatedAt:   time.Now().UTC(),
	}

	if err := WriteMetadata(
		runtimePath,
		metadata,
	); err != nil {
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

	backend, ok := m.registry.Get(
		metadata.Multiplexer,
	)
	if !ok {
		return nil, fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	session := Session{
		Name:       metadata.Name,
		NativeName: metadata.NativeName,
		Runtime:    runtimePath,
		Endpoint:   metadata.Endpoint,
		ShellName:  metadata.ShellName,
		ShellPath:  metadata.ShellPath,
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

	backend, ok := m.registry.Get(
		metadata.Multiplexer,
	)
	if !ok {
		return fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	session := Session{
		Name:       metadata.Name,
		NativeName: metadata.NativeName,
		Runtime:    runtimePath,
		Endpoint:   metadata.Endpoint,
		ShellName:  metadata.ShellName,
		ShellPath:  metadata.ShellPath,
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

func (m *Manager) Discover(
	runtimePath string,
	sessionName string,
) (*ManagedSession, error) {
	metadata, err := ReadMetadata(runtimePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, err
	}

	if metadata.Name != sessionName {
		return nil, ErrSessionNotFound
	}

	backend, ok := m.registry.Get(
		metadata.Multiplexer,
	)
	if !ok {
		return nil, fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	if !backend.Available() {
		return nil, fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	session := Session{
		Name:       metadata.Name,
		NativeName: metadata.NativeName,
		Runtime:    runtimePath,
		Endpoint:   metadata.Endpoint,
		ShellName:  metadata.ShellName,
		ShellPath:  metadata.ShellPath,
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

func (m *Manager) DiscoverByName(
	versionRuntime string,
	sessionName string,
) (*ManagedSession, error) {
	entries, err := os.ReadDir(versionRuntime)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, fmt.Errorf(
			"inspect runtime sessions: %w",
			err,
		)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		runtimePath := filepath.Join(
			versionRuntime,
			entry.Name(),
		)

		session, err := m.Discover(
			runtimePath,
			sessionName,
		)
		if err == nil {
			return session, nil
		}

		if errors.Is(err, ErrSessionNotFound) {
			continue
		}

		return nil, err
	}

	return nil, ErrSessionNotFound
}

func (m *Manager) Reconcile(
	versionRuntime string,
) error {
	entries, err := os.ReadDir(versionRuntime)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf(
			"inspect runtime sessions: %w",
			err,
		)
	}

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

			return fmt.Errorf(
				"read session metadata %q: %w",
				runtimePath,
				err,
			)
		}

		backend, ok := m.registry.Get(
			metadata.Multiplexer,
		)
		if !ok {
			continue
		}

		if !backend.Available() {
			continue
		}

		session := Session{
			Name:       metadata.Name,
			NativeName: metadata.NativeName,
			Runtime:    runtimePath,
			Endpoint:   metadata.Endpoint,
			ShellName:  metadata.ShellName,
			ShellPath:  metadata.ShellPath,
		}

		if backend.IsAlive(session) {
			continue
		}

		if err := os.RemoveAll(runtimePath); err != nil {
			return fmt.Errorf(
				"remove stale multiplexer session %q: %w",
				runtimePath,
				err,
			)
		}
	}

	return nil
}

func (m *Manager) ReconcileSession(
	runtimePath string,
) error {
	if runtimePath == "" {
		return fmt.Errorf(
			"multiplexer runtime path cannot be empty",
		)
	}

	metadata, err := ReadMetadata(runtimePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return os.RemoveAll(runtimePath)
		}

		return err
	}

	backend, ok := m.registry.Get(
		metadata.Multiplexer,
	)
	if !ok {
		return fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	if !backend.Available() {
		return fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	session := Session{
		Name:       metadata.Name,
		NativeName: metadata.NativeName,
		Runtime:    runtimePath,
		Endpoint:   metadata.Endpoint,
		ShellName:  metadata.ShellName,
		ShellPath:  metadata.ShellPath,
	}

	if backend.IsAlive(session) {
		return nil
	}

	if err := os.RemoveAll(runtimePath); err != nil {
		return fmt.Errorf(
			"remove stale multiplexer session %q: %w",
			runtimePath,
			err,
		)
	}

	return nil
}

func (m *Manager) Cleanup(
	runtimePath string,
) error {
	if runtimePath == "" {
		return fmt.Errorf(
			"multiplexer runtime path cannot be empty",
		)
	}

	metadata, err := ReadMetadata(runtimePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return os.RemoveAll(runtimePath)
		}

		return err
	}

	backend, ok := m.registry.Get(
		metadata.Multiplexer,
	)
	if !ok {
		return fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	if !backend.Available() {
		return fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
	}

	session := Session{
		Name:       metadata.Name,
		NativeName: metadata.NativeName,
		Runtime:    runtimePath,
		Endpoint:   metadata.Endpoint,
		ShellName:  metadata.ShellName,
		ShellPath:  metadata.ShellPath,
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

	if err := os.RemoveAll(runtimePath); err != nil {
		return fmt.Errorf(
			"remove multiplexer runtime %q: %w",
			runtimePath,
			err,
		)
	}

	return nil
}
