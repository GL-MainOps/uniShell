package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	archiveCachePrefix       = "archive-v2-"
	legacyArchiveCachePrefix = "archive-v1-"
)

// ArchiveCacheResult describes the private binary archive opened for
// extraction. A cache miss builds it atomically before returning it.
type ArchiveCacheResult struct {
	File                 *os.File
	Hit                  bool
	DecompressionElapsed time.Duration
}

// OpenArchiveCache opens the cached binary archive for this runtime version
// and authenticated bundle. The cache stores entries, not extracted session
// files, so each launch still materializes a private session runtime.
//
// cacheDir should be a private directory owned by the current user. Cache files
// are created with owner-only permissions and keyed by both version and bundle
// contents, so a new version or bundle payload cannot reuse stale files.
func OpenArchiveCache(
	data []byte,
	version string,
	cacheDir string,
) (ArchiveCacheResult, error) {
	fingerprint := sha256.New()
	_, _ = io.WriteString(fingerprint, version)
	_, _ = fingerprint.Write([]byte{0})
	_, _ = fingerprint.Write(data)
	key := hex.EncodeToString(fingerprint.Sum(nil))
	cachePath := filepath.Join(cacheDir, archiveCachePrefix+key+".bin")

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return ArchiveCacheResult{}, fmt.Errorf("create archive cache: %w", err)
	}
	if err := os.Chmod(cacheDir, 0700); err != nil {
		return ArchiveCacheResult{}, fmt.Errorf("protect archive cache: %w", err)
	}

	if file, err := openPrivateCacheFile(cachePath); err == nil {
		pruneArchiveCache(cacheDir, cachePath)
		return ArchiveCacheResult{File: file, Hit: true}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(cachePath)
	}

	temp, err := os.CreateTemp(cacheDir, archiveCachePrefix+"*.tmp")
	if err != nil {
		return ArchiveCacheResult{}, fmt.Errorf("create archive cache file: %w", err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if err := temp.Chmod(0600); err != nil {
		_ = temp.Close()
		return ArchiveCacheResult{}, fmt.Errorf("protect archive cache file: %w", err)
	}

	reader, err := DecompressAuthenticatedReader(data)
	if err != nil {
		_ = temp.Close()
		return ArchiveCacheResult{}, err
	}
	timed := &archiveCacheTimedReader{reader: reader}
	copyErr := writeOpaqueArchiveCache(timed, temp)
	closeReaderErr := reader.Close()
	elapsed := timed.elapsed
	if copyErr != nil {
		_ = temp.Close()
		return ArchiveCacheResult{}, fmt.Errorf("write binary archive cache: %w", copyErr)
	}
	if closeReaderErr != nil {
		_ = temp.Close()
		return ArchiveCacheResult{}, fmt.Errorf("close decompressed archive reader: %w", closeReaderErr)
	}
	if err := temp.Close(); err != nil {
		return ArchiveCacheResult{}, fmt.Errorf("close archive cache file: %w", err)
	}
	if err := os.Rename(tempPath, cachePath); err != nil {
		// Another launcher may have populated this exact key concurrently.
		if file, openErr := openPrivateCacheFile(cachePath); openErr == nil {
			return ArchiveCacheResult{
				File:                 file,
				Hit:                  true,
				DecompressionElapsed: elapsed,
			}, nil
		}
		return ArchiveCacheResult{}, fmt.Errorf("publish archive cache: %w", err)
	}

	file, err := openPrivateCacheFile(cachePath)
	if err != nil {
		return ArchiveCacheResult{}, fmt.Errorf("open published archive cache: %w", err)
	}
	pruneArchiveCache(cacheDir, cachePath)
	return ArchiveCacheResult{
		File:                 file,
		DecompressionElapsed: elapsed,
	}, nil
}

type archiveCacheTimedReader struct {
	reader  io.Reader
	elapsed time.Duration
}

func (reader *archiveCacheTimedReader) Read(buffer []byte) (int, error) {
	started := time.Now()
	n, err := reader.reader.Read(buffer)
	reader.elapsed += time.Since(started)
	return n, err
}

func openPrivateCacheFile(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, fmt.Errorf("archive cache is not a regular file")
	}
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func pruneArchiveCache(cacheDir, currentPath string) {
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		currentFormat := strings.HasPrefix(name, archiveCachePrefix) && strings.HasSuffix(name, ".bin")
		legacyFormat := strings.HasPrefix(name, legacyArchiveCachePrefix) && strings.HasSuffix(name, ".tar")
		if entry.IsDir() || !currentFormat && !legacyFormat {
			continue
		}
		path := filepath.Join(cacheDir, name)
		if path == currentPath {
			continue
		}
		_ = os.Remove(path)
	}
}
