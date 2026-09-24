package bundle

import (
	"archive/tar"
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
func ExtractArchive(reader io.Reader, destination string) error {
	destination, directories, err := prepareExtractionDestination(destination)
	if err != nil {
		return err
	}
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
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

		if err := extractEntry(tarReader, header, target, directories); err != nil {
			return fmt.Errorf(
				"extract archive entry %q: %w",
				header.Name,
				err,
			)
		}
	}

	return nil
}

func prepareExtractionDestination(destination string) (string, *directoryCache, error) {
	destination, err := filepath.Abs(destination)
	if err != nil {
		return "", nil, fmt.Errorf("resolve extraction destination: %w", err)
	}
	if err := os.MkdirAll(destination, defaultDirectoryMode); err != nil {
		if info, statErr := os.Stat(destination); statErr == nil && !info.IsDir() {
			return "", nil, fmt.Errorf("%w: extraction destination is not a directory", ErrInvalidSource)
		}
		return "", nil, fmt.Errorf("create extraction destination: %w", err)
	}
	return destination, &directoryCache{
		root:    destination,
		ensured: map[string]struct{}{destination: {}},
	}, nil
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

type directoryCache struct {
	root    string
	ensured map[string]struct{}
}

func (cache *directoryCache) ensure(target string) error {
	if _, ok := cache.ensured[target]; ok {
		return nil
	}
	relative, err := filepath.Rel(cache.root, target)
	if err != nil {
		return fmt.Errorf("calculate directory path: %w", err)
	}
	if relative == "." {
		return nil
	}

	current := cache.root
	for _, component := range splitPath(relative) {
		current = filepath.Join(current, component)
		if _, ok := cache.ensured[current]; ok {
			continue
		}

		err := os.Mkdir(current, defaultDirectoryMode)
		if err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create directory %q: %w", current, err)
		}
		if errors.Is(err, os.ErrExist) {
			info, statErr := os.Stat(current)
			if statErr != nil {
				return fmt.Errorf("inspect directory %q: %w", current, statErr)
			}
			if !info.IsDir() {
				return fmt.Errorf("path %q is not a directory", current)
			}
		}
		cache.ensured[current] = struct{}{}
	}
	return nil
}

func splitPath(path string) []string {
	var components []string
	for path != "." && path != string(filepath.Separator) {
		parent, component := filepath.Split(path)
		if component == "" {
			break
		}
		components = append(components, component)
		path = filepath.Clean(parent)
	}
	for left, right := 0, len(components)-1; left < right; left, right = left+1, right-1 {
		components[left], components[right] = components[right], components[left]
	}
	return components
}

func extractEntry(
	reader *tar.Reader,
	header *tar.Header,
	target string,
	directories *directoryCache,
) error {
	switch header.Typeflag {
	case tar.TypeDir:
		return extractDirectory(header.FileInfo().Mode().Perm(), target, directories)

	case tar.TypeReg, tar.TypeRegA:
		return extractFile(reader, header, target, directories)

	default:
		return fmt.Errorf(
			"%w: type %d",
			ErrUnsupportedEntry,
			header.Typeflag,
		)
	}
}

func extractDirectory(mode os.FileMode, target string, directories *directoryCache) error {
	if err := directories.ensure(target); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.Chmod(target, mode.Perm()); err != nil {
		return fmt.Errorf("set directory permissions: %w", err)
	}

	return nil
}

func extractFile(
	reader *tar.Reader,
	header *tar.Header,
	target string,
	directories *directoryCache,
) error {
	if header.Size < 0 {
		return fmt.Errorf("invalid file size %d", header.Size)
	}
	return extractFileContents(
		reader,
		header.Size,
		header.FileInfo().Mode().Perm(),
		target,
		directories,
	)
}

func extractFileContents(
	reader io.Reader,
	size int64,
	mode os.FileMode,
	target string,
	directories *directoryCache,
) error {
	parent := filepath.Dir(target)
	if err := directories.ensure(parent); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultFileMode)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	if _, err := io.CopyN(file, reader, size); err != nil {
		_ = file.Close()
		return fmt.Errorf("write file: %w", err)
	}
	if err := file.Chmod(mode.Perm()); err != nil {
		_ = file.Close()
		return fmt.Errorf("set file permissions: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}
