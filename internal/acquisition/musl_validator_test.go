package acquisition

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func muslRequirements() ValidationRequirements {
	return ValidationRequirements{
		Musl: true,
	}
}

func TestMuslValidatorSkipsWhenMuslNotRequired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-elf")

	if err := os.WriteFile(
		path,
		[]byte("not an ELF"),
		0o755,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := (MuslValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		ValidationRequirements{},
	)
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestMuslValidatorRejectsMissingBinaryPath(t *testing.T) {
	err := (MuslValidator{}).Validate(
		context.Background(),
		StagedArtifact{
			RootPath:   t.TempDir(),
			BinaryName: "tool",
		},
		muslRequirements(),
	)
	if err == nil {
		t.Fatal("Validate() error = nil, want missing binary path error")
	}
}

func TestMuslValidatorRejectsInvalidELF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid")

	if err := os.WriteFile(
		path,
		[]byte("not an ELF"),
		0o755,
	); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := (MuslValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		muslRequirements(),
	)
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid ELF error")
	}
}

func TestMuslValidatorAcceptsETEXEC(t *testing.T) {
	path := filepath.Join(t.TempDir(), "static")

	writeELF64Fixture(
		t,
		path,
		2,
	)

	err := (MuslValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		muslRequirements(),
	)
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestMuslValidatorAcceptsETDYN(t *testing.T) {
	path := filepath.Join(t.TempDir(), "static-pie")

	writeELF64Fixture(
		t,
		path,
		3,
	)

	err := (MuslValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		muslRequirements(),
	)
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestMuslValidatorAcceptsELFWithoutMuslMarker(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unmarked")

	writeELF64Fixture(
		t,
		path,
		3,
	)

	err := (MuslValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		muslRequirements(),
	)
	if err != nil {
		t.Fatalf(
			"Validate() error = %v, want nil for ELF without musl marker",
			err,
		)
	}
}

func TestMuslValidatorPropagatesCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "static")

	writeELF64Fixture(
		t,
		path,
		2,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := (MuslValidator{}).Validate(
		ctx,
		stagedELFFixture(path),
		muslRequirements(),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"Validate() error = %v, want context.Canceled",
			err,
		)
	}
}
