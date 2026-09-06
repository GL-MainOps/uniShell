package acquisition

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestFilesystemCachePutAndGet(t *testing.T) {
	cache := NewFilesystemCache(t.TempDir())

	artifact := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/tool",
	}

	const want = "cached artifact"

	if err := cache.Put(
		context.Background(),
		artifact,
		strings.NewReader(want),
	); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	reader, err := cache.Get(context.Background(), artifact)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(got) != want {
		t.Fatalf("cached content = %q, want %q", got, want)
	}
}

func TestFilesystemCacheGetMissing(t *testing.T) {
	cache := NewFilesystemCache(t.TempDir())

	artifact := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/missing",
	}

	_, err := cache.Get(context.Background(), artifact)
	if !errors.Is(err, ErrCacheMiss) {
		t.Fatalf("Get() error = %v, want ErrCacheMiss", err)
	}
}

func TestFilesystemCacheRejectsNilReader(t *testing.T) {
	cache := NewFilesystemCache(t.TempDir())

	artifact := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/tool",
	}

	err := cache.Put(context.Background(), artifact, nil)
	if !errors.Is(err, ErrCacheFailed) {
		t.Fatalf("Put() error = %v, want ErrCacheFailed", err)
	}
}

func TestFilesystemCacheHonorsContextCancellation(t *testing.T) {
	cache := NewFilesystemCache(t.TempDir())

	artifact := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/tool",
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := cache.Put(ctx, artifact, strings.NewReader("data"))
	if !errors.Is(err, ErrCacheFailed) {
		t.Fatalf("Put() error = %v, want ErrCacheFailed", err)
	}

	_, err = cache.Get(ctx, artifact)
	if !errors.Is(err, ErrCacheFailed) {
		t.Fatalf("Get() error = %v, want ErrCacheFailed", err)
	}
}
