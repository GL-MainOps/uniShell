package acquisition

import (
	"context"
	"io"
	"strings"
	"testing"
)

type testDownloader struct{}

func (testDownloader) Download(_ context.Context, _ ResolvedArtifact, dst io.Writer) error {
	_, err := io.WriteString(dst, "downloaded")
	return err
}

type testCache struct {
	data string
}

func (c *testCache) Get(_ context.Context, _ ResolvedArtifact) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(c.data)), nil
}

func (c *testCache) Put(_ context.Context, _ ResolvedArtifact, src io.Reader) error {
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	c.data = string(data)
	return nil
}

func TestDownloaderContract(t *testing.T) {
	var downloader Downloader = testDownloader{}

	var dst strings.Builder
	err := downloader.Download(context.Background(), ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     "linux",
		Architecture: "amd64",
		URL:          "https://example.com/tool",
	}, &dst)
	if err != nil {
		t.Fatalf("unexpected download error: %v", err)
	}

	if got, want := dst.String(), "downloaded"; got != want {
		t.Fatalf("downloaded data = %q, want %q", got, want)
	}
}

func TestCacheContract(t *testing.T) {
	var cache Cache = &testCache{}

	artifact := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     "linux",
		Architecture: "amd64",
		URL:          "https://example.com/tool",
	}

	if err := cache.Put(
		context.Background(),
		artifact,
		strings.NewReader("cached"),
	); err != nil {
		t.Fatalf("unexpected cache put error: %v", err)
	}

	reader, err := cache.Get(context.Background(), artifact)
	if err != nil {
		t.Fatalf("unexpected cache get error: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected cache read error: %v", err)
	}

	if got, want := string(data), "cached"; got != want {
		t.Fatalf("cached data = %q, want %q", got, want)
	}
}
