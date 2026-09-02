package zellij

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

func TestCreateUsesBackgroundSession(t *testing.T) {
	var (
		gotArgs []string
		gotEnv  []string
	)

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			env []string,
		) error {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil
		},
	}

	wantEnv := []string{
		"PATH=/runtime/work/bin:/usr/bin",
		"SHELL=/bin/bash",
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/bash",
		Env:        wantEnv,
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	wantArgs := []string{
		"--config",
		"/tmp/uHome/.config/zellij/config.kdl",
		"attach",
		"--create-background",
		"--close-on-exit",
		"work",
		"--",
		"/runtime/bin/bash",
	}

	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			wantArgs,
		)
	}

	if !reflect.DeepEqual(gotEnv, wantEnv) {
		t.Fatalf(
			"env = %#v, want %#v",
			gotEnv,
			wantEnv,
		)
	}
}

func TestCreateUsesCloseOnExit(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			_ []string,
		) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/bash",
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"--config",
		"/tmp/uHome/.config/zellij/config.kdl",
		"attach",
		"--create-background",
		"--close-on-exit",
		"work",
		"--",
		"/runtime/bin/bash",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			want,
		)
	}
}

func TestCreateUsesSessionShellPath(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			_ []string,
		) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/zsh",
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"--config",
		"/tmp/uHome/.config/zellij/config.kdl",
		"attach",
		"--create-background",
		"--close-on-exit",
		"work",
		"--",
		"/runtime/bin/zsh",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			want,
		)
	}
}

func TestCreateUsesSessionShellArgs(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			_ []string,
		) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/bash",
		ShellArgs: []string{
			"--noprofile",
			"--rcfile",
			"/runtime/config/shell-generated/work.bash",
		},
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"--config",
		"/tmp/uHome/.config/zellij/config.kdl",
		"attach",
		"--create-background",
		"--close-on-exit",
		"work",
		"--",
		"/runtime/bin/bash",
		"--noprofile",
		"--rcfile",
		"/runtime/config/shell-generated/work.bash",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf(
			"zellij invocation args = %#v, want %#v",
			gotArgs,
			want,
		)
	}
}

func TestCreateRejectsEmptyShellPath(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			_ []string,
			_ []string,
		) error {
			t.Fatal("Run() must not be called")
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
	})
	if err == nil {
		t.Fatal("Create() returned nil error, want empty shell path error")
	}

	if err.Error() != "zellij shell path cannot be empty" {
		t.Fatalf(
			"Create() error = %q, want %q",
			err.Error(),
			"zellij shell path cannot be empty",
		)
	}
}

func TestAttachUsesSessionName(t *testing.T) {
	var (
		gotArgs []string
		gotEnv  []string
	)

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			env []string,
		) error {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil
		},
	}

	err := backend.Attach(api.Session{
		NativeName: "work",
		Env: []string{
			"SHOULD_NOT_BE_USED=value",
		},
	})
	if err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	wantArgs := []string{
		"attach",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			wantArgs,
		)
	}

	if gotEnv != nil {
		t.Fatalf(
			"Attach() env = %#v, want nil",
			gotEnv,
		)
	}
}

func TestDetachDoesNotUseSessionEnvironment(t *testing.T) {
	var (
		gotArgs []string
		gotEnv  []string
	)

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			env []string,
		) error {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil
		},
	}

	err := backend.Detach(api.Session{
		NativeName: "work",
		Env: []string{
			"SHOULD_NOT_BE_USED=value",
		},
	})
	if err != nil {
		t.Fatalf("Detach() returned error: %v", err)
	}

	wantArgs := []string{
		"action",
		"detach",
	}

	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			wantArgs,
		)
	}

	if gotEnv != nil {
		t.Fatalf(
			"Detach() env = %#v, want nil",
			gotEnv,
		)
	}
}

func TestIsAliveUsesQuietRunner(t *testing.T) {
	called := false

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			_ []string,
			_ []string,
		) error {
			t.Fatal("Run() must not be called by IsAlive()")
			return nil
		},
		RunQuiet: func(
			_ string,
			args []string,
			_ []string,
		) ([]byte, error) {
			called = true

			want := []string{"list-sessions", "--short"}

			if !reflect.DeepEqual(args, want) {
				t.Fatalf(
					"args = %#v, want %#v",
					args,
					want,
				)
			}

			return []byte(
				"other [Created 2h 0m ago]\n" +
					"work [Created 0s ago]\n",
			), nil
		},
	}

	if !backend.IsAlive(api.Session{
		NativeName: "work",
	}) {
		t.Fatal("IsAlive() = false, want true")
	}

	if !called {
		t.Fatal("RunQuiet() was not called")
	}
}

func TestIsAliveDoesNotMatchSessionNamePrefix(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		RunQuiet: func(
			_ string,
			_ []string,
			_ []string,
		) ([]byte, error) {
			return []byte(
				"work-other [Created 0s ago]\n",
			), nil
		},
	}

	if backend.IsAlive(api.Session{
		NativeName: "work",
	}) {
		t.Fatal("IsAlive() = true, want false")
	}
}

func TestIsAliveRejectsMissingSession(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		RunQuiet: func(
			_ string,
			_ []string,
			_ []string,
		) ([]byte, error) {
			return []byte("other\n"), nil
		},
	}

	if backend.IsAlive(api.Session{
		NativeName: "work",
	}) {
		t.Fatal("IsAlive() = true, want false")
	}
}

func TestIsAliveWithoutSessionNameDetectsAnySession(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		RunQuiet: func(
			_ string,
			_ []string,
			_ []string,
		) ([]byte, error) {
			return []byte("work\n"), nil
		},
	}

	if !backend.IsAlive(api.Session{}) {
		t.Fatal("IsAlive() = false, want true")
	}
}

func TestDestroyUsesSessionName(t *testing.T) {
	var (
		gotArgs []string
		gotEnv  []string
	)

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			env []string,
		) error {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil
		},
	}

	err := backend.Destroy(api.Session{
		NativeName: "work",
		Env: []string{
			"SHOULD_NOT_BE_USED=value",
		},
	})
	if err != nil {
		t.Fatalf("Destroy() returned error: %v", err)
	}

	wantArgs := []string{
		"delete-session",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			wantArgs,
		)
	}

	if gotEnv != nil {
		t.Fatalf(
			"Destroy() env = %#v, want nil",
			gotEnv,
		)
	}
}

func TestCreateUsesConfiguredOptions(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			_ []string,
		) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/bash",
		Options: api.Options{
			Zellij: api.ZellijOptions{
				CreateArgs: []string{
					"--test-option",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"--config",
		"/tmp/uHome/.config/zellij/config.kdl",
		"attach",
		"--create-background",
		"--close-on-exit",
		"--test-option",
		"work",
		"--",
		"/runtime/bin/bash",
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
		"zellij",
		"config.kdl",
	)

	if err := os.MkdirAll(
		filepath.Dir(config),
		0700,
	); err != nil {
		t.Fatalf("create config directory: %v", err)
	}

	if err := os.WriteFile(
		config,
		[]byte("pane_frames false\n"),
		0600,
	); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var got []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			_ []string,
		) error {
			got = append([]string(nil), args...)
			return nil
		},
	}

	session := api.Session{
		NativeName: "work",
		ShellPath:  "/runtime/bin/bash",
		Runtime:    runtime,
	}

	if err := backend.Create(session); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	wantPrefix := []string{
		"--config",
		config,
		"attach",
		"--create-background",
		"--close-on-exit",
	}

	if len(got) < len(wantPrefix) ||
		!reflect.DeepEqual(
			got[:len(wantPrefix)],
			wantPrefix,
		) {
		t.Fatalf(
			"args = %#v, want prefix %#v",
			got,
			wantPrefix,
		)
	}
}

func TestCreateRejectsLifecycleOptions(t *testing.T) {
	tests := []string{
		"--session-name",
		"--attach-to-session",
		"--config",
		"--config-dir",
	}

	for _, option := range tests {
		t.Run(option, func(t *testing.T) {
			backend := &Backend{
				Binary: "fake-zellij",
				Run: func(
					_ string,
					_ []string,
					_ []string,
				) error {
					t.Fatal("Run() must not be called")
					return nil
				},
			}

			err := backend.Create(api.Session{
				NativeName: "work",
				Runtime:    t.TempDir(),
				Options: api.Options{
					Zellij: api.ZellijOptions{
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

func TestCreateWithNativeNamePreservesExplicitName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			_ []string,
		) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
		RunQuiet: func(
			_ string,
			_ []string,
			_ []string,
		) ([]byte, error) {
			t.Fatal(
				"RunQuiet() must not be called for explicit native name",
			)
			return nil, nil
		},
	}

	got, err := backend.CreateWithNativeName(api.Session{
		NativeName: "work",
		ShellPath:  "/bin/bash",
	})
	if err != nil {
		t.Fatalf(
			"CreateWithNativeName() returned error: %v",
			err,
		)
	}

	if got != "work" {
		t.Fatalf(
			"native name = %q, want %q",
			got,
			"work",
		)
	}

	want := []string{
		"--config",
		"/tmp/uHome/.config/zellij/config.kdl",
		"attach",
		"--create-background",
		"--close-on-exit",
		"work",
		"--",
		"/bin/bash",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf(
			"args = %#v, want %#v",
			gotArgs,
			want,
		)
	}
}

func TestCreateWithNativeNameGeneratesNativeName(
	t *testing.T,
) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			_ []string,
		) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	got, err := backend.CreateWithNativeName(api.Session{
		ShellPath: "/bin/bash",
	})
	if err != nil {
		t.Fatalf(
			"CreateWithNativeName() returned error: %v",
			err,
		)
	}

	if !strings.HasPrefix(got, "unishell-") {
		t.Fatalf(
			"native name = %q, want unishell- prefix",
			got,
		)
	}

	wantPrefix := []string{
		"--config",
		"/tmp/uHome/.config/zellij/config.kdl",
		"attach",
		"--create-background",
		"--close-on-exit",
	}

	if len(gotArgs) < len(wantPrefix)+3 {
		t.Fatalf(
			"args = %#v, want generated session name and shell",
			gotArgs,
		)
	}

	if !reflect.DeepEqual(
		gotArgs[:len(wantPrefix)],
		wantPrefix,
	) {
		t.Fatalf(
			"args prefix = %#v, want %#v",
			gotArgs[:len(wantPrefix)],
			wantPrefix,
		)
	}

	if gotArgs[5] != got {
		t.Fatalf(
			"session name argument = %q, want %q",
			gotArgs[5],
			got,
		)
	}

	if !reflect.DeepEqual(
		gotArgs[6:],
		[]string{"--", "/bin/bash"},
	) {
		t.Fatalf(
			"shell args = %#v, want %#v",
			gotArgs[6:],
			[]string{"--", "/bin/bash"},
		)
	}
}

func TestCreateWithNativeNameFailsWhenCreationFails(
	t *testing.T,
) {
	var runQuietCalled bool

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			_ []string,
			_ []string,
		) error {
			return errors.New("create failed")
		},
		RunQuiet: func(
			_ string,
			_ []string,
			_ []string,
		) ([]byte, error) {
			runQuietCalled = true
			return nil, nil
		},
	}

	_, err := backend.CreateWithNativeName(
		api.Session{
			ShellPath: "/bin/bash",
		},
	)
	if err == nil {
		t.Fatal(
			"CreateWithNativeName() returned nil error",
		)
	}

	if runQuietCalled {
		t.Fatal(
			"RunQuiet() must not be called after failed creation",
		)
	}
}
