package acquisition

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilesystemStagerStagesDirectArtifact(t *testing.T) {
	baseDir := t.TempDir()

	stager := NewFilesystemStager(baseDir)

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "",
		BinaryName:   "example",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	staged, err := stager.Stage(
		context.Background(),
		artifact,
		strings.NewReader("example-binary"),
	)
	if err != nil {
		t.Fatalf("Stage() returned error: %v", err)
	}
	defer os.RemoveAll(staged.RootPath)

	if staged.BinaryName != "example" {
		t.Fatalf("BinaryName = %q, want %q", staged.BinaryName, "example")
	}

	wantPath := filepath.Join(staged.RootPath, "example")
	if staged.BinaryPath != wantPath {
		t.Fatalf("BinaryPath = %q, want %q", staged.BinaryPath, wantPath)
	}

	data, err := os.ReadFile(staged.BinaryPath)
	if err != nil {
		t.Fatalf("read staged binary: %v", err)
	}

	if string(data) != "example-binary" {
		t.Fatalf("staged binary = %q, want %q", data, "example-binary")
	}
}

func TestFilesystemStagerRejectsArchive(t *testing.T) {
	stager := NewFilesystemStager(t.TempDir())

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "tar.gz",
		BinaryName:   "example",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	_, err := stager.Stage(
		context.Background(),
		artifact,
		nil,
	)
	if !errors.Is(err, ErrUnsupportedArchiveType) {
		t.Fatalf(
			"Stage() error = %v, want %v",
			err,
			ErrUnsupportedArchiveType,
		)
	}
}
