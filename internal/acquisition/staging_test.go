package acquisition

import (
	"archive/tar"
	"archive/zip"
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
		ArchiveType:  "7z",
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

func TestFilesystemStagerStagesTgzArtifact(t *testing.T) {
	baseDir := t.TempDir()
	stager := NewFilesystemStager(baseDir)

	archiveData := createTarGz(t, map[string]string{
		"example-1.0.0-linux-amd64/README.md": "documentation",
		"example-1.0.0-linux-amd64/example":   "example-binary",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "tgz",
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
}

func TestFilesystemStagerStagesTarArtifact(t *testing.T) {
	baseDir := t.TempDir()
	stager := NewFilesystemStager(baseDir)

	archiveData := createTar(t, map[string]string{
		"example-1.0.0-linux-amd64/README.md": "documentation",
		"example-1.0.0-linux-amd64/example":   "example-binary",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "tar",
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
}

func createTar(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	tarWriter := tar.NewWriter(&buffer)

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

	return buffer.Bytes()
}

func createZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	zipWriter := zip.NewWriter(&buffer)

	for name, contents := range files {
		writer, err := zipWriter.Create(name)
		if err != nil {
			t.Fatalf("Create() returned error: %v", err)
		}

		if _, err := writer.Write([]byte(contents)); err != nil {
			t.Fatalf("Write() returned error: %v", err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		t.Fatalf("zip Close() returned error: %v", err)
	}

	return buffer.Bytes()
}

func TestFilesystemStagerStagesZipArtifact(t *testing.T) {
	baseDir := t.TempDir()
	stager := NewFilesystemStager(baseDir)

	archiveData := createZip(t, map[string]string{
		"example-1.0.0-linux-amd64/README.md": "documentation",
		"example-1.0.0-linux-amd64/example":   "example-binary",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "zip",
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

func TestFilesystemStagerSelectsBinaryPath(t *testing.T) {
	stager := NewFilesystemStager(t.TempDir())

	archiveData := createTar(t, map[string]string{
		"zellij":       "binary",
		"README":       "documentation",
		"other/helper": "helper",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "tar",
		BinaryPath:   "zellij",
		BinaryName:   "zellij",
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
		t.Fatalf("Stage() error = %v", err)
	}

	data, err := os.ReadFile(staged.BinaryPath)
	if err != nil {
		t.Fatalf("ReadFile(BinaryPath) error = %v", err)
	}

	if string(data) != "binary" {
		t.Fatalf("BinaryPath content = %q, want %q", data, "binary")
	}

	if staged.BinaryPath != filepath.Join(staged.RootPath, "zellij") {
		t.Fatalf(
			"BinaryPath = %q, want %q",
			staged.BinaryPath,
			filepath.Join(staged.RootPath, "zellij"),
		)
	}
}

func TestFilesystemStagerRejectsMissingBinaryPath(t *testing.T) {
	stager := NewFilesystemStager(t.TempDir())

	archiveData := createTar(t, map[string]string{
		"README": "documentation",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "tar",
		BinaryPath:   "zellij",
		BinaryName:   "zellij",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	_, err := stager.Stage(
		context.Background(),
		artifact,
		bytes.NewReader(archiveData),
	)
	if err == nil {
		t.Fatal("Stage() returned nil error, want missing binary path error")
	}

	if !strings.Contains(err.Error(), `staged binary path "zellij" is unavailable`) {
		t.Fatalf("Stage() error = %v, want missing binary path error", err)
	}
}

func TestFilesystemStagerRejectsMissingZipBinaryPath(t *testing.T) {
	stager := NewFilesystemStager(t.TempDir())

	archiveData := createZip(t, map[string]string{
		"README": "documentation",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "zip",
		BinaryPath:   "zellij",
		BinaryName:   "zellij",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	_, err := stager.Stage(
		context.Background(),
		artifact,
		bytes.NewReader(archiveData),
	)
	if err == nil {
		t.Fatal("Stage() returned nil error, want missing binary path error")
	}

	if !strings.Contains(err.Error(), `staged binary path "zellij" is unavailable`) {
		t.Fatalf("Stage() error = %v, want missing binary path error", err)
	}
}

func TestFilesystemStagerRejectsTarPathTraversal(t *testing.T) {
	stager := NewFilesystemStager(t.TempDir())

	archiveData := createTar(t, map[string]string{
		"../../outside": "malicious",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "tar",
		BinaryPath:   "../../outside",
		BinaryName:   "example",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	_, err := stager.Stage(
		context.Background(),
		artifact,
		bytes.NewReader(archiveData),
	)
	if err == nil {
		t.Fatal("Stage() returned nil error, want path traversal error")
	}
	if !strings.Contains(err.Error(), "escapes staging root") {
		t.Fatalf("Stage() error = %v, want path traversal error", err)
	}
}

func TestFilesystemStagerRejectsZipPathTraversal(t *testing.T) {
	stager := NewFilesystemStager(t.TempDir())

	archiveData := createZip(t, map[string]string{
		"../../outside": "malicious",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "zip",
		BinaryPath:   "../../outside",
		BinaryName:   "example",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	_, err := stager.Stage(
		context.Background(),
		artifact,
		bytes.NewReader(archiveData),
	)
	if err == nil {
		t.Fatal("Stage() returned nil error, want path traversal error")
	}
	if !strings.Contains(err.Error(), "escapes staging root") {
		t.Fatalf("Stage() error = %v, want path traversal error", err)
	}
}

func TestFilesystemStagerRejectsAbsoluteArchivePath(t *testing.T) {
	stager := NewFilesystemStager(t.TempDir())

	archiveData := createTar(t, map[string]string{
		"/outside": "malicious",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "tar",
		BinaryPath:   "/outside",
		BinaryName:   "example",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	_, err := stager.Stage(
		context.Background(),
		artifact,
		bytes.NewReader(archiveData),
	)
	if err == nil {
		t.Fatal("Stage() returned nil error, want absolute path error")
	}
	if !strings.Contains(err.Error(), "archive entry path is absolute") {
		t.Fatalf("Stage() error = %v, want absolute path error", err)
	}
}

func TestFilesystemStagerRejectsZipAbsoluteArchivePath(t *testing.T) {
	stager := NewFilesystemStager(t.TempDir())

	archiveData := createZip(t, map[string]string{
		"/outside": "malicious",
	})

	artifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		ArchiveType:  "zip",
		BinaryPath:   "/outside",
		BinaryName:   "example",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	_, err := stager.Stage(
		context.Background(),
		artifact,
		bytes.NewReader(archiveData),
	)
	if err == nil {
		t.Fatal("Stage() returned nil error, want absolute path error")
	}
	if !strings.Contains(err.Error(), "archive entry path is absolute") {
		t.Fatalf("Stage() error = %v, want absolute path error", err)
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
