package bundle

import (
	"archive/tar"
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

var opaqueArchiveMagic = [8]byte{'U', 'S', 'C', 'A', 'C', 'H', 'E', 2}

const (
	cacheEntryEnd byte = iota
	cacheEntryDirectory
	cacheEntryFile
	cacheEntryHeaderSize = 17
	maxCachePathSize     = 1 << 20
)

// writeOpaqueArchiveCache stores tar entries in a private length-prefixed
// format. The format is not encryption; filesystem permissions protect it
// from other OS accounts.
func writeOpaqueArchiveCache(source io.Reader, output io.Writer) error {
	if _, err := output.Write(opaqueArchiveMagic[:]); err != nil {
		return err
	}
	archive := tar.NewReader(source)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read source tar archive: %w", err)
		}
		if err := validateArchivePath(header.Name); err != nil {
			return fmt.Errorf("archive entry %q: %w", header.Name, err)
		}
		name := filepath.ToSlash(header.Name)
		if len(name) == 0 || len(name) > maxCachePathSize {
			return fmt.Errorf("archive entry path length %d is invalid", len(name))
		}
		entryType := byte(cacheEntryFile)
		switch header.Typeflag {
		case tar.TypeDir:
			entryType = cacheEntryDirectory
			if header.Size != 0 {
				return fmt.Errorf("directory entry %q has data", name)
			}
		case tar.TypeReg, tar.TypeRegA:
			if header.Size < 0 {
				return fmt.Errorf("archive entry %q has invalid size", name)
			}
		default:
			return fmt.Errorf("%w: type %d", ErrUnsupportedEntry, header.Typeflag)
		}

		var fixed [cacheEntryHeaderSize]byte
		fixed[0] = entryType
		binary.BigEndian.PutUint32(fixed[1:5], uint32(header.FileInfo().Mode().Perm()))
		binary.BigEndian.PutUint32(fixed[5:9], uint32(len(name)))
		binary.BigEndian.PutUint64(fixed[9:17], uint64(header.Size))
		if _, err := output.Write(fixed[:]); err != nil {
			return err
		}
		if _, err := io.WriteString(output, name); err != nil {
			return err
		}
		if entryType == cacheEntryFile {
			if _, err := io.CopyN(output, archive, header.Size); err != nil {
				return fmt.Errorf("cache archive entry %q: %w", name, err)
			}
		}
	}
	if _, err := output.Write([]byte{cacheEntryEnd}); err != nil {
		return err
	}
	if _, err := io.Copy(io.Discard, source); err != nil {
		return fmt.Errorf("finish source archive: %w", err)
	}
	return nil
}

// ExtractArchiveCache expands the private length-prefixed cache into a fresh
// session directory.
func ExtractArchiveCache(reader io.Reader, destination string) error {
	destination, directories, err := prepareExtractionDestination(destination)
	if err != nil {
		return err
	}
	input := bufio.NewReader(reader)
	var magic [len(opaqueArchiveMagic)]byte
	if _, err := io.ReadFull(input, magic[:]); err != nil {
		return fmt.Errorf("read archive cache header: %w", err)
	}
	if magic != opaqueArchiveMagic {
		return errors.New("invalid archive cache format")
	}

	for {
		entryType, err := input.ReadByte()
		if err != nil {
			return fmt.Errorf("read archive cache entry: %w", err)
		}
		if entryType == cacheEntryEnd {
			if _, err := input.ReadByte(); err != io.EOF {
				if err == nil {
					return errors.New("trailing data after archive cache end")
				}
				return fmt.Errorf("check archive cache end: %w", err)
			}
			return nil
		}
		if entryType != cacheEntryDirectory && entryType != cacheEntryFile {
			return fmt.Errorf("unsupported archive cache entry type %d", entryType)
		}
		var fixed [cacheEntryHeaderSize - 1]byte
		if _, err := io.ReadFull(input, fixed[:]); err != nil {
			return fmt.Errorf("read archive cache entry header: %w", err)
		}
		mode := os.FileMode(binary.BigEndian.Uint32(fixed[:4]))
		nameSize := binary.BigEndian.Uint32(fixed[4:8])
		contentSize := binary.BigEndian.Uint64(fixed[8:16])
		if nameSize == 0 || nameSize > maxCachePathSize {
			return fmt.Errorf("invalid archive cache path length %d", nameSize)
		}
		if contentSize > math.MaxInt64 {
			return errors.New("archive cache entry is too large")
		}
		if entryType == cacheEntryDirectory && contentSize != 0 {
			return errors.New("archive cache directory entry has data")
		}
		nameBytes := make([]byte, int(nameSize))
		if _, err := io.ReadFull(input, nameBytes); err != nil {
			return fmt.Errorf("read archive cache path: %w", err)
		}
		name := string(nameBytes)
		if err := validateArchivePath(name); err != nil {
			return fmt.Errorf("archive cache entry %q: %w", name, err)
		}
		target, err := secureArchivePath(destination, name)
		if err != nil {
			return fmt.Errorf("archive cache entry %q: %w", name, err)
		}
		if entryType == cacheEntryDirectory {
			err = extractDirectory(mode, target, directories)
		} else {
			err = extractFileContents(input, int64(contentSize), mode, target, directories)
		}
		if err != nil {
			return fmt.Errorf("extract archive cache entry %q: %w", name, err)
		}
	}
}
