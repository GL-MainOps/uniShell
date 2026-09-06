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

func TestFilesystemCacheSeparatesArtifactsWithSameURLBasename(t *testing.T) {
	cache := NewFilesystemCache(t.TempDir())

	first := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/releases/tool",
	}

	second := first
	second.URL = "https://mirror.example.com/releases/tool"

	if err := cache.Put(
		context.Background(),
		first,
		strings.NewReader("first artifact"),
	); err != nil {
		t.Fatalf("Put(first) error = %v", err)
	}

	if err := cache.Put(
		context.Background(),
		second,
		strings.NewReader("second artifact"),
	); err != nil {
		t.Fatalf("Put(second) error = %v", err)
	}

	firstReader, err := cache.Get(context.Background(), first)
	if err != nil {
		t.Fatalf("Get(first) error = %v", err)
	}
	defer firstReader.Close()

	firstData, err := io.ReadAll(firstReader)
	if err != nil {
		t.Fatalf("ReadAll(first) error = %v", err)
	}

	secondReader, err := cache.Get(context.Background(), second)
	if err != nil {
		t.Fatalf("Get(second) error = %v", err)
	}
	defer secondReader.Close()

	secondData, err := io.ReadAll(secondReader)
	if err != nil {
		t.Fatalf("ReadAll(second) error = %v", err)
	}

	if string(firstData) != "first artifact" {
		t.Fatalf("first cached content = %q, want %q", firstData, "first artifact")
	}

	if string(secondData) != "second artifact" {
		t.Fatalf("second cached content = %q, want %q", secondData, "second artifact")
	}
}
