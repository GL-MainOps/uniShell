package bundle

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateArchive(t *testing.T) {
	source := t.TempDir()

	if err := os.Mkdir(filepath.Join(source, "bin"), 0700); err != nil {
		t.Fatalf("create bin directory: %v", err)
	}

	if err := os.Mkdir(filepath.Join(source, "config"), 0700); err != nil {
		t.Fatalf("create config directory: %v", err)
	}

	tool := []byte("#!/bin/sh\necho test\n")

	if err := os.WriteFile(
		filepath.Join(source, "bin", "tool"),
		tool,
		0755,
	); err != nil {
		t.Fatalf("write tool: %v", err)
	}

	config := []byte("test=true\n")

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

	files := readArchive(t, archive)

	assertArchiveFile(t, files, "bin/tool", tool, 0755)
	assertArchiveFile(t, files, "config/test.conf", config, 0600)
	assertArchiveEntry(t, files, "bin")
	assertArchiveEntry(t, files, "config")
}

func TestCreateArchivePreservesEmptyDirectories(t *testing.T) {
	source := t.TempDir()

	emptyDir := filepath.Join(source, "empty")

	if err := os.Mkdir(emptyDir, 0700); err != nil {
		t.Fatalf("create empty directory: %v", err)
	}

	archive, err := CreateArchive(source)
	if err != nil {
		t.Fatalf("CreateArchive() returned error: %v", err)
	}

	files := readArchive(t, archive)

	assertArchiveEntry(t, files, "empty")
}

func TestCreateArchiveRejectsNonDirectory(t *testing.T) {
	source := filepath.Join(t.TempDir(), "file")

	if err := os.WriteFile(source, []byte("test"), 0600); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	_, err := CreateArchive(source)

	if err == nil {
		t.Fatal("CreateArchive() returned nil error")
	}
}

func TestValidateArchivePathRejectsTraversal(t *testing.T) {
	tests := []string{
		"../etc/passwd",
		"../../etc/passwd",
		"foo/../../etc/passwd",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			if err := validateArchivePath(path); err == nil {
				t.Fatalf("validateArchivePath(%q) returned nil error", path)
			}
		})
	}
}

func TestValidateArchivePathRejectsAbsolutePath(t *testing.T) {
	if err := validateArchivePath("/etc/passwd"); err == nil {
		t.Fatal("validateArchivePath() returned nil error")
	}
}

func TestValidateArchivePathAcceptsRelativePath(t *testing.T) {
	tests := []string{
		"bin/tool",
		"config/test.conf",
		"nested/directory/file",
		".",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			if err := validateArchivePath(path); err != nil {
				t.Fatalf(
					"validateArchivePath(%q) returned error: %v",
					path,
					err,
				)
			}
		})
	}
}

type archiveEntry struct {
	mode    os.FileMode
	content []byte
	dir     bool
}

func readArchive(t *testing.T, data []byte) map[string]archiveEntry {
	t.Helper()

	reader := tar.NewReader(bytes.NewReader(data))

	entries := make(map[string]archiveEntry)

	for {
		header, err := reader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatalf("read tar archive: %v", err)
		}

		content, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("read archive entry %q: %v", header.Name, err)
		}

		entries[header.Name] = archiveEntry{
			mode:    header.FileInfo().Mode(),
			content: content,
			dir:     header.FileInfo().IsDir(),
		}
	}

	return entries
}

func assertArchiveEntry(
	t *testing.T,
	entries map[string]archiveEntry,
	name string,
) {
	t.Helper()

	entry, ok := entries[name]
	if !ok {
		t.Fatalf("archive entry %q not found", name)
	}

	if !entry.dir {
		t.Fatalf("archive entry %q is not a directory", name)
	}
}

func assertArchiveFile(
	t *testing.T,
	entries map[string]archiveEntry,
	name string,
	want []byte,
	mode os.FileMode,
) {
	t.Helper()

	entry, ok := entries[name]
	if !ok {
		t.Fatalf("archive entry %q not found", name)
	}

	if entry.dir {
		t.Fatalf("archive entry %q is a directory", name)
	}

	if !bytes.Equal(entry.content, want) {
		t.Fatalf("archive file %q content mismatch", name)
	}

	if entry.mode.Perm() != mode.Perm() {
		t.Fatalf(
			"archive file %q mode = %o, want %o",
			name,
			entry.mode.Perm(),
			mode.Perm(),
		)
	}
}
