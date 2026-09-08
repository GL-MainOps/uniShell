package acquisition

import (
	"context"
	"debug/elf"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeELF64Fixture(
	t *testing.T,
	path string,
	elfType uint16,
	programTypes ...uint32,
) {
	t.Helper()

	const (
		elfHeaderSize     = 64
		programHeaderSize = 56
		programHeaderOff  = elfHeaderSize
	)

	data := make([]byte, elfHeaderSize+programHeaderSize*len(programTypes))

	data[0] = 0x7f
	data[1] = 'E'
	data[2] = 'L'
	data[3] = 'F'
	data[4] = 2
	data[5] = 1
	data[6] = 1

	binary.LittleEndian.PutUint16(data[16:], elfType)
	binary.LittleEndian.PutUint16(data[18:], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint64(data[24:], 0)
	binary.LittleEndian.PutUint64(data[32:], programHeaderOff)
	binary.LittleEndian.PutUint64(data[40:], 0)
	binary.LittleEndian.PutUint32(data[48:], 0)
	binary.LittleEndian.PutUint16(data[52:], elfHeaderSize)
	binary.LittleEndian.PutUint16(data[54:], programHeaderSize)
	binary.LittleEndian.PutUint16(data[56:], uint16(len(programTypes)))
	binary.LittleEndian.PutUint16(data[58:], 0)
	binary.LittleEndian.PutUint16(data[60:], 0)
	binary.LittleEndian.PutUint16(data[62:], 0)

	for index, programType := range programTypes {
		offset := programHeaderOff + index*programHeaderSize
		binary.LittleEndian.PutUint32(data[offset:], programType)
	}

	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func writeELF64DynamicFixture(
	t *testing.T,
	path string,
	dynamicTags ...int64,
) {
	t.Helper()

	const (
		elfHeaderSize     = 64
		programHeaderSize = 56
		programHeaderOff  = elfHeaderSize
		dynamicOff        = programHeaderOff + programHeaderSize
		dynamicEntrySize  = 16
	)

	data := make(
		[]byte,
		dynamicOff+dynamicEntrySize*(len(dynamicTags)+1),
	)

	data[0] = 0x7f
	data[1] = 'E'
	data[2] = 'L'
	data[3] = 'F'
	data[4] = 2
	data[5] = 1
	data[6] = 1

	binary.LittleEndian.PutUint16(data[16:], uint16(elf.ET_DYN))
	binary.LittleEndian.PutUint16(data[18:], uint16(elf.EM_X86_64))
	binary.LittleEndian.PutUint32(data[20:], 1)
	binary.LittleEndian.PutUint64(data[24:], 0)
	binary.LittleEndian.PutUint64(data[32:], programHeaderOff)
	binary.LittleEndian.PutUint64(data[40:], 0)
	binary.LittleEndian.PutUint32(data[48:], 0)
	binary.LittleEndian.PutUint16(data[52:], elfHeaderSize)
	binary.LittleEndian.PutUint16(data[54:], programHeaderSize)
	binary.LittleEndian.PutUint16(data[56:], 1)
	binary.LittleEndian.PutUint16(data[58:], 0)
	binary.LittleEndian.PutUint16(data[60:], 0)
	binary.LittleEndian.PutUint16(data[62:], 0)

	binary.LittleEndian.PutUint32(
		data[programHeaderOff:],
		uint32(elf.PT_DYNAMIC),
	)
	binary.LittleEndian.PutUint32(
		data[programHeaderOff+4:],
		uint32(elf.PF_R),
	)
	binary.LittleEndian.PutUint64(
		data[programHeaderOff+8:],
		uint64(dynamicOff),
	)
	binary.LittleEndian.PutUint64(
		data[programHeaderOff+16:],
		uint64(dynamicOff),
	)
	binary.LittleEndian.PutUint64(
		data[programHeaderOff+32:],
		uint64(dynamicEntrySize*(len(dynamicTags)+1)),
	)
	binary.LittleEndian.PutUint64(
		data[programHeaderOff+40:],
		uint64(dynamicEntrySize*(len(dynamicTags)+1)),
	)
	binary.LittleEndian.PutUint64(
		data[programHeaderOff+48:],
		8,
	)

	for index, tag := range dynamicTags {
		offset := dynamicOff + index*dynamicEntrySize

		binary.LittleEndian.PutUint64(
			data[offset:],
			uint64(tag),
		)
		binary.LittleEndian.PutUint64(
			data[offset+8:],
			1,
		)
	}

	endOffset := dynamicOff + len(dynamicTags)*dynamicEntrySize
	binary.LittleEndian.PutUint64(
		data[endOffset:],
		uint64(elf.DT_NULL),
	)

	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func staticELFRequirements() ValidationRequirements {
	return ValidationRequirements{
		StaticELF: true,
	}
}

func stagedELFFixture(path string) StagedArtifact {
	return StagedArtifact{
		RootPath:   filepath.Dir(path),
		BinaryPath: path,
		BinaryName: "tool",
	}
}

func TestStaticELFValidatorSkipsWhenStaticELFNotRequired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-elf")

	if err := os.WriteFile(path, []byte("not an ELF"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := (StaticELFValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		ValidationRequirements{},
	)
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestStaticELFValidatorRejectsMissingBinaryPath(t *testing.T) {
	err := (StaticELFValidator{}).Validate(
		context.Background(),
		StagedArtifact{
			RootPath:   t.TempDir(),
			BinaryName: "tool",
		},
		staticELFRequirements(),
	)
	if err == nil {
		t.Fatal("Validate() error = nil, want missing binary path error")
	}
}

func TestStaticELFValidatorRejectsInvalidELF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid")

	if err := os.WriteFile(path, []byte("not an ELF"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	err := (StaticELFValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		staticELFRequirements(),
	)
	if err == nil {
		t.Fatal("Validate() error = nil, want invalid ELF error")
	}
}

func TestStaticELFValidatorAcceptsStaticELF(t *testing.T) {
	path := filepath.Join(t.TempDir(), "static")

	writeELF64Fixture(
		t,
		path,
		2,
	)

	err := (StaticELFValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		staticELFRequirements(),
	)
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestStaticELFValidatorAcceptsStaticPIE(t *testing.T) {
	path := filepath.Join(t.TempDir(), "static-pie")

	writeELF64Fixture(
		t,
		path,
		3,
	)

	err := (StaticELFValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		staticELFRequirements(),
	)
	if err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestStaticELFValidatorRejectsInterpreter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dynamic-loader")

	writeELF64Fixture(
		t,
		path,
		2,
		3,
	)

	err := (StaticELFValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		staticELFRequirements(),
	)
	if err == nil {
		t.Fatal("Validate() error = nil, want interpreter rejection")
	}
}

func TestStaticELFValidatorRejectsSharedLibraryDependency(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dynamic")

	writeELF64DynamicFixture(
		t,
		path,
		int64(elf.DT_NEEDED),
	)

	err := (StaticELFValidator{}).Validate(
		context.Background(),
		stagedELFFixture(path),
		staticELFRequirements(),
	)
	if err == nil {
		t.Fatal(
			"Validate() error = nil, want shared-library dependency rejection",
		)
	}
}

func TestStaticELFValidatorPropagatesCanceledContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "static")

	writeELF64Fixture(
		t,
		path,
		2,
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := (StaticELFValidator{}).Validate(
		ctx,
		stagedELFFixture(path),
		staticELFRequirements(),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"Validate() error = %v, want context.Canceled",
			err,
		)
	}
}
