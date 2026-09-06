package acquisition

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
)

type fakeDownloader struct {
	calls   int
	content string
	err     error
}

func (d *fakeDownloader) Download(
	_ context.Context,
	_ DownloadRequest,
	dst io.Writer,
) error {
	d.calls++

	if d.err != nil {
		return d.err
	}

	_, err := io.WriteString(dst, d.content)
	return err
}

type fakeCache struct {
	getCalls int
	putCalls int

	content string
	getErr  error
	putErr  error
}

func (c *fakeCache) Get(
	_ context.Context,
	_ ResolvedArtifact,
) (io.ReadCloser, error) {
	c.getCalls++

	if c.getErr != nil {
		return nil, c.getErr
	}

	return io.NopCloser(strings.NewReader(c.content)), nil
}

func (c *fakeCache) Put(
	_ context.Context,
	_ ResolvedArtifact,
	src io.Reader,
) error {
	c.putCalls++

	if c.putErr != nil {
		return c.putErr
	}

	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}

	c.content = string(data)
	return nil
}

func testResolvedArtifact() ResolvedArtifact {
	return ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/tool",
	}
}

func TestAcquirerReturnsCachedArtifact(t *testing.T) {
	downloader := &fakeDownloader{
		content: "downloaded",
	}
	cache := &fakeCache{
		content: "cached",
	}

	acquirer := NewAcquirer(downloader, cache)

	reader, err := acquirer.Acquire(
		context.Background(),
		testResolvedArtifact(),
		nil,
	)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != "cached" {
		t.Fatalf("content = %q, want %q", data, "cached")
	}

	if cache.getCalls != 1 {
		t.Fatalf("cache get calls = %d, want 1", cache.getCalls)
	}

	if downloader.calls != 0 {
		t.Fatalf("downloader calls = %d, want 0", downloader.calls)
	}

	if cache.putCalls != 0 {
		t.Fatalf("cache put calls = %d, want 0", cache.putCalls)
	}
}

func TestAcquirerDownloadsAndCachesOnMiss(t *testing.T) {
	downloader := &fakeDownloader{
		content: "downloaded",
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}

	acquirer := NewAcquirer(downloader, cache)

	reader, err := acquirer.Acquire(
		context.Background(),
		testResolvedArtifact(),
		nil,
	)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != "downloaded" {
		t.Fatalf("content = %q, want %q", data, "downloaded")
	}

	if downloader.calls != 1 {
		t.Fatalf("downloader calls = %d, want 1", downloader.calls)
	}

	if cache.putCalls != 1 {
		t.Fatalf("cache put calls = %d, want 1", cache.putCalls)
	}

	if cache.content != "downloaded" {
		t.Fatalf("cached content = %q, want %q", cache.content, "downloaded")
	}
}

func TestAcquirerVerifiesDownloadedArtifactBeforeCaching(t *testing.T) {
	content := "downloaded artifact"
	sum := sha256.Sum256([]byte(content))
	checksum := hex.EncodeToString(sum[:])

	downloader := &fakeDownloader{
		content: content,
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}

	artifact := testResolvedArtifact()
	artifact.Checksum = checksum

	acquirer := NewAcquirer(downloader, cache)

	reader, err := acquirer.Acquire(
		context.Background(),
		artifact,
		nil,
	)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != content {
		t.Fatalf("content = %q, want %q", data, content)
	}

	if cache.putCalls != 1 {
		t.Fatalf("cache put calls = %d, want 1", cache.putCalls)
	}

	if cache.content != content {
		t.Fatalf("cached content = %q, want %q", cache.content, content)
	}
}

func TestAcquirerRejectsDownloadedChecksumMismatch(t *testing.T) {
	downloader := &fakeDownloader{
		content: "downloaded artifact",
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}

	artifact := testResolvedArtifact()
	artifact.Checksum = strings.Repeat("0", sha256.Size*2)

	acquirer := NewAcquirer(downloader, cache)

	_, err := acquirer.Acquire(
		context.Background(),
		artifact,
		nil,
	)
	if err == nil {
		t.Fatal("Acquire() error = nil, want checksum mismatch")
	}

	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf(
			"Acquire() error = %v, want ErrChecksumMismatch",
			err,
		)
	}

	if cache.putCalls != 0 {
		t.Fatalf(
			"cache put calls = %d, want 0",
			cache.putCalls,
		)
	}
}

func TestAcquirerRedownloadsCachedChecksumMismatch(t *testing.T) {
	cache := &fakeCache{
		content: "corrupted cached artifact",
	}
	downloader := &fakeDownloader{
		content: "downloaded artifact",
	}

	content := "downloaded artifact"
	sum := sha256.Sum256([]byte(content))

	artifact := testResolvedArtifact()
	artifact.Checksum = hex.EncodeToString(sum[:])

	acquirer := NewAcquirer(downloader, cache)

	reader, err := acquirer.Acquire(
		context.Background(),
		artifact,
		nil,
	)
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != content {
		t.Fatalf("content = %q, want %q", data, content)
	}

	if downloader.calls != 1 {
		t.Fatalf(
			"downloader calls = %d, want 1",
			downloader.calls,
		)
	}

	if cache.putCalls != 1 {
		t.Fatalf(
			"cache put calls = %d, want 1",
			cache.putCalls,
		)
	}

	if cache.content != content {
		t.Fatalf(
			"cached content = %q, want %q",
			cache.content,
			content,
		)
	}
}

func TestAcquirerPropagatesCacheFailure(t *testing.T) {
	downloader := &fakeDownloader{
		content: "downloaded",
	}
	cache := &fakeCache{
		getErr: errors.New("cache failure"),
	}

	acquirer := NewAcquirer(downloader, cache)

	_, err := acquirer.Acquire(
		context.Background(),
		testResolvedArtifact(),
		nil,
	)
	if err == nil {
		t.Fatal("Acquire() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "cache failure") {
		t.Fatalf("Acquire() error = %v, want cache failure", err)
	}

	if downloader.calls != 0 {
		t.Fatalf("downloader calls = %d, want 0", downloader.calls)
	}
}

func TestAcquirerPropagatesDownloadFailure(t *testing.T) {
	downloadErr := errors.New("download failure")

	downloader := &fakeDownloader{
		err: downloadErr,
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}

	acquirer := NewAcquirer(downloader, cache)

	_, err := acquirer.Acquire(
		context.Background(),
		testResolvedArtifact(),
		nil,
	)
	if !errors.Is(err, downloadErr) {
		t.Fatalf("Acquire() error = %v, want download failure", err)
	}

	if cache.putCalls != 0 {
		t.Fatalf("cache put calls = %d, want 0", cache.putCalls)
	}
}

func TestAcquirerPropagatesCachePutFailure(t *testing.T) {
	putErr := errors.New("cache put failure")

	downloader := &fakeDownloader{
		content: "downloaded",
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
		putErr: putErr,
	}

	acquirer := NewAcquirer(downloader, cache)

	_, err := acquirer.Acquire(
		context.Background(),
		testResolvedArtifact(),
		nil,
	)
	if !errors.Is(err, putErr) {
		t.Fatalf("Acquire() error = %v, want cache put failure", err)
	}

	if downloader.calls != 1 {
		t.Fatalf("downloader calls = %d, want 1", downloader.calls)
	}

	if cache.putCalls != 1 {
		t.Fatalf("cache put calls = %d, want 1", cache.putCalls)
	}
}

func TestAcquirerRejectsNilDownloader(t *testing.T) {
	cache := &fakeCache{}

	acquirer := NewAcquirer(nil, cache)

	_, err := acquirer.Acquire(
		context.Background(),
		testResolvedArtifact(),
		nil,
	)
	if !errors.Is(err, ErrDownloadFailed) {
		t.Fatalf("Acquire() error = %v, want ErrDownloadFailed", err)
	}
}

func TestAcquirerRejectsNilCache(t *testing.T) {
	downloader := &fakeDownloader{}

	acquirer := NewAcquirer(downloader, nil)

	_, err := acquirer.Acquire(
		context.Background(),
		testResolvedArtifact(),
		nil,
	)
	if !errors.Is(err, ErrCacheFailed) {
		t.Fatalf("Acquire() error = %v, want ErrCacheFailed", err)
	}
}
