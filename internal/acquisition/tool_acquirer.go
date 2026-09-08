package acquisition

import (
	"context"
	"fmt"
)

type PipelineRunner interface {
	AcquireAndStage(
		context.Context,
		Artifact,
		ResolvedArtifact,
		map[string]string,
	) (StagedArtifact, error)
}

type ToolAcquirer struct {
	Pipeline  PipelineRunner
	Providers map[SourceKind]Provider
}

func NewToolAcquirer(
	pipeline PipelineRunner,
	providers map[SourceKind]Provider,
) ToolAcquirer {
	return ToolAcquirer{
		Pipeline:  pipeline,
		Providers: providers,
	}
}

func (a ToolAcquirer) AcquireTool(
	ctx context.Context,
	tool Tool,
	platform Platform,
	architecture Architecture,
) (StagedArtifact, error) {
	if err := tool.Validate(); err != nil {
		return StagedArtifact{}, fmt.Errorf(
			"validate tool %q: %w",
			tool.Name,
			err,
		)
	}

	if a.Pipeline == nil {
		return StagedArtifact{}, fmt.Errorf(
			"acquire tool %q: pipeline is nil",
			tool.Name,
		)
	}

	artifact, err := selectArtifact(
		tool,
		platform,
		architecture,
	)
	if err != nil {
		return StagedArtifact{}, err
	}

	sourceKind := artifact.Source.Kind()

	provider, ok := a.Providers[sourceKind]
	if !ok || provider == nil {
		return StagedArtifact{}, fmt.Errorf(
			"tool %q artifact source kind %q has no provider",
			tool.Name,
			sourceKind,
		)
	}

	resolved, err := provider.Resolve(ctx, artifact)
	if err != nil {
		return StagedArtifact{}, fmt.Errorf(
			"resolve tool %q artifact: %w",
			tool.Name,
			err,
		)
	}

	if resolved.Platform != platform {
		return StagedArtifact{}, fmt.Errorf(
			"tool %q resolved platform %q does not match requested platform %q",
			tool.Name,
			resolved.Platform,
			platform,
		)
	}

	if resolved.Architecture != architecture {
		return StagedArtifact{}, fmt.Errorf(
			"tool %q resolved architecture %q does not match requested architecture %q",
			tool.Name,
			resolved.Architecture,
			architecture,
		)
	}

	headers, err := provider.Headers(artifact)
	if err != nil {
		return StagedArtifact{}, fmt.Errorf(
			"resolve credentials for tool %q: %w",
			tool.Name,
			err,
		)
	}

	staged, err := a.Pipeline.AcquireAndStage(
		ctx,
		artifact,
		resolved,
		headers,
	)
	if err != nil {
		return StagedArtifact{}, fmt.Errorf(
			"acquire tool %q: %w",
			tool.Name,
			err,
		)
	}

	return staged, nil
}

func selectArtifact(
	tool Tool,
	platform Platform,
	architecture Architecture,
) (Artifact, error) {
	var selected Artifact
	matches := 0

	for _, artifact := range tool.Artifacts {
		if artifact.Platform != platform ||
			artifact.Architecture != architecture {
			continue
		}

		selected = artifact
		matches++
	}

	switch matches {
	case 0:
		return Artifact{}, fmt.Errorf(
			"tool %q has no artifact for platform %q and architecture %q",
			tool.Name,
			platform,
			architecture,
		)
	case 1:
		return selected, nil
	default:
		return Artifact{}, fmt.Errorf(
			"tool %q has multiple artifacts for platform %q and architecture %q",
			tool.Name,
			platform,
			architecture,
		)
	}
}
