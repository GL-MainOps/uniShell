package acquisition

import "context"

type CompositeValidator struct {
	Validators []ArtifactValidator
}

func NewCompositeValidator(
	validators ...ArtifactValidator,
) CompositeValidator {
	return CompositeValidator{
		Validators: validators,
	}
}

func (v CompositeValidator) Validate(
	ctx context.Context,
	staged StagedArtifact,
	requirements ValidationRequirements,
) error {
	for _, validator := range v.Validators {
		if err := validator.Validate(
			ctx,
			staged,
			requirements,
		); err != nil {
			return err
		}
	}

	return nil
}
