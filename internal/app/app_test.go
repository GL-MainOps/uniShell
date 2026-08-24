package app

import (
	"path/filepath"
	"testing"
)

func TestNewUsesDefaultRuntimeRoot(t *testing.T) {
	t.Setenv("UNISHELL_RUNTIME_DIR", "")
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	want := "/var/tmp/.lesscache"

	if application.Paths.Root != want {
		t.Fatalf("runtime root = %q, want %q", application.Paths.Root, want)
	}
}

func TestNewUsesEnvironmentRuntimeRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")
	t.Setenv("UNISHELL_RUNTIME_DIR", root)
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if application.Paths.Root != root {
		t.Fatalf("runtime root = %q, want %q", application.Paths.Root, root)
	}
}

func TestNewUsesExplicitRuntimeRoot(t *testing.T) {
	explicitRoot := filepath.Join(t.TempDir(), "explicit")
	envRoot := filepath.Join(t.TempDir(), "environment")

	t.Setenv("UNISHELL_RUNTIME_DIR", envRoot)
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
		Root:    explicitRoot,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if application.Paths.Root != explicitRoot {
		t.Fatalf("runtime root = %q, want %q", application.Paths.Root, explicitRoot)
	}
}

func TestNewBuildsVersionedRuntimePaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")

	application, err := New(Options{
		Version: "1.2.3",
		Commit:  "test",
		Root:    root,
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	wantRuntime := filepath.Join(root, "runtime", "1.2.3")
	wantBin := filepath.Join(wantRuntime, "bin")
	wantConfig := filepath.Join(wantRuntime, "config")

	if application.Paths.Runtime != wantRuntime {
		t.Fatalf("runtime path = %q, want %q", application.Paths.Runtime, wantRuntime)
	}

	if application.Paths.Bin != wantBin {
		t.Fatalf("bin path = %q, want %q", application.Paths.Bin, wantBin)
	}

	if application.Paths.Config != wantConfig {
		t.Fatalf("config path = %q, want %q", application.Paths.Config, wantConfig)
	}
}

func TestNewRejectsEmptyVersion(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "test-token")
	_, err := New(Options{
		Version: "",
		Commit:  "test",
	})
	if err == nil {
		t.Fatal("New() returned nil error, want error")
	}
}

func TestNewRequiresAuthentication(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "")

	_, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})

	if err == nil {
		t.Fatal("New() returned nil error, want authentication error")
	}
}

func TestNewUsesEnvironmentAuthenticationToken(t *testing.T) {
	t.Setenv("UNISHELL_AUTH_TOKEN", "environment-token")

	application, err := New(Options{
		Version: "1.0.0",
		Commit:  "test",
	})
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if application.AuthToken != "environment-token" {
		t.Fatalf(
			"auth token = %q, want %q",
			application.AuthToken,
			"environment-token",
		)
	}
}
