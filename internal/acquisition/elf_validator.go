package acquisition

import (
	"context"
	"debug/elf"
	"fmt"
	"io"
)

type StaticELFValidator struct{}

func hasDynamicDependency(
	file *elf.File,
	program *elf.Prog,
) (bool, error) {
	data, err := io.ReadAll(program.Open())
	if err != nil {
		return false, err
	}

	entrySize := 8
	if file.Class == elf.ELFCLASS64 {
		entrySize = 16
	}

	if len(data)%entrySize != 0 {
		return false, fmt.Errorf(
			"dynamic segment size is not a multiple of entry size",
		)
	}

	for offset := 0; offset < len(data); offset += entrySize {
		var tag int64

		if file.Class == elf.ELFCLASS64 {
			tag = int64(
				file.ByteOrder.Uint64(data[offset:]),
			)
		} else {
			tag = int64(
				int32(
					file.ByteOrder.Uint32(data[offset:]),
				),
			)
		}

		if tag == int64(elf.DT_NULL) {
			break
		}

		if tag == int64(elf.DT_NEEDED) {
			return true, nil
		}
	}

	return false, nil
}

func (StaticELFValidator) Validate(
	ctx context.Context,
	staged StagedArtifact,
	requirements ValidationRequirements,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if !requirements.StaticELF {
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

	for _, program := range file.Progs {
		switch program.Type {
		case elf.PT_INTERP:
			return fmt.Errorf(
				"ELF binary contains an interpreter",
			)

		case elf.PT_DYNAMIC:
			hasDependency, err := hasDynamicDependency(
				file,
				program,
			)
			if err != nil {
				return fmt.Errorf(
					"read ELF dynamic segment: %w",
					err,
				)
			}

			if hasDependency {
				return fmt.Errorf(
					"ELF binary has a shared-library dependency",
				)
			}
		}
	}

	return nil
}
