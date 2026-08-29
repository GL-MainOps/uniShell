package multiplexer

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

const endpoint = "/tmp/test.endpoint"

type managerTestBackend struct {
	name      string
	available bool
	alive     bool

	created        bool
	createdSession Session
	destroyed      bool
}

func (b *managerTestBackend) Name() string {
	return b.name
}

func (b *managerTestBackend) Capabilities() map[Capability]bool {
	return map[Capability]bool{
		CapabilitySessions: true,
		CapabilityAttach:   true,
		CapabilityDetach:   true,
		CapabilityDestroy:  true,
	}
}

func (b *managerTestBackend) Available() bool {
	return b.available
}

func (b *managerTestBackend) Create(session Session) error {
	b.created = true
	b.createdSession = session
	b.alive = true
	return nil
}

func (b *managerTestBackend) Attach(Session) error {
	return nil
}

func (b *managerTestBackend) Detach(Session) error {
	return nil
}

func (b *managerTestBackend) IsAlive(Session) bool {
	return b.alive
}

func (b *managerTestBackend) Destroy(Session) error {
	b.destroyed = true
	b.alive = false
	return nil
}

func TestManagerCreateWritesMetadata(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	session, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if !backend.created {
		t.Fatal("backend Create() was not called")
	}

	if session.Metadata.ID == "" {
		t.Fatal("session ID is empty")
	}

	if session.Metadata.Name != "default" {
		t.Fatalf(
			"session name = %q, want %q",
			session.Metadata.Name,
			"default",
		)
	}

	if session.Metadata.NativeName != "" {
		t.Fatalf(
			"session native name = %q, want empty",
			session.Metadata.NativeName,
		)
	}

	metadata, err := ReadMetadata(runtimePath)
	if err != nil {
		t.Fatalf("ReadMetadata() returned error: %v", err)
	}

	if metadata.ID != session.Metadata.ID {
		t.Fatalf(
			"metadata ID = %q, want %q",
			metadata.ID,
			session.Metadata.ID,
		)
	}
}

func TestManagerCreatePassesEnvironmentToBackend(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	env := []string{
		"PATH=/runtime/work/bin:/usr/bin",
		"SHELL=/bin/bash",
	}

	_, err := manager.Create(
		"test",
		"default",
		"native-work",
		runtimePath,
		endpoint,
		"bash",
		"/bin/bash",
		nil,
		env,
		api.Options{},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if !reflect.DeepEqual(
		backend.createdSession.Env,
		env,
	) {
		t.Fatalf(
			"backend environment = %#v, want %#v",
			backend.createdSession.Env,
			env,
		)
	}

	if backend.createdSession.Name != "default" {
		t.Fatalf(
			"backend session name = %q, want %q",
			backend.createdSession.Name,
			"default",
		)
	}

	if backend.createdSession.NativeName != "native-work" {
		t.Fatalf(
			"backend native name = %q, want %q",
			backend.createdSession.NativeName,
			"native-work",
		)
	}

	if backend.createdSession.Endpoint != endpoint {
		t.Fatalf(
			"backend endpoint = %q, want %q",
			backend.createdSession.Endpoint,
			endpoint,
		)
	}
}

func TestManagerCreatePassesShellToBackend(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	shellName := "bash"
	shellPath := "/runtime/bin/bash"

	session, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		shellName,
		shellPath,
		nil,
		nil,
		api.Options{},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if backend.createdSession.ShellName != shellName {
		t.Fatalf(
			"backend shell name = %q, want %q",
			backend.createdSession.ShellName,
			shellName,
		)
	}

	if backend.createdSession.ShellPath != shellPath {
		t.Fatalf(
			"backend shell path = %q, want %q",
			backend.createdSession.ShellPath,
			shellPath,
		)
	}

	if session.Session.ShellName != shellName {
		t.Fatalf(
			"session shell name = %q, want %q",
			session.Session.ShellName,
			shellName,
		)
	}

	if session.Session.ShellPath != shellPath {
		t.Fatalf(
			"session shell path = %q, want %q",
			session.Session.ShellPath,
			shellPath,
		)
	}
}

func TestManagerCreatePassesMultiplexerOptionsToBackend(
	t *testing.T,
) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	options := api.Options{
		Tmux: api.TmuxOptions{
			CreateArgs: []string{
				"--test-create",
			},
		},
		Zellij: api.ZellijOptions{
			CreateArgs: []string{
				"--zellij-create",
			},
		},
	}

	_, err := manager.Create(
		"test",
		"default",
		"native-work",
		runtimePath,
		endpoint,
		"bash",
		"/bin/bash",
		nil,
		nil,
		options,
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if !reflect.DeepEqual(
		backend.createdSession.Options,
		options,
	) {
		t.Fatalf(
			"backend options = %#v, want %#v",
			backend.createdSession.Options,
			options,
		)
	}
}

func TestManagerAttachRequiresLiveSession(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	created, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	attached, err := manager.Attach(runtimePath)
	if err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	if attached.Metadata.ID != created.Metadata.ID {
		t.Fatalf(
			"attached ID = %q, want %q",
			attached.Metadata.ID,
			created.Metadata.ID,
		)
	}

	backend.alive = false

	_, err = manager.Attach(runtimePath)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf(
			"Attach() error = %v, want %v",
			err,
			ErrSessionNotFound,
		)
	}
}

func TestManagerDestroyRemovesMetadata(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if err := manager.Destroy(runtimePath); err != nil {
		t.Fatalf("Destroy() returned error: %v", err)
	}

	if !backend.destroyed {
		t.Fatal("backend Destroy() was not called")
	}

	if _, err := ReadMetadata(runtimePath); err == nil {
		t.Fatal("ReadMetadata() succeeded after Destroy()")
	}
}

func TestManagerDiscoverFindsLiveSession(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	created, err := manager.Create(
		"test",
		"default",
		"native-default",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	discovered, err := manager.Discover(
		runtimePath,
		"default",
	)
	if err != nil {
		t.Fatalf("Discover() returned error: %v", err)
	}

	if discovered.Metadata.ID != created.Metadata.ID {
		t.Fatalf(
			"discovered ID = %q, want %q",
			discovered.Metadata.ID,
			created.Metadata.ID,
		)
	}

	if discovered.Metadata.Name != "default" {
		t.Fatalf(
			"discovered name = %q, want %q",
			discovered.Metadata.Name,
			"default",
		)
	}

	if discovered.Metadata.NativeName != "native-default" {
		t.Fatalf(
			"discovered native name = %q, want %q",
			discovered.Metadata.NativeName,
			"native-default",
		)
	}

	if discovered.Session.Name != "default" {
		t.Fatalf(
			"discovered session name = %q, want %q",
			discovered.Session.Name,
			"default",
		)
	}

	if discovered.Session.NativeName != "native-default" {
		t.Fatalf(
			"discovered native name = %q, want %q",
			discovered.Session.NativeName,
			"native-default",
		)
	}
}

func TestManagerDiscoverRejectsDifferentSessionName(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	_, err := manager.Discover(
		runtimePath,
		"work",
	)

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf(
			"Discover() error = %v, want %v",
			err,
			ErrSessionNotFound,
		)
	}
}

func TestManagerDiscoverRejectsStaleMetadata(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     false,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	metadata := Metadata{
		ID:          "stale-session",
		Name:        "default",
		NativeName:  "native-default",
		Multiplexer: "test",
		Endpoint:    "/tmp/test.endpoint",
	}

	if err := WriteMetadata(runtimePath, metadata); err != nil {
		t.Fatalf("WriteMetadata() returned error: %v", err)
	}

	_, err := manager.Discover(
		runtimePath,
		"default",
	)

	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf(
			"Discover() error = %v, want %v",
			err,
			ErrSessionNotFound,
		)
	}
}

func TestManagerDiscoverRejectsUnavailableBackend(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: false,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	metadata := Metadata{
		ID:          "session",
		Name:        "default",
		NativeName:  "native-default",
		Multiplexer: "test",
		Endpoint:    "/tmp/test.endpoint",
	}

	if err := WriteMetadata(runtimePath, metadata); err != nil {
		t.Fatalf("WriteMetadata() returned error: %v", err)
	}

	_, err := manager.Discover(
		runtimePath,
		"default",
	)

	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf(
			"Discover() error = %v, want %v",
			err,
			ErrUnavailable,
		)
	}
}

func TestManagerDiscoverByNameFindsSessionAcrossRuntimeDirectories(t *testing.T) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	firstRuntime := filepath.Join(versionRuntime, "first")
	secondRuntime := filepath.Join(versionRuntime, "second")

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"other",
		"",
		firstRuntime,
		"/tmp/first.endpoint",
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create(first) returned error: %v", err)
	}

	if _, err := manager.Create(
		"test",
		"default",
		"",
		secondRuntime,
		"/tmp/second.endpoint",
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create(second) returned error: %v", err)
	}

	session, err := manager.DiscoverByName(
		versionRuntime,
		"default",
	)
	if err != nil {
		t.Fatalf(
			"DiscoverByName() returned error: %v",
			err,
		)
	}

	if session.Session.Runtime != secondRuntime {
		t.Fatalf(
			"runtime = %q, want %q",
			session.Session.Runtime,
			secondRuntime,
		)
	}
}

func TestManagerReconcilePreservesLiveSession(t *testing.T) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	sessionRuntime := filepath.Join(
		versionRuntime,
		"session",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		sessionRuntime,
		"/tmp/test.endpoint",
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(sessionRuntime, "payload"),
		[]byte("runtime"),
		0600,
	); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	if err := manager.Reconcile(versionRuntime); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	if _, err := os.Stat(sessionRuntime); err != nil {
		t.Fatalf(
			"live session was removed: %v",
			err,
		)
	}
}

func TestManagerReconcileRemovesDeadSession(t *testing.T) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	sessionRuntime := filepath.Join(
		versionRuntime,
		"session",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     false,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		sessionRuntime,
		"/tmp/test.endpoint",
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	backend.alive = false

	if err := manager.Reconcile(versionRuntime); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	if _, err := os.Stat(sessionRuntime); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf(
			"dead session runtime still exists, stat error = %v",
			err,
		)
	}
}

func TestManagerReconcileIgnoresUnavailableBackend(t *testing.T) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	sessionRuntime := filepath.Join(
		versionRuntime,
		"session",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		sessionRuntime,
		"/tmp/test.endpoint",
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	backend.available = false
	backend.alive = false

	if err := manager.Reconcile(versionRuntime); err != nil {
		t.Fatalf("Reconcile() returned error: %v", err)
	}

	if _, err := os.Stat(sessionRuntime); err != nil {
		t.Fatalf(
			"session was removed while backend was unavailable: %v",
			err,
		)
	}
}

func TestManagerCreatePreservesNativeSessionName(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	session, err := manager.Create(
		"test",
		"default",
		"native-work",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if session.Metadata.Name != "default" {
		t.Fatalf(
			"metadata name = %q, want %q",
			session.Metadata.Name,
			"default",
		)
	}

	if session.Metadata.NativeName != "native-work" {
		t.Fatalf(
			"metadata native name = %q, want %q",
			session.Metadata.NativeName,
			"native-work",
		)
	}

	if session.Session.Name != "default" {
		t.Fatalf(
			"session name = %q, want %q",
			session.Session.Name,
			"default",
		)
	}

	if session.Session.NativeName != "native-work" {
		t.Fatalf(
			"session native name = %q, want %q",
			session.Session.NativeName,
			"native-work",
		)
	}
}

func TestManagerCreatePreservesEmptyNativeSessionName(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	session, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if session.Metadata.Name != "default" {
		t.Fatalf(
			"metadata name = %q, want %q",
			session.Metadata.Name,
			"default",
		)
	}

	if session.Metadata.NativeName != "" {
		t.Fatalf(
			"metadata native name = %q, want empty",
			session.Metadata.NativeName,
		)
	}

	if session.Session.Name != "default" {
		t.Fatalf(
			"session name = %q, want %q",
			session.Session.Name,
			"default",
		)
	}

	if session.Session.NativeName != "" {
		t.Fatalf(
			"session native name = %q, want empty",
			session.Session.NativeName,
		)
	}
}

func TestManagerCleanupDestroysLiveSessionAndRemovesRuntime(
	t *testing.T,
) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if err := manager.Cleanup(runtimePath); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}

	if !backend.destroyed {
		t.Fatal("Cleanup() did not destroy live session")
	}

	if _, err := os.Stat(runtimePath); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf(
			"runtime still exists, stat error = %v",
			err,
		)
	}
}

func TestManagerCleanupRemovesStaleSessionRuntime(
	t *testing.T,
) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     false,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	backend.alive = false

	if err := manager.Cleanup(runtimePath); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}

	if _, err := os.Stat(runtimePath); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf(
			"stale runtime still exists, stat error = %v",
			err,
		)
	}
}

func TestManagerCleanupPreservesRuntimeWhenBackendUnavailable(
	t *testing.T,
) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	backend.available = false

	err := manager.Cleanup(runtimePath)

	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf(
			"Cleanup() error = %v, want %v",
			err,
			ErrUnavailable,
		)
	}

	if _, err := os.Stat(runtimePath); err != nil {
		t.Fatalf(
			"runtime was removed while backend unavailable: %v",
			err,
		)
	}
}

func TestManagerCreatePersistsShell(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	const (
		shellName = "zsh"
		shellPath = "/runtime/bin/zsh"
	)

	session, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		shellName,
		shellPath,
		nil,
		nil,
		api.Options{},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if session.Metadata.ShellName != shellName {
		t.Fatalf(
			"metadata shell name = %q, want %q",
			session.Metadata.ShellName,
			shellName,
		)
	}

	if session.Metadata.ShellPath != shellPath {
		t.Fatalf(
			"metadata shell path = %q, want %q",
			session.Metadata.ShellPath,
			shellPath,
		)
	}

	if session.Session.ShellName != shellName {
		t.Fatalf(
			"session shell name = %q, want %q",
			session.Session.ShellName,
			shellName,
		)
	}

	if session.Session.ShellPath != shellPath {
		t.Fatalf(
			"session shell path = %q, want %q",
			session.Session.ShellPath,
			shellPath,
		)
	}

	metadata, err := ReadMetadata(runtimePath)
	if err != nil {
		t.Fatalf("ReadMetadata() returned error: %v", err)
	}

	if metadata.ShellName != shellName {
		t.Fatalf(
			"stored shell name = %q, want %q",
			metadata.ShellName,
			shellName,
		)
	}

	if metadata.ShellPath != shellPath {
		t.Fatalf(
			"stored shell path = %q, want %q",
			metadata.ShellPath,
			shellPath,
		)
	}
}

func TestManagerAttachPreservesShell(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	created, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"fish",
		"/runtime/bin/fish",
		nil,
		nil,
		api.Options{},
	)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	attached, err := manager.Attach(runtimePath)
	if err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	if attached.Session.ShellName != created.Session.ShellName {
		t.Fatalf(
			"attached shell name = %q, want %q",
			attached.Session.ShellName,
			created.Session.ShellName,
		)
	}

	if attached.Session.ShellPath != created.Session.ShellPath {
		t.Fatalf(
			"attached shell path = %q, want %q",
			attached.Session.ShellPath,
			created.Session.ShellPath,
		)
	}
}

func TestManagerReconcileSessionPreservesLiveSession(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(runtimePath, "payload"),
		[]byte("runtime"),
		0600,
	); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	if err := manager.ReconcileSession(runtimePath); err != nil {
		t.Fatalf(
			"ReconcileSession() returned error: %v",
			err,
		)
	}

	if _, err := os.Stat(runtimePath); err != nil {
		t.Fatalf(
			"live session runtime was removed: %v",
			err,
		)
	}

	if backend.destroyed {
		t.Fatal(
			"ReconcileSession() destroyed live session",
		)
	}
}

func TestManagerReconcileSessionRemovesExitedSession(
	t *testing.T,
) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	// Simulate the user exiting the multiplexer session.
	backend.alive = false

	if err := manager.ReconcileSession(runtimePath); err != nil {
		t.Fatalf(
			"ReconcileSession() returned error: %v",
			err,
		)
	}

	if _, err := os.Stat(runtimePath); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf(
			"exited session runtime still exists, stat error = %v",
			err,
		)
	}

	if backend.destroyed {
		t.Fatal(
			"ReconcileSession() destroyed an already exited session",
		)
	}
}

func TestManagerReconcileSessionPreservesRuntimeWhenBackendUnavailable(
	t *testing.T,
) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	backend.available = false

	err := manager.ReconcileSession(runtimePath)

	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf(
			"ReconcileSession() error = %v, want %v",
			err,
			ErrUnavailable,
		)
	}

	if _, err := os.Stat(runtimePath); err != nil {
		t.Fatalf(
			"runtime was removed while backend was unavailable: %v",
			err,
		)
	}

	if backend.destroyed {
		t.Fatal(
			"ReconcileSession() destroyed session while backend was unavailable",
		)
	}
}

func TestManagerReconcileSessionIsIdempotent(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	if _, err := manager.Create(
		"test",
		"default",
		"",
		runtimePath,
		endpoint,
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	backend.alive = false

	if err := manager.ReconcileSession(runtimePath); err != nil {
		t.Fatalf(
			"first ReconcileSession() returned error: %v",
			err,
		)
	}

	if err := manager.ReconcileSession(runtimePath); err != nil {
		t.Fatalf(
			"second ReconcileSession() returned error: %v",
			err,
		)
	}

	if _, err := os.Stat(runtimePath); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf(
			"runtime still exists after repeated reconciliation, stat error = %v",
			err,
		)
	}
}
