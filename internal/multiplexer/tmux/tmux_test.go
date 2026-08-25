package tmux

import (
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

func TestCreateUsesSessionEndpointAndNativeName(t *testing.T) {
	var gotName string
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(name string, args ...string) error {
			gotName = name
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Create(api.Session{
		Name:       "work",
		NativeName: "native-work",
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if gotName != "fake-tmux" {
		t.Fatalf("command = %q, want %q", gotName, "fake-tmux")
	}

	want := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"new-session",
		"-d",
		"-s",
		"native-work",
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
		Name:       "work",
		NativeName: "native-work",
	})

	if err == nil {
		t.Fatal("Create() returned nil error")
	}
}

func TestCreatePreservesEmptyNativeSessionName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := backend.Create(api.Session{
		Name:     "work",
		Endpoint: "/runtime/default/multiplexer/tmux.sock",
	}); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/default/multiplexer/tmux.sock",
		"new-session",
		"-d",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestAttachUsesSessionEndpointAndNativeName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := backend.Attach(api.Session{
		Name:       "work",
		NativeName: "native-work",
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	}); err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"attach-session",
		"-t",
		"native-work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestIsAliveUsesSessionEndpointAndNativeName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if !backend.IsAlive(api.Session{
		Name:       "work",
		NativeName: "native-work",
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	}) {
		t.Fatal("IsAlive() = false, want true")
	}

	want := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"has-session",
		"-t",
		"native-work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestDestroyUsesSessionEndpointAndNativeName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := backend.Destroy(api.Session{
		Name:       "work",
		NativeName: "native-work",
		Endpoint:   "/runtime/work/multiplexer/tmux.sock",
	}); err != nil {
		t.Fatalf("Destroy() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/work/multiplexer/tmux.sock",
		"kill-session",
		"-t",
		"native-work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}
