package multiplexer

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
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
	Metadata sessionmeta.Metadata
	Backend  Backend
	Session  Session
}

func sessionEnvironment(
	env []string,
	metadata sessionmeta.Metadata,
) []string {
	result := append([]string(nil), env...)

	for key, value := range metadata.Environment() {
		prefix := key + "="

		replaced := false
		filtered := result[:0]

		for _, entry := range result {
			if strings.HasPrefix(entry, prefix) {
				if !replaced {
					filtered = append(
						filtered,
						prefix+value,
					)
					replaced = true
				}
				continue
			}

			filtered = append(filtered, entry)
		}

		result = filtered

		if !replaced {
			result = append(result, prefix+value)
		}
	}

	return result
}

func (m *Manager) Create(
	backendName string,
	sessionName string,
	nativeName string,
	runtimePath string,
	shellName string,
	shellPath string,
	shellArgs []string,
	env []string,
	options api.Options,
	metadataHint ...sessionmeta.Metadata,
) (*ManagedSession, error) {
	backend, ok := m.registry.Get(backendName)
	if !ok {
		return nil, fmt.Errorf(
			"multiplexer %q: %w",
			backendName,
			ErrUnavailable,
		)
	}

	endpoint, err := backend.ResolveEndpoint(
		runtimePath,
		options,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve %s endpoint: %w",
			backendName,
			err,
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

	if !backend.AvailableForSession(session) {
		return nil, fmt.Errorf(
			"multiplexer %q: %w",
			backendName,
			ErrUnavailable,
		)
	}

	var existingMetadata sessionmeta.Metadata
	if len(metadataHint) > 0 && metadataHint[0].ID != "" {
		existingMetadata = metadataHint[0]
	} else {
		existingMetadata, err = sessionmeta.ReadMetadata(runtimePath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf(
				"read existing session metadata: %w",
				err,
			)
		}
	}

	runtimeSessionName := existingMetadata.Name
	if runtimeSessionName == "" {
		runtimeSessionName = sessionName
	}

	var sessionMetadata = sessionmeta.Metadata{
		ID:                     id,
		Version:                filepath.Base(filepath.Dir(runtimePath)),
		Mode:                   sessionmeta.ModeMultiplexer,
		Name:                   runtimeSessionName,
		NativeName:             nativeName,
		MultiplexerSessionName: sessionName,
		Multiplexer:            backendName,
		Endpoint:               endpoint,
		ShellName:              shellName,
		ShellPath:              shellPath,
		ShellProfile:           existingMetadata.ShellProfile,
	}

	session.Env = sessionEnvironment(
		session.Env,
		sessionMetadata,
	)

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
	sessionMetadata.NativeName = createdNativeName

	identityProvider, ok := backend.(api.ProcessIdentityProvider)
	if !ok {
		_ = backend.Destroy(session)

		return nil, fmt.Errorf(
			"multiplexer %q does not provide managed process identity",
			backendName,
		)
	}

	identity, err := identityProvider.ProcessIdentity(session)
	if err != nil {
		_ = backend.Destroy(session)

		return nil, fmt.Errorf(
			"discover %s session process identity: %w",
			backendName,
			err,
		)
	}

	if identity.PID <= 0 ||
		identity.ProcessStartTicks == 0 ||
		identity.ProcessGroupID <= 0 {
		_ = backend.Destroy(session)

		return nil, fmt.Errorf(
			"discover %s session process identity: invalid identity",
			backendName,
		)
	}

	sessionMetadata.PID = identity.PID
	sessionMetadata.ProcessStartTicks = identity.ProcessStartTicks
	sessionMetadata.ProcessGroupID = identity.ProcessGroupID
	sessionMetadata.CreatedAt = time.Now().UTC()

	if err := sessionmeta.WriteMetadata(
		runtimePath,
		sessionMetadata,
	); err != nil {
		_ = backend.Destroy(session)

		return nil, err
	}

	return &ManagedSession{
		Metadata: sessionMetadata,
		Backend:  backend,
		Session:  session,
	}, nil
}

func (m *Manager) Attach(
	runtimePath string,
) (*ManagedSession, error) {
	metadata, err := sessionmeta.ReadMetadata(runtimePath)
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
	metadata, err := sessionmeta.ReadMetadata(runtimePath)
	if err != nil {
		return err
	}

	processIdentity := sessionmeta.ProcessIdentity{
		PID:               metadata.PID,
		ProcessStartTicks: metadata.ProcessStartTicks,
		ProcessGroupID:    metadata.ProcessGroupID,
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

	if err := sessionmeta.TerminateProcessGroup(processIdentity); err != nil {
		return fmt.Errorf(
			"terminate %s session process group: %w",
			metadata.Multiplexer,
			err,
		)
	}

	return sessionmeta.RemoveMetadata(runtimePath)
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
	sessionID string,
) (*ManagedSession, error) {
	metadata, err := sessionmeta.ReadMetadata(runtimePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrSessionNotFound
		}

		return nil, err
	}

	if metadata.Mode != sessionmeta.ModeMultiplexer {
		return nil, ErrSessionNotFound
	}

	if metadata.ID != sessionID {
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

	session := Session{
		Name:       metadata.Name,
		NativeName: metadata.NativeName,
		Runtime:    runtimePath,
		Endpoint:   metadata.Endpoint,
		ShellName:  metadata.ShellName,
		ShellPath:  metadata.ShellPath,
	}

	if !backend.AvailableForSession(session) {
		return nil, fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
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

func (m *Manager) DiscoverByID(
	versionRuntime string,
	sessionID string,
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
			sessionID,
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

		metadata, err := sessionmeta.ReadMetadata(runtimePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, err
		}

		if metadata.Mode != sessionmeta.ModeMultiplexer {
			continue
		}

		baseName := strings.TrimSpace(sessionName)
		if baseName == "" {
			baseName = "unnamed"
		}

		if !strings.HasPrefix(
			metadata.MultiplexerSessionName,
			baseName+"@",
		) {
			continue
		}

		return m.DiscoverByID(
			versionRuntime,
			metadata.ID,
		)
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

		metadata, err := sessionmeta.ReadMetadata(runtimePath)
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

		session := Session{
			Name:       metadata.Name,
			NativeName: metadata.NativeName,
			Runtime:    runtimePath,
			Endpoint:   metadata.Endpoint,
			ShellName:  metadata.ShellName,
			ShellPath:  metadata.ShellPath,
		}

		if !backend.AvailableForSession(session) {
			continue
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

	metadata, err := sessionmeta.ReadMetadata(runtimePath)
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

	session := Session{
		Name:       metadata.Name,
		NativeName: metadata.NativeName,
		Runtime:    runtimePath,
		Endpoint:   metadata.Endpoint,
		ShellName:  metadata.ShellName,
		ShellPath:  metadata.ShellPath,
	}

	if !backend.AvailableForSession(session) {
		return fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
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

	metadata, err := sessionmeta.ReadMetadata(runtimePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return os.RemoveAll(runtimePath)
		}

		return err
	}

	processIdentity := sessionmeta.ProcessIdentity{
		PID:               metadata.PID,
		ProcessStartTicks: metadata.ProcessStartTicks,
		ProcessGroupID:    metadata.ProcessGroupID,
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

	if !backend.AvailableForSession(session) {
		return fmt.Errorf(
			"multiplexer %q: %w",
			metadata.Multiplexer,
			ErrUnavailable,
		)
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

	if err := sessionmeta.TerminateProcessGroup(processIdentity); err != nil {
		return fmt.Errorf(
			"terminate %s session process group: %w",
			metadata.Multiplexer,
			err,
		)
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
