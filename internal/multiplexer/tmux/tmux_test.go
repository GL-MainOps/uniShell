package tmux

import (
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

func TestCreateUsesSessionEndpointAndName(t *testing.T) {
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
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"new-session",
		"-d",
		"-s",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
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
