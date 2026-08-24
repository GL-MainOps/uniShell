package bundle

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrInvalidSource = errors.New("invalid bundle source")
	ErrInvalidPath   = errors.New("invalid bundle path")
)

const archiveRoot = "."

// CreateArchive packages sourceDir into a tar archive.
//
// Paths stored in the archive are relative to sourceDir and always use
// forward slashes. Absolute paths and path traversal are rejected.
func CreateArchive(sourceDir string) ([]byte, error) {
	sourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("resolve bundle source: %w", err)
	}

	info, err := os.Stat(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("stat bundle source: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%w: source is not a directory", ErrInvalidSource)
	}

	var buffer bytes.Buffer

	writer := tar.NewWriter(&buffer)

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("calculate archive path: %w", err)
		}

		relative = filepath.ToSlash(relative)

		if relative == "." {
			relative = archiveRoot
		}

		if err := validateArchivePath(relative); err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("create archive header for %q: %w", relative, err)
		}

		header.Name = relative

		if err := writer.WriteHeader(header); err != nil {
			return fmt.Errorf("write archive header for %q: %w", relative, err)
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open bundle file %q: %w", path, err)
		}
		defer file.Close()

		if _, err := io.Copy(writer, file); err != nil {
			return fmt.Errorf("write bundle file %q: %w", path, err)
		}

		return nil
	})

	if err != nil {
		_ = writer.Close()
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize bundle archive: %w", err)
	}

	return buffer.Bytes(), nil
}

func validateArchivePath(path string) error {
	if path == "" {
		return ErrInvalidPath
	}

	if filepath.IsAbs(path) {
		return fmt.Errorf("%w: absolute path %q", ErrInvalidPath, path)
	}

	path = filepath.ToSlash(path)

	clean := filepath.ToSlash(filepath.Clean(path))

	if clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("%w: path traversal %q", ErrInvalidPath, path)
	}

	return nil
}
