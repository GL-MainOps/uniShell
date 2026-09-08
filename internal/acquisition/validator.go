package acquisition

import "context"

type ArtifactValidator interface {
	Validate(context.Context, StagedArtifact, ValidationRequirements) error
}
