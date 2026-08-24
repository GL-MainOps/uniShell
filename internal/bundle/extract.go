package bundle

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	defaultDirectoryMode os.FileMode = 0700
	defaultFileMode      os.FileMode = 0600
)

var ErrUnsupportedEntry = errors.New("unsupported archive entry")

// ExtractArchive extracts a tar archive into destination.
//
// Every archive path is validated before it is used. Extraction is
// restricted to destination and cannot escape it through absolute paths
// or path traversal.
func ExtractArchive(data []byte, destination string) error {
	destination, err := filepath.Abs(destination)
	if err != nil {
		return fmt.Errorf("resolve extraction destination: %w", err)
	}

	info, err := os.Stat(destination)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(destination, defaultDirectoryMode); err != nil {
				return fmt.Errorf("create extraction destination: %w", err)
			}
		} else {
			return fmt.Errorf("stat extraction destination: %w", err)
		}
	} else if !info.IsDir() {
		return fmt.Errorf(
			"%w: extraction destination is not a directory",
			ErrInvalidSource,
		)
	}

	reader := tar.NewReader(bytes.NewReader(data))

	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}

		if err := validateArchivePath(header.Name); err != nil {
			return fmt.Errorf("archive entry %q: %w", header.Name, err)
		}

		target, err := secureArchivePath(destination, header.Name)
		if err != nil {
			return fmt.Errorf("archive entry %q: %w", header.Name, err)
		}

		if err := extractEntry(reader, header, target); err != nil {
			return fmt.Errorf(
				"extract archive entry %q: %w",
				header.Name,
				err,
			)
		}
	}

	return nil
}

func secureArchivePath(destination, archivePath string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(archivePath))

	if clean == "." {
		return destination, nil
	}

	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("%w: absolute path", ErrInvalidPath)
	}

	target := filepath.Join(destination, clean)

	relative, err := filepath.Rel(destination, target)
	if err != nil {
		return "", fmt.Errorf("calculate relative path: %w", err)
	}

	if relative == ".." ||
		len(relative) >= 3 &&
			relative[:3] == ".."+string(filepath.Separator) {
		return "", fmt.Errorf(
			"%w: path escapes extraction destination",
			ErrInvalidPath,
		)
	}

	return target, nil
}

func extractEntry(
	reader *tar.Reader,
	header *tar.Header,
	target string,
) error {
	switch header.Typeflag {
	case tar.TypeDir:
		return extractDirectory(header, target)

	case tar.TypeReg, tar.TypeRegA:
		return extractFile(reader, header, target)

	default:
		return fmt.Errorf(
			"%w: type %d",
			ErrUnsupportedEntry,
			header.Typeflag,
		)
	}
}

func extractDirectory(header *tar.Header, target string) error {
	if err := os.MkdirAll(target, defaultDirectoryMode); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.Chmod(target, header.FileInfo().Mode().Perm()); err != nil {
		return fmt.Errorf("set directory permissions: %w", err)
	}

	return nil
}

func extractFile(
	reader *tar.Reader,
	header *tar.Header,
	target string,
) error {
	parent := filepath.Dir(target)

	if err := os.MkdirAll(parent, defaultDirectoryMode); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}

	file, err := os.OpenFile(
		target,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		defaultFileMode,
	)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	if _, err := io.Copy(file, reader); err != nil {
		_ = file.Close()
		return fmt.Errorf("write file: %w", err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	if err := os.Chmod(target, header.FileInfo().Mode().Perm()); err != nil {
		return fmt.Errorf("set file permissions: %w", err)
	}

	return nil
}
