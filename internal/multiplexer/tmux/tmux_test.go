package tmux

import (
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer"
)

func TestCreateUsesSessionName(t *testing.T) {
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
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := backend.Create(multiplexer.Session{}); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"new-session",
		"-d",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestAttachUsesSessionName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := backend.Attach(multiplexer.Session{Name: "work"}); err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	want := []string{
		"attach-session",
		"-t",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestIsAliveReturnsTrueWhenCommandSucceeds(t *testing.T) {
	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, _ ...string) error {
			return nil
		},
	}

	if !backend.IsAlive(multiplexer.Session{Name: "work"}) {
		t.Fatal("IsAlive() = false, want true")
	}
}

func TestIsAliveReturnsFalseWhenCommandFails(t *testing.T) {
	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, _ ...string) error {
			return errFakeFailure
		},
	}

	if backend.IsAlive(multiplexer.Session{Name: "work"}) {
		t.Fatal("IsAlive() = true, want false")
	}
}

func TestDestroyUsesSessionName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-tmux",
		Run: func(_ string, args ...string) error {
			gotArgs = append([]string(nil), args...)
			return nil
		},
	}

	if err := backend.Destroy(multiplexer.Session{Name: "work"}); err != nil {
		t.Fatalf("Destroy() returned error: %v", err)
	}

	want := []string{
		"kill-session",
		"-t",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

var errFakeFailure = &fakeError{}

type fakeError struct{}

func (*fakeError) Error() string {
	return "fake command failure"
}
