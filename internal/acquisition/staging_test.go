package acquisition

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
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
		ArchiveType:  "zip",
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

func TestFilesystemStagerStagesTarGzArtifact(t *testing.T) {
	baseDir := t.TempDir()
	stager := NewFilesystemStager(baseDir)

	archiveData := createTarGz(t, map[string]string{
		"example-1.0.0-linux-amd64/README.md": "documentation",
		"example-1.0.0-linux-amd64/example":   "example-binary",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "tar.gz",
		BinaryPath:   "example-1.0.0-linux-amd64/example",
		BinaryName:   "example",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	staged, err := stager.Stage(
		context.Background(),
		artifact,
		bytes.NewReader(archiveData),
	)
	if err != nil {
		t.Fatalf("Stage() returned error: %v", err)
	}
	defer os.RemoveAll(staged.RootPath)

	if staged.BinaryName != "example" {
		t.Fatalf("BinaryName = %q, want %q", staged.BinaryName, "example")
	}

	wantPath := filepath.Join(
		staged.RootPath,
		"example-1.0.0-linux-amd64",
		"example",
	)
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

	readmePath := filepath.Join(
		staged.RootPath,
		"example-1.0.0-linux-amd64",
		"README.md",
	)
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read staged README: %v", err)
	}

	if string(readme) != "documentation" {
		t.Fatalf("staged README = %q, want %q", readme, "documentation")
	}
}

func createTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer

	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	for name, contents := range files {
		data := []byte(contents)

		header := &tar.Header{
			Name: name,
			Mode: 0700,
			Size: int64(len(data)),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader() returned error: %v", err)
		}

		if _, err := tarWriter.Write(data); err != nil {
			t.Fatalf("Write() returned error: %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tar Close() returned error: %v", err)
	}

	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzip Close() returned error: %v", err)
	}

	return buffer.Bytes()
}
