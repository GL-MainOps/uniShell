package acquisition

import (
	"context"
	"debug/elf"
	"fmt"
)

type MuslValidator struct{}

func (MuslValidator) Validate(
	ctx context.Context,
	staged StagedArtifact,
	requirements ValidationRequirements,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if !requirements.Musl {
		return nil
	}

	if staged.BinaryPath == "" {
		return fmt.Errorf("binary path is required")
	}

	file, err := elf.Open(staged.BinaryPath)
	if err != nil {
		return fmt.Errorf("open ELF binary: %w", err)
	}
	defer file.Close()

	if file.Type != elf.ET_EXEC && file.Type != elf.ET_DYN {
		return fmt.Errorf(
			"ELF binary has unsupported type %s",
			file.Type,
		)
	}

	return nil
}
