package bundle

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractArchiveCacheRejectsTraversal(t *testing.T) {
	var cache bytes.Buffer
	cache.Write(opaqueArchiveMagic[:])
	cache.WriteByte(cacheEntryFile)
	var fixed [cacheEntryHeaderSize - 1]byte
	binary.BigEndian.PutUint32(fixed[0:4], 0600)
	name := "../outside"
	binary.BigEndian.PutUint32(fixed[4:8], uint32(len(name)))
	binary.BigEndian.PutUint64(fixed[8:16], 0)
	cache.Write(fixed[:])
	cache.WriteString(name)
	cache.WriteByte(cacheEntryEnd)

	destination := t.TempDir()
	err := ExtractArchiveCache(bytes.NewReader(cache.Bytes()), destination)
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("ExtractArchiveCache() error = %v, want ErrInvalidPath", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(destination), "outside")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("traversal created outside file: %v", err)
	}
}
