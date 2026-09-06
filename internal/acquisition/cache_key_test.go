package acquisition

import "testing"

func TestCacheKeyDiffersForDifferentArtifacts(t *testing.T) {
	first := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/tool-a/tool",
		Checksum:     "0000000000000000000000000000000000000000000000000000000000000000",
	}

	second := first
	second.URL = "https://example.com/tool-b/tool"

	firstKey, err := cacheKey(first)
	if err != nil {
		t.Fatalf("cacheKey(first) error = %v", err)
	}

	secondKey, err := cacheKey(second)
	if err != nil {
		t.Fatalf("cacheKey(second) error = %v", err)
	}

	if firstKey == secondKey {
		t.Fatalf("cache keys are equal, want different keys")
	}
}

func TestCacheKeyDiffersForDifferentRevision(t *testing.T) {
	first := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/tool",
		Revision:     "commit-a",
		Checksum:     "0000000000000000000000000000000000000000000000000000000000000000",
	}

	second := first
	second.Revision = "commit-b"

	firstKey, err := cacheKey(first)
	if err != nil {
		t.Fatalf("cacheKey(first) error = %v", err)
	}

	secondKey, err := cacheKey(second)
	if err != nil {
		t.Fatalf("cacheKey(second) error = %v", err)
	}

	if firstKey == secondKey {
		t.Fatalf("cache keys are equal, want different keys")
	}
}

func TestCacheKeyDiffersForDifferentChecksum(t *testing.T) {
	first := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/tool",
		Checksum:     "sha256:first",
	}

	second := first
	second.Checksum = "sha256:second"

	firstKey, err := cacheKey(first)
	if err != nil {
		t.Fatalf("cacheKey(first) error = %v", err)
	}

	secondKey, err := cacheKey(second)
	if err != nil {
		t.Fatalf("cacheKey(second) error = %v", err)
	}

	if firstKey == secondKey {
		t.Fatalf("cache keys are equal, want different keys")
	}
}

func TestCacheKeyIsStable(t *testing.T) {
	artifact := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/tool",
		Revision:     "commit-a",
		Checksum:     "sha256:abc",
	}

	first, err := cacheKey(artifact)
	if err != nil {
		t.Fatalf("cacheKey() error = %v", err)
	}

	second, err := cacheKey(artifact)
	if err != nil {
		t.Fatalf("cacheKey() second call error = %v", err)
	}

	if first != second {
		t.Fatalf("cache keys differ: first = %q, second = %q", first, second)
	}
}
