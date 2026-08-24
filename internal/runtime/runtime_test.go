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

	if err := paths.Prepare(); err != nil {
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
