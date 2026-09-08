package acquisition

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const toolAcquirerTestChecksum = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

type toolAcquirerTestProvider struct {
	resolved     ResolvedArtifact
	resolveErr   error
	headers      map[string]string
	headersErr   error
	resolveCalls int
	headerCalls  int
}

func (p *toolAcquirerTestProvider) Resolve(
	_ context.Context,
	_ Artifact,
) (ResolvedArtifact, error) {
	p.resolveCalls++

	if p.resolveErr != nil {
		return ResolvedArtifact{}, p.resolveErr
	}

	return p.resolved, nil
}

func (p *toolAcquirerTestProvider) Headers(
	_ Artifact,
) (map[string]string, error) {
	p.headerCalls++

	if p.headersErr != nil {
		return nil, p.headersErr
	}

	return p.headers, nil
}

type toolAcquirerTestPipeline struct {
	staged    StagedArtifact
	err       error
	artifact  Artifact
	resolved  ResolvedArtifact
	headers   map[string]string
	callCount int
}

func (p *toolAcquirerTestPipeline) AcquireAndStage(
	_ context.Context,
	artifact Artifact,
	resolved ResolvedArtifact,
	headers map[string]string,
) (StagedArtifact, error) {
	p.callCount++
	p.artifact = artifact
	p.resolved = resolved
	p.headers = headers

	if p.err != nil {
		return StagedArtifact{}, p.err
	}

	return p.staged, nil
}

func newToolAcquirerTestTool() Tool {
	return Tool{
		Name: "zellij",
		Artifacts: []Artifact{
			{
				Version:      "0.45.1",
				Platform:     "linux",
				Architecture: "amd64",
				ArchiveType:  "tar.gz",
				BinaryPath:   "zellij-0.45.1/zellij",
				BinaryName:   "zellij",
				Checksum:     toolAcquirerTestChecksum,
				Source: GitHubReleaseSource{
					Owner:      "zellij-org",
					Repository: "zellij",
					Release:    "latest",
					Asset:      "zellij-x86_64-unknown-linux-musl.tar.gz",
				},
			},
		},
	}
}

func newResolvedToolAcquirerTestArtifact() ResolvedArtifact {
	return ResolvedArtifact{
		Version:      "0.45.1",
		Platform:     "linux",
		Architecture: "amd64",
		URL:          "https://example.invalid/zellij.tar.gz",
		Revision:     "v0.45.1",
		Checksum:     toolAcquirerTestChecksum,
	}
}

func TestToolAcquirerAcquireTool(t *testing.T) {
	provider := &toolAcquirerTestProvider{
		resolved: newResolvedToolAcquirerTestArtifact(),
		headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
	}

	pipeline := &toolAcquirerTestPipeline{
		staged: StagedArtifact{
			RootPath:   "/tmp/staged",
			BinaryPath: "/tmp/staged/zellij",
			BinaryName: "zellij",
		},
	}

	acquirer := NewToolAcquirer(
		pipeline,
		map[SourceKind]Provider{
			SourceKindGitHubRelease: provider,
		},
	)

	staged, err := acquirer.AcquireTool(
		context.Background(),
		newToolAcquirerTestTool(),
		"linux",
		"amd64",
	)
	if err != nil {
		t.Fatalf("acquire tool: %v", err)
	}

	if staged.BinaryName != "zellij" {
		t.Fatalf("unexpected staged binary name: %q", staged.BinaryName)
	}

	if provider.resolveCalls != 1 {
		t.Fatalf("expected one resolve call, got %d", provider.resolveCalls)
	}

	if provider.headerCalls != 1 {
		t.Fatalf("expected one header call, got %d", provider.headerCalls)
	}

	if pipeline.callCount != 1 {
		t.Fatalf("expected one pipeline call, got %d", pipeline.callCount)
	}

	if pipeline.artifact.BinaryName != "zellij" {
		t.Fatalf(
			"unexpected pipeline artifact binary name: %q",
			pipeline.artifact.BinaryName,
		)
	}

	if pipeline.resolved.URL != "https://example.invalid/zellij.tar.gz" {
		t.Fatalf(
			"unexpected resolved URL: %q",
			pipeline.resolved.URL,
		)
	}

	if pipeline.headers["Authorization"] != "Bearer test-token" {
		t.Fatalf(
			"unexpected authorization header: %q",
			pipeline.headers["Authorization"],
		)
	}
}

func TestToolAcquirerSelectsRequestedArchitecture(t *testing.T) {
	tool := newToolAcquirerTestTool()

	tool.Artifacts = append(
		tool.Artifacts,
		Artifact{
			Version:      "0.45.1",
			Platform:     "linux",
			Architecture: "arm64",
			ArchiveType:  "tar.gz",
			BinaryPath:   "zellij-arm64/zellij",
			BinaryName:   "zellij",
			Checksum:     toolAcquirerTestChecksum,
			Source: DirectURLSource{
				URL: "https://example.invalid/zellij-arm64.tar.gz",
			},
		},
	)

	artifact, err := selectArtifact(
		tool,
		"linux",
		"arm64",
	)
	if err != nil {
		t.Fatalf("select artifact: %v", err)
	}

	if artifact.Architecture != "arm64" {
		t.Fatalf(
			"expected arm64 artifact, got %q",
			artifact.Architecture,
		)
	}
}

func TestToolAcquirerRejectsMissingArtifact(t *testing.T) {
	acquirer := NewToolAcquirer(
		&toolAcquirerTestPipeline{},
		map[SourceKind]Provider{},
	)

	_, err := acquirer.AcquireTool(
		context.Background(),
		newToolAcquirerTestTool(),
		"linux",
		"arm64",
	)
	if err == nil {
		t.Fatal("expected missing artifact error")
	}

	if !strings.Contains(
		err.Error(),
		"no artifact for platform",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolAcquirerRejectsDuplicateArtifacts(t *testing.T) {
	tool := newToolAcquirerTestTool()

	tool.Artifacts = append(
		tool.Artifacts,
		tool.Artifacts[0],
	)

	acquirer := NewToolAcquirer(
		&toolAcquirerTestPipeline{},
		map[SourceKind]Provider{},
	)

	_, err := acquirer.AcquireTool(
		context.Background(),
		tool,
		"linux",
		"amd64",
	)
	if err == nil {
		t.Fatal("expected duplicate artifact error")
	}

	if !strings.Contains(
		err.Error(),
		"multiple artifacts for platform",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolAcquirerRejectsMissingProvider(t *testing.T) {
	acquirer := NewToolAcquirer(
		&toolAcquirerTestPipeline{},
		map[SourceKind]Provider{},
	)

	_, err := acquirer.AcquireTool(
		context.Background(),
		newToolAcquirerTestTool(),
		"linux",
		"amd64",
	)
	if err == nil {
		t.Fatal("expected missing provider error")
	}

	if !strings.Contains(
		err.Error(),
		"has no provider",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolAcquirerRejectsNilPipeline(t *testing.T) {
	acquirer := NewToolAcquirer(
		nil,
		map[SourceKind]Provider{},
	)

	_, err := acquirer.AcquireTool(
		context.Background(),
		newToolAcquirerTestTool(),
		"linux",
		"amd64",
	)
	if err == nil {
		t.Fatal("expected nil pipeline error")
	}

	if !strings.Contains(err.Error(), "pipeline is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolAcquirerRejectsResolvedPlatformMismatch(t *testing.T) {
	provider := &toolAcquirerTestProvider{
		resolved: ResolvedArtifact{
			Version:      "0.45.1",
			Platform:     "darwin",
			Architecture: "amd64",
			URL:          "https://example.invalid/zellij.tar.gz",
			Checksum:     toolAcquirerTestChecksum,
		},
	}

	acquirer := NewToolAcquirer(
		&toolAcquirerTestPipeline{},
		map[SourceKind]Provider{
			SourceKindGitHubRelease: provider,
		},
	)

	_, err := acquirer.AcquireTool(
		context.Background(),
		newToolAcquirerTestTool(),
		"linux",
		"amd64",
	)
	if err == nil {
		t.Fatal("expected platform mismatch error")
	}

	if !strings.Contains(err.Error(), "resolved platform") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolAcquirerRejectsResolvedArchitectureMismatch(t *testing.T) {
	provider := &toolAcquirerTestProvider{
		resolved: ResolvedArtifact{
			Version:      "0.45.1",
			Platform:     "linux",
			Architecture: "arm64",
			URL:          "https://example.invalid/zellij.tar.gz",
			Checksum:     toolAcquirerTestChecksum,
		},
	}

	acquirer := NewToolAcquirer(
		&toolAcquirerTestPipeline{},
		map[SourceKind]Provider{
			SourceKindGitHubRelease: provider,
		},
	)

	_, err := acquirer.AcquireTool(
		context.Background(),
		newToolAcquirerTestTool(),
		"linux",
		"amd64",
	)
	if err == nil {
		t.Fatal("expected architecture mismatch error")
	}

	if !strings.Contains(err.Error(), "resolved architecture") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestToolAcquirerPropagatesResolutionFailure(t *testing.T) {
	expected := errors.New("resolution failed")

	provider := &toolAcquirerTestProvider{
		resolveErr: expected,
	}

	acquirer := NewToolAcquirer(
		&toolAcquirerTestPipeline{},
		map[SourceKind]Provider{
			SourceKindGitHubRelease: provider,
		},
	)

	_, err := acquirer.AcquireTool(
		context.Background(),
		newToolAcquirerTestTool(),
		"linux",
		"amd64",
	)
	if err == nil {
		t.Fatal("expected resolution failure")
	}

	if !errors.Is(err, expected) {
		t.Fatalf(
			"expected wrapped resolution error, got: %v",
			err,
		)
	}
}

func TestToolAcquirerPropagatesCredentialFailure(t *testing.T) {
	expected := errors.New("credential failure")

	provider := &toolAcquirerTestProvider{
		resolved:   newResolvedToolAcquirerTestArtifact(),
		headersErr: expected,
	}

	acquirer := NewToolAcquirer(
		&toolAcquirerTestPipeline{},
		map[SourceKind]Provider{
			SourceKindGitHubRelease: provider,
		},
	)

	_, err := acquirer.AcquireTool(
		context.Background(),
		newToolAcquirerTestTool(),
		"linux",
		"amd64",
	)
	if err == nil {
		t.Fatal("expected credential failure")
	}

	if !errors.Is(err, expected) {
		t.Fatalf(
			"expected wrapped credential error, got: %v",
			err,
		)
	}
}

func TestToolAcquirerPropagatesPipelineFailure(t *testing.T) {
	expected := errors.New("pipeline failure")

	provider := &toolAcquirerTestProvider{
		resolved: newResolvedToolAcquirerTestArtifact(),
	}

	pipeline := &toolAcquirerTestPipeline{
		err: expected,
	}

	acquirer := NewToolAcquirer(
		pipeline,
		map[SourceKind]Provider{
			SourceKindGitHubRelease: provider,
		},
	)

	_, err := acquirer.AcquireTool(
		context.Background(),
		newToolAcquirerTestTool(),
		"linux",
		"amd64",
	)
	if err == nil {
		t.Fatal("expected pipeline failure")
	}

	if !errors.Is(err, expected) {
		t.Fatalf(
			"expected wrapped pipeline error, got: %v",
			err,
		)
	}
}
