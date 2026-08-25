package zellij

import (
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

func TestCreateUsesSessionName(t *testing.T) {
	var gotName string
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(name string, args ...string) ([]byte, error) {
			gotName = name
			gotArgs = append([]string(nil), args...)
			return nil, nil
		},
	}

	if err := backend.Create(
		api.Session{Name: "work"},
	); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if gotName != "fake-zellij" {
		t.Fatalf("command = %q, want %q", gotName, "fake-zellij")
	}

	want := []string{
		"--session",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestAttachUsesSessionName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(_ string, args ...string) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			return nil, nil
		},
	}

	if err := backend.Attach(
		api.Session{Name: "work"},
	); err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	want := []string{
		"attach",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestIsAliveFindsSession(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(_ string, _ ...string) ([]byte, error) {
			return []byte("default\nwork\nproduction\n"), nil
		},
	}

	if !backend.IsAlive(
		api.Session{Name: "work"},
	) {
		t.Fatal("IsAlive() = false, want true")
	}
}

func TestIsAliveRejectsMissingSession(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(_ string, _ ...string) ([]byte, error) {
			return []byte("default\nproduction\n"), nil
		},
	}

	if backend.IsAlive(
		api.Session{Name: "work"},
	) {
		t.Fatal("IsAlive() = true, want false")
	}
}

func TestDestroyUsesSessionName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(_ string, args ...string) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			return nil, nil
		},
	}

	if err := backend.Destroy(
		api.Session{Name: "work"},
	); err != nil {
		t.Fatalf("Destroy() returned error: %v", err)
	}

	want := []string{
		"delete-session",
		"work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}
