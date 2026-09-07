package acquisition

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

const testChecksum = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestHTTPProviderResolvesDirectURL(t *testing.T) {
	artifact := Artifact{
		Version:      "1.0.0",
		Platform:     "linux",
		Architecture: "amd64",
		BinaryName:   "example",
		Checksum:     testChecksum,
		Source: DirectURLSource{
			URL: "https://example.invalid/example.tar.gz",
		},
	}

	provider := NewHTTPProvider(nil)

	resolved, err := provider.Resolve(context.Background(), artifact)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.URL != artifact.Source.(DirectURLSource).URL {
		t.Fatalf("unexpected URL: %q", resolved.URL)
	}

	if resolved.Checksum != artifact.Checksum {
		t.Fatalf("unexpected checksum: %q", resolved.Checksum)
	}
}

func TestHTTPProviderResolvesGitHubFile(t *testing.T) {
	artifact := Artifact{
		Version:      "main",
		Platform:     "linux",
		Architecture: "amd64",
		BinaryName:   "example",
		Checksum:     testChecksum,
		Source: GitHubFileSource{
			Owner:      "example",
			Repository: "repo",
			Path:       "bin/example",
			Ref:        "main",
		},
	}

	provider := NewHTTPProvider(nil)

	resolved, err := provider.Resolve(context.Background(), artifact)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "https://raw.githubusercontent.com/example/repo/main/bin/example"
	if resolved.URL != want {
		t.Fatalf("expected URL %q, got %q", want, resolved.URL)
	}

	if resolved.Revision != "main" {
		t.Fatalf("expected revision %q, got %q", "main", resolved.Revision)
	}
}

func TestHTTPProviderResolvesGitHubRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/example/repo/releases/latest" {
			t.Fatalf("unexpected request path: %q", r.URL.Path)
		}

		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("expected authorization header, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")

		response := map[string]any{
			"tag_name": "v1.2.3",
			"assets": []map[string]string{
				{
					"name": "example-linux-amd64.tar.gz",
					"browser_download_url": "https://example.invalid/example-linux-amd64.tar.gz",
				},
			},
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	const credentialEnv = "UNISHELL_TEST_GITHUB_TOKEN"

	previous := os.Getenv(credentialEnv)
	defer os.Setenv(credentialEnv, previous)

	if err := os.Setenv(credentialEnv, "test-token"); err != nil {
		t.Fatal(err)
	}

	artifact := Artifact{
		Version:      "latest",
		Platform:     "linux",
		Architecture: "amd64",
		BinaryName:   "example",
		Checksum:     testChecksum,
		Source: GitHubReleaseSource{
			Owner:         "example",
			Repository:    "repo",
			Release:       "latest",
			Asset:         "example-linux-amd64.tar.gz",
			CredentialEnv: credentialEnv,
		},
	}

	provider := NewHTTPProvider(server.Client())
	provider.GitHubAPIBaseURL = server.URL

	resolved, err := provider.Resolve(context.Background(), artifact)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.URL != "https://example.invalid/example-linux-amd64.tar.gz" {
		t.Fatalf("unexpected resolved URL: %q", resolved.URL)
	}

	if resolved.Revision != "v1.2.3" {
		t.Fatalf("unexpected revision: %q", resolved.Revision)
	}
}

func TestHTTPProviderRequiresConfiguredCredential(t *testing.T) {
	const credentialEnv = "UNISHELL_MISSING_TOKEN"

	previous := os.Getenv(credentialEnv)
	defer os.Setenv(credentialEnv, previous)

	_ = os.Unsetenv(credentialEnv)

	artifact := Artifact{
		Version:      "1.0.0",
		Platform:     "linux",
		Architecture: "amd64",
		BinaryName:   "example",
		Checksum:     testChecksum,
		Source: DirectURLSource{
			URL:           "https://example.invalid/example",
			CredentialEnv: credentialEnv,
		},
	}

	provider := NewHTTPProvider(nil)

	if _, err := provider.Headers(artifact); err == nil {
		t.Fatal("expected missing credential error")
	}
}
