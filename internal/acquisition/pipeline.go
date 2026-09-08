package acquisition

import (
	"context"
	"fmt"
)

type Pipeline struct {
	Acquirer  Acquirer
	Stager    Stager
	Validator ArtifactValidator
}

func NewPipeline(
	acquirer Acquirer,
	stager Stager,
	validator ArtifactValidator,
) Pipeline {
	return Pipeline{
		Acquirer:  acquirer,
		Stager:    stager,
		Validator: validator,
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

	if p.Validator == nil {
		return StagedArtifact{}, fmt.Errorf(
			"validate staged artifact: validator is nil",
		)
	}

	if err := p.Validator.Validate(
		ctx,
		staged,
		artifact.Validation,
	); err != nil {
		return StagedArtifact{}, fmt.Errorf(
			"validate staged artifact: %w",
			err,
		)
	}

	return staged, nil
}
