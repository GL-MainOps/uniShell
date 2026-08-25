package zellij

import (
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

func TestCreateUsesSessionName(t *testing.T) {
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
		"--session",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}

	if !reflect.DeepEqual(gotEnv, wantEnv) {
		t.Fatalf("env = %#v, want %#v", gotEnv, wantEnv)
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

	err := backend.Create(api.Session{
		Env: []string{
			"PATH=/runtime/bin:/usr/bin",
		},
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	wantArgs := []string{
		"--session",
	}

	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}

	wantEnv := []string{
		"PATH=/runtime/bin:/usr/bin",
	}

	if !reflect.DeepEqual(gotEnv, wantEnv) {
		t.Fatalf("env = %#v, want %#v", gotEnv, wantEnv)
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
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
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
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
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
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}

	if gotEnv != nil {
		t.Fatalf(
			"Destroy() env = %#v, want nil",
			gotEnv,
		)
	}
}
