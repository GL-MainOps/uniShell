package multiplexer

import (
	"path/filepath"
	"testing"

	"gitlab.com/mainops/uniShell/internal/multiplexer/api"
	"gitlab.com/mainops/uniShell/internal/multiplexer/tmux"
)

func TestTmuxBackendResolvesRuntimeDefaultEndpoint(t *testing.T) {
	runtimePath := filepath.Join(
		t.TempDir(),
		"runtime",
	)

	backend := tmux.New()

	got, err := backend.ResolveEndpoint(
		runtimePath,
		api.Options{},
	)
	if err != nil {
		t.Fatalf(
			"ResolveEndpoint() returned error: %v",
			err,
		)
	}

	want := filepath.Join(
		runtimePath,
		"multiplexer",
		"tmux.sock",
	)

	if got != want {
		t.Fatalf(
			"endpoint = %q, want %q",
			got,
			want,
		)
	}
}

func TestTmuxBackendRejectsEmptyRuntime(t *testing.T) {
	backend := tmux.New()

	_, err := backend.ResolveEndpoint(
		"",
		api.Options{},
	)
	if err == nil {
		t.Fatal("ResolveEndpoint() returned nil error")
	}
}
