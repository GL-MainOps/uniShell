package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPrepareCreatesRuntimeDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session := NewSession(paths)

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	for _, path := range []string{
		paths.Root,
		paths.Runtime,
		paths.Bin,
		paths.Config,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %q: %v", path, err)
		}

		if !info.IsDir() {
			t.Fatalf("%q is not a directory", path)
		}
	}
}

func TestCleanupRemovesRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	session := NewSession(paths)

	if err := session.Prepare(); err != nil {
		t.Fatalf("Prepare() returned error: %v", err)
	}

	if err := session.Cleanup(); err != nil {
		t.Fatalf("Cleanup() returned error: %v", err)
	}

	if _, err := os.Stat(paths.Runtime); !os.IsNotExist(err) {
		t.Fatalf("runtime still exists: %q", paths.Runtime)
	}
}

func TestCleanupStaleRemovesExistingRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	if err := os.MkdirAll(paths.Bin, 0700); err != nil {
		t.Fatalf("create test runtime: %v", err)
	}

	if err := CleanupStale(paths); err != nil {
		t.Fatalf("CleanupStale() returned error: %v", err)
	}

	if _, err := os.Stat(paths.Runtime); !os.IsNotExist(err) {
		t.Fatalf("stale runtime still exists: %q", paths.Runtime)
	}
}

func TestCleanupStaleIgnoresMissingRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	if err := CleanupStale(paths); err != nil {
		t.Fatalf("CleanupStale() returned error: %v", err)
	}
}

func TestIsWithinRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "unishell")

	paths, err := NewPaths(root, "1.0.0")
	if err != nil {
		t.Fatalf("NewPaths() returned error: %v", err)
	}

	tests := []struct {
		name   string
		target string
		want   bool
	}{
		{
			name:   "runtime directory",
			target: paths.Runtime,
			want:   true,
		},
		{
			name:   "binary directory",
			target: paths.Bin,
			want:   true,
		},
		{
			name:   "file inside runtime",
			target: filepath.Join(paths.Bin, "fzf"),
			want:   true,
		},
		{
			name:   "outside root",
			target: filepath.Join(root, "..", "outside"),
			want:   false,
		},
		{
			name:   "filesystem root",
			target: string(filepath.Separator),
			want:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsWithinRoot(paths, test.target); got != test.want {
				t.Fatalf(
					"IsWithinRoot(%q) = %v, want %v",
					test.target,
					got,
					test.want,
				)
			}
		})
	}
}
