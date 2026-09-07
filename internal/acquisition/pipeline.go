package acquisition

import (
	"context"
	"fmt"
)

type Pipeline struct {
	Acquirer Acquirer
	Stager   Stager
}

func NewPipeline(acquirer Acquirer, stager Stager) Pipeline {
	return Pipeline{
		Acquirer: acquirer,
		Stager:   stager,
	}
}

func (p Pipeline) AcquireAndStage(
	ctx context.Context,
	artifact Artifact,
	resolved ResolvedArtifact,
	headers map[string]string,
) (StagedArtifact, error) {
	if p.Stager == nil {
		return StagedArtifact{}, fmt.Errorf("stage acquired artifact: stager is nil")
	}

	reader, err := p.Acquirer.Acquire(
		ctx,
		resolved,
		headers,
	)
	if err != nil {
		return StagedArtifact{}, fmt.Errorf(
			"acquire artifact: %w",
			err,
		)
	}
	defer reader.Close()

	staged, err := p.Stager.Stage(
		ctx,
		artifact,
		reader,
	)
	if err != nil {
		return StagedArtifact{}, fmt.Errorf(
			"stage acquired artifact: %w",
			err,
		)
	}

	return staged, nil
}
