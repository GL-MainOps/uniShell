package multiplexer

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	sessionmeta "gitlab.com/mainops/uniShell/internal/session"
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

func prepareManagerTestRuntime(
	t *testing.T,
	runtimePath string,
) {
	t.Helper()

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf(
			"create manager test runtime %q: %v",
			runtimePath,
			err,
		)
	}
}

func TestManagerCreateWritesMetadata(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}
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

	metadata, err := sessionmeta.ReadMetadata(runtimePath)
	if err != nil {
		t.Fatalf("sessionmeta.ReadMetadata() returned error: %v", err)
	}

	if metadata.ID != session.Metadata.ID {
		t.Fatalf(
			"metadata ID = %q, want %q",
			metadata.ID,
			session.Metadata.ID,
		)
	}

	if metadata.PID != os.Getpid() {
		t.Fatalf(
			"metadata PID = %d, want %d",
			metadata.PID,
			os.Getpid(),
		)
	}

	if metadata.ProcessStartTicks == 0 {
		t.Fatal("metadata process start ticks is zero")
	}

	if metadata.Version == "" {
		t.Fatal("metadata version is empty")
	}

	if metadata.Mode != sessionmeta.ModeMultiplexer {
		t.Fatalf(
			"metadata mode = %q, want %q",
			metadata.Mode,
			sessionmeta.ModeMultiplexer,
		)
	}

	if metadata.CreatedAt.IsZero() {
		t.Fatal("metadata creation time is zero")
	}

	if sessionmeta.MetadataPath(runtimePath) != filepath.Join(
		runtimePath,
		".session.json",
	) {
		t.Fatalf(
			"metadata path = %q, want session-local .session.json",
			sessionmeta.MetadataPath(runtimePath),
		)
	}

	if _, err := os.Stat(
		filepath.Join(
			runtimePath,
			"multiplexer",
			"session.json",
		),
	); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf(
			"legacy multiplexer metadata still exists, stat error = %v",
			err,
		)
	}
}

func TestManagerCreatePassesEnvironmentToBackend(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)
	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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

func TestManagerCreatePassesMultiplexerOptionsToBackend(
	t *testing.T,
) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	prepareManagerTestRuntime(t, runtimePath)

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
	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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

	if _, err := sessionmeta.ReadMetadata(runtimePath); err == nil {
		t.Fatal("sessionmeta.ReadMetadata() succeeded after Destroy()")
	}
}

func TestManagerDiscoverFindsLiveSession(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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
	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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

	prepareManagerTestRuntime(t, runtimePath)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
		alive:     false,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	metadata := sessionmeta.Metadata{
		PID:               os.Getpid(),
		ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
		CreatedAt:         time.Now().UTC(),
		Version:           "development",
		Mode:              sessionmeta.ModeMultiplexer,
		ID:                "stale-session",
		Name:              "default",
		NativeName:        "native-default",
		Multiplexer:       "test",
		Endpoint:          "/tmp/test.endpoint",
	}

	if err := sessionmeta.WriteMetadata(runtimePath, metadata); err != nil {
		t.Fatalf("sessionmeta.WriteMetadata() returned error: %v", err)
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

	prepareManagerTestRuntime(t, runtimePath)

	backend := &managerTestBackend{
		name:      "test",
		available: false,
		alive:     true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	metadata := sessionmeta.Metadata{
		PID:               os.Getpid(),
		ProcessStartTicks: sessionmeta.CurrentProcessStartTicks(),
		CreatedAt:         time.Now().UTC(),
		Version:           "development",
		Mode:              sessionmeta.ModeMultiplexer,
		ID:                "session",
		Name:              "default",
		NativeName:        "native-default",
		Multiplexer:       "test",
		Endpoint:          "/tmp/test.endpoint",
	}

	if err := sessionmeta.WriteMetadata(runtimePath, metadata); err != nil {
		t.Fatalf("sessionmeta.WriteMetadata() returned error: %v", err)
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

	prepareManagerTestRuntime(t, firstRuntime)
	prepareManagerTestRuntime(t, secondRuntime)

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

func TestManagerDiscoverAllFindsManagedSessions(
	t *testing.T,
) {
	versionRuntime := filepath.Join(
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

	firstRuntime := filepath.Join(
		versionRuntime,
		"session-one",
	)

	secondRuntime := filepath.Join(
		versionRuntime,
		"session-two",
	)

	if err := os.MkdirAll(firstRuntime, 0700); err != nil {
		t.Fatalf(
			"create first runtime path: %v",
			err,
		)
	}

	if err := os.MkdirAll(secondRuntime, 0700); err != nil {
		t.Fatalf(
			"create second runtime path: %v",
			err,
		)
	}

	if _, err := manager.Create(
		"test",
		"development",
		"native-development",
		firstRuntime,
		filepath.Join(firstRuntime, "endpoint"),
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf(
			"Create() first session returned error: %v",
			err,
		)
	}

	if _, err := manager.Create(
		"test",
		"production",
		"native-production",
		secondRuntime,
		filepath.Join(secondRuntime, "endpoint"),
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf(
			"Create() second session returned error: %v",
			err,
		)
	}

	sessions, err := manager.DiscoverAll(versionRuntime)
	if err != nil {
		t.Fatalf(
			"DiscoverAll() returned error: %v",
			err,
		)
	}

	if len(sessions) != 2 {
		t.Fatalf(
			"session count = %d, want %d",
			len(sessions),
			2,
		)
	}

	if sessions[0].Metadata.Name != "development" {
		t.Fatalf(
			"first session name = %q, want %q",
			sessions[0].Metadata.Name,
			"development",
		)
	}

	if sessions[1].Metadata.Name != "production" {
		t.Fatalf(
			"second session name = %q, want %q",
			sessions[1].Metadata.Name,
			"production",
		)
	}
}

func TestManagerDiscoverAllReturnsEmptyForMissingRuntime(
	t *testing.T,
) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"missing",
	)

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	sessions, err := manager.DiscoverAll(versionRuntime)
	if err != nil {
		t.Fatalf(
			"DiscoverAll() returned error: %v",
			err,
		)
	}

	if len(sessions) != 0 {
		t.Fatalf(
			"session count = %d, want %d",
			len(sessions),
			0,
		)
	}
}

func TestManagerDiscoverAllIgnoresDirectoriesWithoutMetadata(
	t *testing.T,
) {
	versionRuntime := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	if err := os.MkdirAll(
		filepath.Join(versionRuntime, "unrelated"),
		0700,
	); err != nil {
		t.Fatalf(
			"create unrelated directory: %v",
			err,
		)
	}

	backend := &managerTestBackend{
		name:      "test",
		available: true,
	}

	manager := NewManager(
		NewRegistry(backend),
	)

	sessions, err := manager.DiscoverAll(versionRuntime)
	if err != nil {
		t.Fatalf(
			"DiscoverAll() returned error: %v",
			err,
		)
	}

	if len(sessions) != 0 {
		t.Fatalf(
			"session count = %d, want %d",
			len(sessions),
			0,
		)
	}
}

func TestManagerDiscoverAllIncludesExitedManagedSessions(
	t *testing.T,
) {
	versionRuntime := filepath.Join(
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

	runtimePath := filepath.Join(
		versionRuntime,
		"session-one",
	)

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

	if _, err := manager.Create(
		"test",
		"development",
		"native-development",
		runtimePath,
		filepath.Join(runtimePath, "endpoint"),
		"",
		"",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf(
			"Create() returned error: %v",
			err,
		)
	}

	sessions, err := manager.DiscoverAll(versionRuntime)
	if err != nil {
		t.Fatalf(
			"DiscoverAll() returned error: %v",
			err,
		)
	}

	if len(sessions) != 1 {
		t.Fatalf(
			"session count = %d, want %d",
			len(sessions),
			1,
		)
	}

	if sessions[0].Metadata.Name != "development" {
		t.Fatalf(
			"session name = %q, want %q",
			sessions[0].Metadata.Name,
			"development",
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

	prepareManagerTestRuntime(t, sessionRuntime)

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

	prepareManagerTestRuntime(t, sessionRuntime)

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

	prepareManagerTestRuntime(t, sessionRuntime)

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

	prepareManagerTestRuntime(t, runtimePath)

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

	prepareManagerTestRuntime(t, runtimePath)

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

	prepareManagerTestRuntime(t, runtimePath)

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

	prepareManagerTestRuntime(t, runtimePath)

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

	prepareManagerTestRuntime(t, runtimePath)

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

	prepareManagerTestRuntime(t, runtimePath)

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

	metadata, err := sessionmeta.ReadMetadata(runtimePath)
	if err != nil {
		t.Fatalf("sessionmeta.ReadMetadata() returned error: %v", err)
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

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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

	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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
	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

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

func TestManagerCreateDoesNotCreateLegacyMultiplexerMetadata(
	t *testing.T,
) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)
	if err := os.MkdirAll(runtimePath, 0700); err != nil {
		t.Fatalf("create runtime path: %v", err)
	}

	backend := &managerTestBackend{
		name:      "test",
		available: true,
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
		"bash",
		"/bin/bash",
		nil,
		nil,
		api.Options{},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	legacyPath := filepath.Join(
		runtimePath,
		"multiplexer",
		"session.json",
	)

	if _, err := os.Stat(legacyPath); !errors.Is(
		err,
		os.ErrNotExist,
	) {
		t.Fatalf(
			"legacy multiplexer metadata exists, stat error = %v",
			err,
		)
	}

	if _, err := os.Stat(
		filepath.Join(runtimePath, ".session.json"),
	); err != nil {
		t.Fatalf(
			"unified session metadata does not exist: %v",
			err,
		)
	}
}
