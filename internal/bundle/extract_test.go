package bundle

import (
	"archive/tar"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractArchive(t *testing.T) {
	source := t.TempDir()
	destination := filepath.Join(t.TempDir(), "runtime")

	if err := os.Mkdir(filepath.Join(source, "bin"), 0700); err != nil {
		t.Fatalf("create bin directory: %v", err)
	}

	if err := os.Mkdir(filepath.Join(source, "config"), 0700); err != nil {
		t.Fatalf("create config directory: %v", err)
	}

	tool := []byte("#!/bin/sh\necho test\n")
	config := []byte("test=true\n")

	if err := os.WriteFile(
		filepath.Join(source, "bin", "tool"),
		tool,
		0755,
	); err != nil {
		t.Fatalf("write tool: %v", err)
	}

	if err := os.WriteFile(
		filepath.Join(source, "config", "test.conf"),
		config,
		0600,
	); err != nil {
		t.Fatalf("write config: %v", err)
	}

	archive, err := CreateArchive(source)
	if err != nil {
		t.Fatalf("CreateArchive() returned error: %v", err)
	}

	if err := ExtractArchive(archive, destination); err != nil {
		t.Fatalf("ExtractArchive() returned error: %v", err)
	}

	assertExtractedFile(t, destination, "bin/tool", tool, 0755)
	assertExtractedFile(t, destination, "config/test.conf", config, 0600)
}

func TestExtractArchivePreservesEmptyDirectories(t *testing.T) {
	source := t.TempDir()
	destination := filepath.Join(t.TempDir(), "runtime")

	emptyDir := filepath.Join(source, "empty")

	if err := os.Mkdir(emptyDir, 0700); err != nil {
		t.Fatalf("create empty directory: %v", err)
	}

	archive, err := CreateArchive(source)
	if err != nil {
		t.Fatalf("CreateArchive() returned error: %v", err)
	}

	if err := ExtractArchive(archive, destination); err != nil {
		t.Fatalf("ExtractArchive() returned error: %v", err)
	}

	info, err := os.Stat(filepath.Join(destination, "empty"))
	if err != nil {
		t.Fatalf("stat extracted directory: %v", err)
	}

	if !info.IsDir() {
		t.Fatal("extracted empty entry is not a directory")
	}
}

func TestExtractArchiveRejectsTraversal(t *testing.T) {
	archive := createSingleEntryArchive(
		t,
		"../../outside",
		[]byte("malicious"),
		tar.TypeReg,
	)

	destination := t.TempDir()
	outside := filepath.Join(filepath.Dir(destination), "outside")

	err := ExtractArchive(archive, destination)

	if err == nil {
		t.Fatal("ExtractArchive() returned nil error")
	}

	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf(
			"ExtractArchive() error = %v, want ErrInvalidPath",
			err,
		)
	}

	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		t.Fatalf("outside file was created: %q", outside)
	}
}

func TestExtractArchiveRejectsAbsolutePath(t *testing.T) {
	archive := createSingleEntryArchive(
		t,
		"/tmp/unishell-outside",
		[]byte("malicious"),
		tar.TypeReg,
	)

	err := ExtractArchive(archive, t.TempDir())

	if err == nil {
		t.Fatal("ExtractArchive() returned nil error")
	}

	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf(
			"ExtractArchive() error = %v, want ErrInvalidPath",
			err,
		)
	}
}

func TestExtractArchiveRejectsUnsupportedEntry(t *testing.T) {
	var buffer bytes.Buffer

	writer := tar.NewWriter(&buffer)

	header := &tar.Header{
		Name:     "symlink",
		Typeflag: tar.TypeSymlink,
		Linkname: "/etc/passwd",
	}

	if err := writer.WriteHeader(header); err != nil {
		t.Fatalf("write tar header: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	err := ExtractArchive(buffer.Bytes(), t.TempDir())

	if err == nil {
		t.Fatal("ExtractArchive() returned nil error")
	}

	if !errors.Is(err, ErrUnsupportedEntry) {
		t.Fatalf(
			"ExtractArchive() error = %v, want ErrUnsupportedEntry",
			err,
		)
	}
}

func TestExtractArchiveCreatesDestination(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "nested", "runtime")

	archive := createSingleEntryArchive(
		t,
		"file",
		[]byte("test"),
		tar.TypeReg,
	)

	if err := ExtractArchive(archive, destination); err != nil {
		t.Fatalf("ExtractArchive() returned error: %v", err)
	}

	assertExtractedFile(
		t,
		destination,
		"file",
		[]byte("test"),
		0600,
	)
}

func TestSecureArchivePathRejectsEscape(t *testing.T) {
	destination := t.TempDir()

	tests := []string{
		"../outside",
		"../../outside",
		"foo/../../outside",
	}

	for _, archivePath := range tests {
		t.Run(archivePath, func(t *testing.T) {
			_, err := secureArchivePath(destination, archivePath)

			if err == nil {
				t.Fatalf(
					"secureArchivePath(%q) returned nil error",
					archivePath,
				)
			}

			if !errors.Is(err, ErrInvalidPath) {
				t.Fatalf(
					"secureArchivePath(%q) error = %v, want ErrInvalidPath",
					archivePath,
					err,
				)
			}
		})
	}
}

func createSingleEntryArchive(
	t *testing.T,
	name string,
	content []byte,
	entryType byte,
) []byte {
	t.Helper()

	var buffer bytes.Buffer

	writer := tar.NewWriter(&buffer)

	header := &tar.Header{
		Name:     name,
		Mode:     0600,
		Size:     int64(len(content)),
		Typeflag: entryType,
	}

	if err := writer.WriteHeader(header); err != nil {
		t.Fatalf("write tar header: %v", err)
	}

	if entryType == tar.TypeReg {
		if _, err := writer.Write(content); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	return buffer.Bytes()
}

func assertExtractedFile(
	t *testing.T,
	root string,
	relativePath string,
	wantContent []byte,
	wantMode os.FileMode,
) {
	t.Helper()

	path := filepath.Join(root, relativePath)

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %q: %v", relativePath, err)
	}

	if !info.Mode().IsRegular() {
		t.Fatalf("%q is not a regular file", relativePath)
	}

	if info.Mode().Perm() != wantMode.Perm() {
		t.Fatalf(
			"%q mode = %o, want %o",
			relativePath,
			info.Mode().Perm(),
			wantMode.Perm(),
		)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q: %v", relativePath, err)
	}

	if !bytes.Equal(content, wantContent) {
		t.Fatalf("content mismatch for %q", relativePath)
	}
}
