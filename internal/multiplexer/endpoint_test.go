package multiplexer

import (
	"path/filepath"
	"testing"
)

func TestResolveTmuxSocketUsesRuntimeDefault(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	got, err := ResolveTmuxSocket(runtimePath, "")
	if err != nil {
		t.Fatalf("ResolveTmuxSocket() returned error: %v", err)
	}

	want := filepath.Join(
		runtimePath,
		"multiplexer",
		"tmux.sock",
	)

	if got != want {
		t.Fatalf("socket = %q, want %q", got, want)
	}
}

func TestResolveTmuxSocketUsesExplicitPath(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	explicit := filepath.Join(
		t.TempDir(),
		"custom",
		"tmux.sock",
	)

	got, err := ResolveTmuxSocket(runtimePath, explicit)
	if err != nil {
		t.Fatalf("ResolveTmuxSocket() returned error: %v", err)
	}

	if got != explicit {
		t.Fatalf("socket = %q, want %q", got, explicit)
	}
}

func TestResolveTmuxSocketRejectsEmptyRuntime(t *testing.T) {
	_, err := ResolveTmuxSocket("", "")
	if err == nil {
		t.Fatal("ResolveTmuxSocket() returned nil error")
	}
}
