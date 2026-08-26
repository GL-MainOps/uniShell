package zellij

import (
	"os"
	"path/filepath"
	"reflect"
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
		) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil, nil
		},
	}

	wantEnv := []string{
		"PATH=/runtime/work/bin:/usr/bin",
		"SHELL=/bin/bash",
	}

	err := backend.Create(api.Session{
		NativeName: "work",
		Env:        wantEnv,
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	wantArgs := []string{
		"attach",
		"--create-background",
		"work",
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

func TestCreateWithoutSessionNameUsesNativeDefault(t *testing.T) {
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
		) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil, nil
		},
	}

	wantEnv := []string{
		"PATH=/runtime/bin:/usr/bin",
	}

	err := backend.Create(api.Session{
		Env: wantEnv,
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	wantArgs := []string{
		"attach",
		"--create-background",
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
		) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil, nil
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
		) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil, nil
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

func TestIsAliveUsesSessionName(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
			_ string,
			args []string,
			_ []string,
		) ([]byte, error) {
			want := []string{"list-sessions"}

			if !reflect.DeepEqual(args, want) {
				t.Fatalf(
					"args = %#v, want %#v",
					args,
					want,
				)
			}

			return []byte("other\nwork\n"), nil
		},
	}

	if !backend.IsAlive(api.Session{
		NativeName: "work",
	}) {
		t.Fatal("IsAlive() = false, want true")
	}
}

func TestIsAliveRejectsMissingSession(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(
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
		Run: func(
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
		) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			gotEnv = append([]string(nil), env...)
			return nil, nil
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
		) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			return nil, nil
		},
	}

	err := backend.Create(api.Session{
		NativeName: "work",
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
		"attach",
		"--create-background",
		"--test-option",
		"work",
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
		) ([]byte, error) {
			got = append([]string(nil), args...)
			return nil, nil
		},
	}

	session := api.Session{
		NativeName: "work",
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
				) ([]byte, error) {
					t.Fatal("Run() must not be called")
					return nil, nil
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
