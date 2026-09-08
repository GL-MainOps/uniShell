package acquisition

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type compositeValidatorStub struct {
	name         string
	calls        *[]string
	requirements *ValidationRequirements
	err          error
}

func (v compositeValidatorStub) Validate(
	_ context.Context,
	_ StagedArtifact,
	requirements ValidationRequirements,
) error {
	*v.calls = append(*v.calls, v.name)

	if v.requirements != nil {
		*v.requirements = requirements
	}

	return v.err
}

func TestCompositeValidatorRunsValidatorsInOrder(t *testing.T) {
	var calls []string

	requirements := ValidationRequirements{
		StaticELF:  true,
		Musl:       true,
		Executable: true,
	}

	var firstRequirements ValidationRequirements
	var secondRequirements ValidationRequirements

	validator := NewCompositeValidator(
		compositeValidatorStub{
			name:         "static-elf",
			calls:        &calls,
			requirements: &firstRequirements,
		},
		compositeValidatorStub{
			name:         "musl",
			calls:        &calls,
			requirements: &secondRequirements,
		},
	)

	err := validator.Validate(
		context.Background(),
		StagedArtifact{
			RootPath:   "/tmp/runtime",
			BinaryPath: "/tmp/runtime/bin/tool",
			BinaryName: "tool",
		},
		requirements,
	)
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}

	wantCalls := []string{
		"static-elf",
		"musl",
	}

	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf(
			"validator calls = %v, want %v",
			calls,
			wantCalls,
		)
	}

	if !reflect.DeepEqual(
		firstRequirements,
		requirements,
	) {
		t.Fatalf(
			"first requirements = %+v, want %+v",
			firstRequirements,
			requirements,
		)
	}

	if !reflect.DeepEqual(
		secondRequirements,
		requirements,
	) {
		t.Fatalf(
			"second requirements = %+v, want %+v",
			secondRequirements,
			requirements,
		)
	}
}

func TestCompositeValidatorStopsAfterValidationFailure(t *testing.T) {
	var calls []string

	expectedErr := errors.New("validation failed")

	validator := NewCompositeValidator(
		compositeValidatorStub{
			name:  "static-elf",
			calls: &calls,
			err:   expectedErr,
		},
		compositeValidatorStub{
			name:  "musl",
			calls: &calls,
		},
	)

	err := validator.Validate(
		context.Background(),
		StagedArtifact{
			RootPath:   "/tmp/runtime",
			BinaryPath: "/tmp/runtime/bin/tool",
			BinaryName: "tool",
		},
		ValidationRequirements{
			StaticELF: true,
			Musl:      true,
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"Validate() error = %v, want %v",
			err,
			expectedErr,
		)
	}

	wantCalls := []string{
		"static-elf",
	}

	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf(
			"validator calls = %v, want %v",
			calls,
			wantCalls,
		)
	}
}

func TestCompositeValidatorWithNoValidatorsSucceeds(t *testing.T) {
	validator := NewCompositeValidator()

	err := validator.Validate(
		context.Background(),
		StagedArtifact{},
		ValidationRequirements{},
	)
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}
