package tmux

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

func TestCreateUsesSessionEndpointAndName(t *testing.T) {
	var gotArgs [][]string

	endpoint := filepath.Join(
		t.TempDir(),
		"multiplexer",
		"tmux.sock",
	)

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append(
				gotArgs,
				append([]string(nil), args...),
			)
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/bash",
		Endpoint:   endpoint,
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := [][]string{
		{
			"-S",
			endpoint,
			"new-session",
			"-d",
			"-s",
			"work",
			"/runtime/bin/bash",
		},
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestCreateSetsShellForSession(t *testing.T) {
	var gotArgs [][]string

	endpoint := filepath.Join(
		t.TempDir(),
		"multiplexer",
		"tmux.sock",
	)

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append(
				gotArgs,
				append([]string(nil), args...),
			)
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/zsh",
		Endpoint:   endpoint,
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if len(gotArgs) != 1 {
		t.Fatalf("tmux invocations = %d, want 1", len(gotArgs))
	}

	want := []string{
		"-S",
		endpoint,
		"new-session",
		"-d",
		"-s",
		"work",
		"/runtime/bin/zsh",
	}

	if !reflect.DeepEqual(gotArgs[0], want) {
		t.Fatalf(
			"tmux invocation args = %#v, want %#v",
			gotArgs[0],
			want,
		)
	}
}

func TestCreateRejectsEmptyShellPath(t *testing.T) {
	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, _ ...string) error {
			t.Fatal("Run() must not be called")
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		Runtime:    t.TempDir(),
		Endpoint:   "/tmp/unishell.sock",
	})

	if err == nil {
		t.Fatal("Create() returned nil error, want empty shell path error")
	}
}

func TestCreateUsesSessionEnvironment(t *testing.T) {
	var gotArgs []string

	endpoint := filepath.Join(
		t.TempDir(),
		"multiplexer",
		"tmux.sock",
	)

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/bash",
		Endpoint:   endpoint,
		Env: []string{
			"PATH=/runtime/work/bin:/usr/bin",
			"SHELL=/bin/bash",
		},
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"-S",
		endpoint,
		"new-session",
		"-d",
		"-e",
		"PATH=/runtime/work/bin:/usr/bin",
		"-e",
		"SHELL=/bin/bash",
		"-s",
		"work",
		"/runtime/bin/bash",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestCreateRejectsMissingEndpoint(t *testing.T) {
	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, _ ...string) error {
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
	})

	if err == nil {
		t.Fatal("Create() returned nil error")
	}
}

func TestAttachUsesSessionEndpointAndName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Attach(api.Session{
		NativeName: "work",
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	})
	if err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"attach-session",
		"-t",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestDetachUsesSessionEndpointAndName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Detach(api.Session{
		NativeName: "work",
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	})
	if err != nil {
		t.Fatalf("Detach() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"detach-client",
		"-s",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestIsAliveUsesSessionEndpointAndName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			t.Fatal("IsAlive() must not use Run")
			return nil
		},
		RunQuiet: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if !backend.IsAlive(api.Session{
		NativeName: "work",
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	}) {
		t.Fatal("IsAlive() = false, want true")
	}

	want := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"has-session",
		"-t",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestIsAliveReturnsFalseWhenSessionDoesNotExist(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		RunQuiet: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return fmt.Errorf("no server running")
		},
	}

	if backend.IsAlive(api.Session{
		NativeName: "missing",
		Endpoint:   "/runtime/missing/multiplexer/tmux.sock",
	}) {
		t.Fatal("IsAlive() = true, want false")
	}

	want := []string{
		"-S",
		"/runtime/missing/multiplexer/tmux.sock",
		"has-session",
		"-t",
		"missing",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestDestroyUsesSessionEndpointAndName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Destroy(api.Session{
		NativeName: "work",
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	})
	if err != nil {
		t.Fatalf("Destroy() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"kill-session",
		"-t",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestCreateUsesConfiguredOptions(t *testing.T) {
	var gotArgs [][]string

	endpoint := filepath.Join(
		t.TempDir(),
		"multiplexer",
		"tmux.sock",
	)

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append(
				gotArgs,
				append([]string(nil), args...),
			)
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/bash",
		Endpoint:   endpoint,
		Options: api.Options{
			Tmux: api.TmuxOptions{
				CreateArgs: []string{
					"--test-option",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := [][]string{
		{
			"-S",
			endpoint,
			"new-session",
			"-d",
			"--test-option",
			"-s",
			"work",
			"/runtime/bin/bash",
		},
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			want,
		)
	}
}

func TestCreateUsesBundledConfig(t *testing.T) {
	runtime := t.TempDir()

	config := filepath.Join(
		runtime,
		"config",
		"tmux",
		"tmux.conf",
	)

	if err := os.MkdirAll(
		filepath.Dir(config),
		0700,
	); err != nil {
		t.Fatalf("create config directory: %v", err)
	}

	if err := os.WriteFile(
		config,
		[]byte("set -g mouse on\n"),
		0600,
	); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var gotArgs [][]string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append(
				gotArgs,
				append([]string(nil), args...),
			)
			return nil
		},
	}

	session := api.Session{
		Name:       "work",
		NativeName: "work",
		ShellPath:  "/runtime/bin/bash",
		Runtime:    runtime,
		Endpoint:   filepath.Join(runtime, "tmux.sock"),
	}

	if err := backend.Create(session); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"-f",
		config,
		"-S",
		session.Endpoint,
		"new-session",
		"-d",
		"-s",
		"work",
		"/runtime/bin/bash",
	}

	if len(gotArgs) != 1 ||
		!reflect.DeepEqual(
			gotArgs[0],
			want,
		) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			[][]string{want},
		)
	}
}

func TestCreateRejectsLifecycleOptions(t *testing.T) {
	tests := []string{
		"-S",
		"-L",
		"-c",
		"-D",
		"-N",
	}

	for _, option := range tests {
		t.Run(option, func(t *testing.T) {
			backend := &Backend{
				Binary: "fake-tmux",
				Run: func(_ string, _ ...string) error {
					t.Fatal("Run() must not be called")
					return nil
				},
			}

			err := backend.Create(api.Session{
				NativeName: "work",
				Runtime:    t.TempDir(),
				Endpoint:   "/tmp/unishell.sock",
				Options: api.Options{
					Tmux: api.TmuxOptions{
						CreateArgs: []string{option},
					},
				},
			})

			if err == nil {
				t.Fatalf(
					"Create() returned nil error for %q",
					option,
				)
			}
		})
	}
}

func TestCreatePreparesSocketDirectory(t *testing.T) {
	runtime := t.TempDir()
	endpoint := filepath.Join(
		runtime,
		"multiplexer",
		"tmux.sock",
	)

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, _ ...string) error {
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/bin/bash",
		Endpoint:   endpoint,
	})

	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	info, err := os.Stat(filepath.Dir(endpoint))
	if err != nil {
		t.Fatalf("stat socket directory: %v", err)
	}

	if !info.IsDir() {
		t.Fatal("socket parent is not a directory")
	}

	if info.Mode().Perm() != 0700 {
		t.Fatalf(
			"socket parent permissions = %o, want %o",
			info.Mode().Perm(),
			0700,
		)
	}
}

func TestPrepareSocketPathCreatesPrivateParent(t *testing.T) {
	socketPath := filepath.Join(
		t.TempDir(),
		"multiplexer",
		"tmux.sock",
	)

	if err := prepareSocketPath(socketPath); err != nil {
		t.Fatalf(
			"prepareSocketPath() returned error: %v",
			err,
		)
	}

	info, err := os.Stat(filepath.Dir(socketPath))
	if err != nil {
		t.Fatalf("stat socket directory: %v", err)
	}

	if !info.IsDir() {
		t.Fatal("socket parent is not a directory")
	}

	if info.Mode().Perm() != 0700 {
		t.Fatalf(
			"socket parent permissions = %o, want %o",
			info.Mode().Perm(),
			0700,
		)
	}
}
