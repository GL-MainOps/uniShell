package zellij

import (
	"reflect"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
)

func TestCreateUsesNativeSessionName(t *testing.T) {
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

	if err := backend.Create(api.Session{
		Name:       "work",
		NativeName: "native-work",
	}); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if gotName != "fake-zellij" {
		t.Fatalf("command = %q, want %q", gotName, "fake-zellij")
	}

	want := []string{
		"--session",
		"native-work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestCreatePreservesEmptyNativeSessionName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(_ string, args ...string) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			return nil, nil
		},
	}

	if err := backend.Create(api.Session{
		Name: "work",
	}); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	want := []string{
		"--session",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestAttachUsesNativeSessionName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(_ string, args ...string) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			return nil, nil
		},
	}

	if err := backend.Attach(api.Session{
		Name:       "work",
		NativeName: "native-work",
	}); err != nil {
		t.Fatalf("Attach() returned error: %v", err)
	}

	want := []string{
		"attach",
		"native-work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}

func TestIsAliveFindsNativeSession(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(_ string, _ ...string) ([]byte, error) {
			return []byte("default\nnative-work\nproduction\n"), nil
		},
	}

	if !backend.IsAlive(api.Session{
		Name:       "work",
		NativeName: "native-work",
	}) {
		t.Fatal("IsAlive() = false, want true")
	}
}

func TestIsAliveRejectsMissingNativeSession(t *testing.T) {
	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(_ string, _ ...string) ([]byte, error) {
			return []byte("default\nproduction\n"), nil
		},
	}

	if backend.IsAlive(api.Session{
		Name:       "work",
		NativeName: "native-work",
	}) {
		t.Fatal("IsAlive() = true, want false")
	}
}

func TestDestroyUsesNativeSessionName(t *testing.T) {
	var gotArgs []string

	backend := &Backend{
		Binary: "fake-zellij",
		Run: func(_ string, args ...string) ([]byte, error) {
			gotArgs = append([]string(nil), args...)
			return nil, nil
		},
	}

	if err := backend.Destroy(api.Session{
		Name:       "work",
		NativeName: "native-work",
	}); err != nil {
		t.Fatalf("Destroy() returned error: %v", err)
	}

	want := []string{
		"delete-session",
		"native-work",
	}

	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
}
