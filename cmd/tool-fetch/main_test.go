package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/mainops/uniShell/internal/acquisition"
)

func TestInstallBinary(t *testing.T) {
	dir := t.TempDir()

	source := filepath.Join(dir, "source")
	destination := filepath.Join(dir, "nested", "example")

	if err := os.WriteFile(
		source,
		[]byte("binary"),
		0644,
	); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := installBinary(source, destination); err != nil {
		t.Fatalf("installBinary() error = %v", err)
	}

	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}

	if string(data) != "binary" {
		t.Fatalf(
			"destination contents = %q, want %q",
			string(data),
			"binary",
		)
	}

	info, err := os.Stat(destination)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}

	if info.Mode().Perm() != 0755 {
		t.Fatalf(
			"destination permissions = %o, want %o",
			info.Mode().Perm(),
			0755,
		)
	}
}

func TestRunRejectsMissingToolDirectory(t *testing.T) {
	root := t.TempDir()

	err := run(
		context.Background(),
		filepath.Join(root, "missing-tools"),
		filepath.Join(root, "bin"),
		filepath.Join(root, "cache"),
		acquisition.Platform("linux"),
		acquisition.Architecture("amd64"),
		nil,
	)

	if err == nil {
		t.Fatal("run() error = nil, want missing-tool-directory error")
	}
}
