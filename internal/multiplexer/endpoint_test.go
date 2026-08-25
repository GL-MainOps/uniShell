package multiplexer

import (
	"os"
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

func TestPrepareTmuxSocketPathCreatesPrivateParent(t *testing.T) {
	socketPath := filepath.Join(
		t.TempDir(),
		"multiplexer",
		"tmux.sock",
	)

	if err := PrepareTmuxSocketPath(socketPath); err != nil {
		t.Fatalf(
			"PrepareTmuxSocketPath() returned error: %v",
			err,
		)
	}

	parent := filepath.Dir(socketPath)

	info, err := os.Stat(parent)
	if err != nil {
		t.Fatalf("stat socket parent: %v", err)
	}

	if !info.IsDir() {
		t.Fatal("socket parent is not a directory")
	}

	if info.Mode().Perm() != 0700 {
		t.Fatalf(
			"socket parent permissions = %o, want %o",
			info.Mode().Perm(),
			0700,
		)
	}
}
