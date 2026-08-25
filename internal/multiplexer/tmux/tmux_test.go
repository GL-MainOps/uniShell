package tmux

import (
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer"
)

func TestCreateUsesSocketAndSessionName(t *testing.T) {
	var gotName string
	var gotArgs []string

	backend := &Backend{
		Binary:     "fake-tmux",
		SocketPath: "/runtime/multiplexer/tmux.sock",
		Run: func(name string, args ...string) error {
			gotName = name
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	err := backend.Create(multiplexer.Session{
		Name: "work",
	})
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if gotName != "fake-tmux" {
		t.Fatalf("command = %q, want %q", gotName, "fake-tmux")
	}

	want := []string{
		"-S",
		"/runtime/multiplexer/tmux.sock",
		"new-session",
		"-d",
		"-s",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestCreatePreservesNativeSessionNaming(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary:     "fake-tmux",
		SocketPath: "/runtime/multiplexer/tmux.sock",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := backend.Create(multiplexer.Session{}); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/multiplexer/tmux.sock",
		"new-session",
		"-d",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestAttachUsesSocketAndSessionName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary:     "fake-tmux",
		SocketPath: "/runtime/multiplexer/tmux.sock",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := backend.Attach(
		multiplexer.Session{Name: "work"},
	); err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/multiplexer/tmux.sock",
		"attach-session",
		"-t",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestIsAliveUsesSocket(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary:     "fake-tmux",
		SocketPath: "/runtime/multiplexer/tmux.sock",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if !backend.IsAlive(
		multiplexer.Session{Name: "work"},
	) {
		t.Fatal("IsAlive() = false, want true")
	}

	want := []string{
		"-S",
		"/runtime/multiplexer/tmux.sock",
		"has-session",
		"-t",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestDestroyUsesSocket(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary:     "fake-tmux",
		SocketPath: "/runtime/multiplexer/tmux.sock",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := backend.Destroy(
		multiplexer.Session{Name: "work"},
	); err != nil {
		t.Fatalf("Destroy() returned error: %v", err)
	}

	want := []string{
		"-S",
		"/runtime/multiplexer/tmux.sock",
		"kill-session",
		"-t",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestValidateRejectsEmptySocket(t *testing.T) {
	backend := &Backend{}

	if err := backend.Validate(); err == nil {
		t.Fatal("Validate() returned nil error")
	}
}

func TestValidateAcceptsSocket(t *testing.T) {
	backend := &Backend{
		SocketPath: "/runtime/multiplexer/tmux.sock",
	}

	if err := backend.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
}
