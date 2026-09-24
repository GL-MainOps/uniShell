package bundle

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenArchiveCacheCreatesPrivateVersionedCache(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), ".cache")
	archive := createSingleEntryArchive(
		t,
		"bin/tool",
		[]byte("first version"),
		tar.TypeReg,
	)
	compressed, err := defaultCompressor.Compress(archive)
	if err != nil {
		t.Fatalf("compress archive: %v", err)
	}

	first, err := OpenArchiveCache(compressed, "v1.0.0", cacheDir)
	if err != nil {
		t.Fatalf("OpenArchiveCache() returned error: %v", err)
	}
	defer first.File.Close()
	if first.Hit {
		t.Fatal("first cache open unexpectedly reported a hit")
	}
	assertArchiveCacheMode(t, cacheDir, first.File.Name())
	magic := make([]byte, len(opaqueArchiveMagic))
	if _, err := io.ReadFull(first.File, magic); err != nil {
		t.Fatalf("read cache format header: %v", err)
	}
	if !bytes.Equal(magic, opaqueArchiveMagic[:]) {
		t.Fatal("cache does not use the private binary format")
	}
	if _, err := first.File.Seek(0, io.SeekStart); err != nil {
		t.Fatalf("rewind cached archive: %v", err)
	}
	destination := t.TempDir()
	if err := ExtractArchiveCache(first.File, destination); err != nil {
		t.Fatalf("extract cached archive: %v", err)
	}
	assertExtractedFile(t, destination, "bin/tool", []byte("first version"), 0600)

	second, err := OpenArchiveCache(compressed, "v1.0.0", cacheDir)
	if err != nil {
		t.Fatalf("open existing archive cache: %v", err)
	}
	defer second.File.Close()
	if !second.Hit {
		t.Fatal("second cache open did not report a hit")
	}
	if second.DecompressionElapsed != 0 {
		t.Fatalf("cache hit decompression time = %s, want 0", second.DecompressionElapsed)
	}

	updatedArchive := createSingleEntryArchive(
		t,
		"bin/tool",
		[]byte("updated bundled tool"),
		tar.TypeReg,
	)
	updatedBundle, err := defaultCompressor.Compress(updatedArchive)
	if err != nil {
		t.Fatalf("compress updated archive: %v", err)
	}
	updated, err := OpenArchiveCache(updatedBundle, "v1.0.0", cacheDir)
	if err != nil {
		t.Fatalf("open cache for updated bundle: %v", err)
	}
	defer updated.File.Close()
	if updated.Hit {
		t.Fatal("changed bundle payload reused a stale cache")
	}

	newVersion, err := OpenArchiveCache(compressed, "v1.1.0", cacheDir)
	if err != nil {
		t.Fatalf("open cache for updated version: %v", err)
	}
	defer newVersion.File.Close()
	if newVersion.Hit {
		t.Fatal("new runtime version reused an old version cache")
	}
	if newVersion.File.Name() == first.File.Name() {
		t.Fatal("updated version reused the old cache path")
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("read cache directory: %v", err)
	}
	cacheFiles := 0
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".bin" {
			cacheFiles++
		}
	}
	if cacheFiles != 1 {
		t.Fatalf("cache contains %d archive files, want 1", cacheFiles)
	}
}

func assertArchiveCacheMode(t *testing.T, cacheDir, cacheFile string) {
	t.Helper()
	dirInfo, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatalf("stat cache directory: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf("cache directory mode = %04o, want 0700", got)
	}
	fileInfo, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatalf("stat cache file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0600 {
		t.Fatalf("cache file mode = %04o, want 0600", got)
	}
}
