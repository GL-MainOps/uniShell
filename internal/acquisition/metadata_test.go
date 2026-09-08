package acquisition

import (
	"strings"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	const document = `
[[tools]]
name = "zellij"

[[tools.artifacts]]
version = "latest"
platform = "linux"
architecture = "amd64"
archive_type = "tar.gz"
binary_path = "zellij-x86_64-unknown-linux-musl/zellij"
binary_name = "zellij"
checksum = "sha256:example"

[tools.artifacts.validation]
static_elf = true
musl = true
executable = true

[tools.artifacts.source]
kind = "github-release"

[tools.artifacts.source.github_release]
owner = "zellij-org"
repository = "zellij"
release = "latest"
asset = "zellij-x86_64-unknown-linux-musl.tar.gz"
`

	manifest, err := LoadManifest(strings.NewReader(document))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(manifest.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(manifest.Tools))
	}

	tools, err := manifest.BuildTools()
	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected 1 converted tool, got %d", len(tools))
	}

	tool := tools[0]

	if tool.Name != "zellij" {
		t.Fatalf("expected tool name %q, got %q", "zellij", tool.Name)
	}

	if len(tool.Artifacts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(tool.Artifacts))
	}

	artifact := tool.Artifacts[0]

	if artifact.Version != "latest" {
		t.Fatalf("expected version %q, got %q", "latest", artifact.Version)
	}

	if artifact.ArchiveType != "tar.gz" {
		t.Fatalf("expected archive type %q, got %q", "tar.gz", artifact.ArchiveType)
	}

	if artifact.BinaryPath != "zellij-x86_64-unknown-linux-musl/zellij" {
		t.Fatalf(
			"expected binary path %q, got %q",
			"zellij-x86_64-unknown-linux-musl/zellij",
			artifact.BinaryPath,
		)
	}

	if artifact.BinaryName != "zellij" {
		t.Fatalf("expected binary name %q, got %q", "zellij", artifact.BinaryName)
	}

	if artifact.Source.Kind() != SourceKindGitHubRelease {
		t.Fatalf(
			"expected source kind %q, got %q",
			SourceKindGitHubRelease,
			artifact.Source.Kind(),
		)
	}
}

func TestLoadManifestRejectsUnknownFields(t *testing.T) {
	tests := []string{
		`
[[tools]]
name = "example"
unexpected = true
`,
		`
[[tools]]
name = "example"

[[tools.artifacts]]
platform = "linux"
architecture = "amd64"

[tools.artifacts.source]
kind = "direct-url"

[tools.artifacts.source.direct_url]
url = "https://example.com/tool"
unexpected = true
`,
	}

	for _, document := range tests {
		_, err := LoadManifest(strings.NewReader(document))
		if err == nil {
			t.Fatal("expected unknown field error")
		}
	}
}

func TestSourceMetadataValidation(t *testing.T) {
	tests := []struct {
		name     string
		metadata SourceMetadata
		wantErr  bool
	}{
		{
			name: "github release",
			metadata: SourceMetadata{
				Kind: SourceKindGitHubRelease,
				GitHubRelease: &GitHubReleaseSource{
					Owner:      "owner",
					Repository: "repo",
					Release:    "latest",
					Asset:      "tool.tar.gz",
				},
			},
		},
		{
			name: "github file",
			metadata: SourceMetadata{
				Kind: SourceKindGitHubFile,
				GitHubFile: &GitHubFileSource{
					Owner:      "owner",
					Repository: "repo",
					Path:       "bin/tool",
					Ref:        "main",
				},
			},
		},
		{
			name: "direct URL",
			metadata: SourceMetadata{
				Kind: SourceKindDirectURL,
				DirectURL: &DirectURLSource{
					URL: "https://example.com/tool",
				},
			},
		},
		{
			name: "missing github release configuration",
			metadata: SourceMetadata{
				Kind: SourceKindGitHubRelease,
			},
			wantErr: true,
		},
		{
			name: "unsupported kind",
			metadata: SourceMetadata{
				Kind: SourceKind("unsupported"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestGitHubReleaseSourceMetadata(t *testing.T) {
	const document = `
[[tools]]
name = "example"

[[tools.artifacts]]
platform = "linux"
architecture = "amd64"

[tools.artifacts.source]
kind = "github-release"

[tools.artifacts.source.github_release]
owner = "example"
repository = "tool"
release = "latest"
asset = "tool-linux-amd64.tar.gz"
`

	manifest, err := LoadManifest(strings.NewReader(document))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tools, err := manifest.BuildTools()
	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}

	source, ok := tools[0].Artifacts[0].Source.(GitHubReleaseSource)
	if !ok {
		t.Fatalf("expected GitHubReleaseSource, got %T", tools[0].Artifacts[0].Source)
	}

	if source.Owner != "example" {
		t.Fatalf("expected owner %q, got %q", "example", source.Owner)
	}

	if source.Repository != "tool" {
		t.Fatalf("expected repository %q, got %q", "tool", source.Repository)
	}

	if source.Release != "latest" {
		t.Fatalf("expected release %q, got %q", "latest", source.Release)
	}

	if source.Asset != "tool-linux-amd64.tar.gz" {
		t.Fatalf("expected asset %q, got %q", "tool-linux-amd64.tar.gz", source.Asset)
	}
}

func TestSourceCredentialEnvironmentMetadata(t *testing.T) {
	const document = `
[[tools]]
name = "example"

[[tools.artifacts]]
platform = "linux"
architecture = "amd64"

[tools.artifacts.source]
kind = "github-release"

[tools.artifacts.source.github_release]
owner = "example"
repository = "tool"
release = "latest"
asset = "tool-linux-amd64.tar.gz"
credential_env = "UNISHELL_ARTIFACT_TOKEN"
`

	manifest, err := LoadManifest(strings.NewReader(document))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tools, err := manifest.BuildTools()
	if err != nil {
		t.Fatalf("unexpected conversion error: %v", err)
	}

	source, ok := tools[0].Artifacts[0].Source.(GitHubReleaseSource)
	if !ok {
		t.Fatalf("expected GitHubReleaseSource, got %T", tools[0].Artifacts[0].Source)
	}

	if source.CredentialEnv != "UNISHELL_ARTIFACT_TOKEN" {
		t.Fatalf(
			"expected credential environment variable %q, got %q",
			"UNISHELL_ARTIFACT_TOKEN",
			source.CredentialEnv,
		)
	}
}
