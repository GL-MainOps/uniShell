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
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := [][]string{
		{
			"-S",
			"/runtime/work/multiplexer/tmux.sock",
			"set-option",
			"-g",
			"default-shell",
			"/runtime/bin/bash",
		},
		{
			"-S",
			"/runtime/work/multiplexer/tmux.sock",
			"new-session",
			"-d",
			"-s",
			"work",
		},
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestCreateSetsShellBeforeCreatingSession(t *testing.T) {
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

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/zsh",
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if len(gotArgs) != 2 {
		t.Fatalf("tmux invocations = %d, want 2", len(gotArgs))
	}

	wantSetShell := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"set-option",
		"-g",
		"default-shell",
		"/runtime/bin/zsh",
	}

	if !reflect.DeepEqual(gotArgs[0], wantSetShell) {
		t.Fatalf(
			"first invocation args = %#v, want %#v",
			gotArgs[0],
			wantSetShell,
		)
	}

	if gotArgs[1][0] != "-S" ||
		gotArgs[1][1] != "/runtime/work/multiplexer/tmux.sock" ||
		gotArgs[1][2] != "new-session" {
		t.Fatalf(
			"second invocation = %#v, want new-session on same endpoint",
			gotArgs[1],
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
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
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
		"/runtime/work/multiplexer/tmux.sock",
		"new-session",
		"-d",
		"-e",
		"PATH=/runtime/work/bin:/usr/bin",
		"-e",
		"SHELL=/bin/bash",
		"-s",
		"work",
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
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
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
			"/runtime/work/multiplexer/tmux.sock",
			"set-option",
			"-g",
			"default-shell",
			"/runtime/bin/bash",
		},
		{
			"-S",
			"/runtime/work/multiplexer/tmux.sock",
			"new-session",
			"-d",
			"--test-option",
			"-s",
			"work",
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

	wantPrefix := []string{
		"-f",
		config,
		"-S",
		session.Endpoint,
		"set-option",
		"-g",
		"default-shell",
		"/runtime/bin/bash",
	}

	if len(gotArgs) == 0 ||
		!reflect.DeepEqual(
			gotArgs[0],
			wantPrefix,
		) {
		t.Fatalf(
			"first invocation args = %#v, want %#v",
			gotArgs[0],
			wantPrefix,
		)
	}

	wantCreate := []string{
		"-f",
		config,
		"-S",
		session.Endpoint,
		"new-session",
		"-d",
	}

	if len(gotArgs) < 2 ||
		len(gotArgs[1]) < len(wantCreate) ||
		!reflect.DeepEqual(
			gotArgs[1][:len(wantCreate)],
			wantCreate,
		) {
		t.Fatalf(
			"second invocation args = %#v, want prefix %#v",
			gotArgs[1],
			wantCreate,
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
