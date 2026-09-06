package acquisition

import (
	"testing"
)

type testSource struct {
	kind SourceKind
}

func (s testSource) Kind() SourceKind {
	return s.kind
}

func TestToolValidate(t *testing.T) {
	validArtifact := Artifact{
		Platform:     "linux",
		Architecture: "amd64",
		Source: testSource{
			kind: SourceKindDirectURL,
		},
	}

	tests := []struct {
		name    string
		tool    Tool
		wantErr bool
	}{
		{
			name: "valid tool",
			tool: Tool{
				Name:      "example",
				Artifacts: []Artifact{validArtifact},
			},
		},
		{
			name: "missing name",
			tool: Tool{
				Artifacts: []Artifact{validArtifact},
			},
			wantErr: true,
		},
		{
			name: "missing artifacts",
			tool: Tool{
				Name: "example",
			},
			wantErr: true,
		},
		{
			name: "invalid artifact",
			tool: Tool{
				Name: "example",
				Artifacts: []Artifact{
					{
						Platform: "linux",
						Source: testSource{
							kind: SourceKindDirectURL,
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "nil source",
			tool: Tool{
				Name: "example",
				Artifacts: []Artifact{
					{
						Platform:     "linux",
						Architecture: "amd64",
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tool.Validate()

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestResolvedArtifactValidate(t *testing.T) {
	valid := ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     "linux",
		Architecture: "amd64",
		URL:          "https://example.com/tool",
		Revision:     "abcdef1234567890",
		Checksum:     "0123456789abcdef",
	}

	if err := valid.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}

	tests := []ResolvedArtifact{
		{
			Platform:     "linux",
			Architecture: "amd64",
		},
		{
			URL:          "https://example.com/tool",
			Architecture: "amd64",
		},
		{
			URL:      "https://example.com/tool",
			Platform: "linux",
		},
		{
			URL:          "https://example.com/tool",
			Platform:     "linux",
			Architecture: "amd64",
		},
	}

	for _, artifact := range tests {
		if err := artifact.Validate(); err == nil {
			t.Fatalf("expected validation error for %+v", artifact)
		}
	}
}

func TestSourceKinds(t *testing.T) {
	tests := []struct {
		name string
		kind SourceKind
	}{
		{
			name: "github release",
			kind: SourceKindGitHubRelease,
		},
		{
			name: "github file",
			kind: SourceKindGitHubFile,
		},
		{
			name: "direct URL",
			kind: SourceKindDirectURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := testSource{kind: tt.kind}

			if source.Kind() != tt.kind {
				t.Fatalf("expected source kind %q, got %q", tt.kind, source.Kind())
			}
		})
	}
}
