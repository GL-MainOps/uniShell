package acquisition

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/elf"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type pipelineStager struct {
	calls    int
	artifact Artifact
	data     string
	err      error
}

type pipelineValidator struct {
	calls        int
	requirements ValidationRequirements
	err          error
}

func (v *pipelineValidator) Validate(
	_ context.Context,
	_ StagedArtifact,
	requirements ValidationRequirements,
) error {
	v.calls++
	v.requirements = requirements

	return v.err
}

func (s *pipelineStager) Stage(
	_ context.Context,
	artifact Artifact,
	reader io.Reader,
) (StagedArtifact, error) {
	s.calls++
	s.artifact = artifact

	if s.err != nil {
		return StagedArtifact{}, s.err
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return StagedArtifact{}, err
	}

	s.data = string(data)

	return StagedArtifact{
		RootPath:   "/tmp/staged",
		BinaryPath: "/tmp/staged/tool",
		BinaryName: artifact.BinaryName,
	}, nil
}

func pipelineResolvedArtifact(content string) ResolvedArtifact {
	sum := sha256.Sum256([]byte(content))

	return ResolvedArtifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		URL:          "https://example.com/tool",
		Checksum:     hex.EncodeToString(sum[:]),
	}
}

func pipelineArtifact() Artifact {
	return Artifact{
		Version:      "1.0.0",
		Platform:     Platform("linux"),
		Architecture: Architecture("amd64"),
		BinaryName:   "tool",
		Source: DirectURLSource{
			URL: "https://example.com/tool",
		},
	}
}

func TestPipelineAcquireAndStage(t *testing.T) {
	content := "downloaded artifact"

	downloader := &fakeDownloader{
		content: content,
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}
	stager := &pipelineStager{}

	pipeline := NewPipeline(
		NewAcquirer(downloader, cache),
		stager,
		StaticELFValidator{},
	)

	artifact := pipelineArtifact()
	resolved := pipelineResolvedArtifact(content)

	staged, err := pipeline.AcquireAndStage(
		context.Background(),
		artifact,
		resolved,
		nil,
	)
	if err != nil {
		t.Fatalf("AcquireAndStage() error = %v", err)
	}

	if stager.calls != 1 {
		t.Fatalf("stager calls = %d, want 1", stager.calls)
	}

	if stager.data != content {
		t.Fatalf(
			"staged content = %q, want %q",
			stager.data,
			content,
		)
	}

	if stager.artifact.BinaryName != "tool" {
		t.Fatalf(
			"staged artifact BinaryName = %q, want %q",
			stager.artifact.BinaryName,
			"tool",
		)
	}

	if staged.BinaryName != "tool" {
		t.Fatalf(
			"result BinaryName = %q, want %q",
			staged.BinaryName,
			"tool",
		)
	}

	if downloader.calls != 1 {
		t.Fatalf(
			"downloader calls = %d, want 1",
			downloader.calls,
		)
	}

	if cache.putCalls != 1 {
		t.Fatalf(
			"cache put calls = %d, want 1",
			cache.putCalls,
		)
	}
}

func TestPipelineValidatesStagedArtifact(t *testing.T) {
	content := "downloaded artifact"

	downloader := &fakeDownloader{
		content: content,
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}
	stager := &pipelineStager{}
	validator := &pipelineValidator{}

	pipeline := NewPipeline(
		NewAcquirer(downloader, cache),
		stager,
		validator,
	)

	artifact := pipelineArtifact()
	artifact.Validation = ValidationRequirements{
		StaticELF:  true,
		Musl:       true,
		Executable: true,
	}

	resolved := pipelineResolvedArtifact(content)

	_, err := pipeline.AcquireAndStage(
		context.Background(),
		artifact,
		resolved,
		nil,
	)
	if err != nil {
		t.Fatalf("AcquireAndStage() error = %v", err)
	}

	if validator.calls != 1 {
		t.Fatalf(
			"validator calls = %d, want 1",
			validator.calls,
		)
	}

	if validator.requirements != artifact.Validation {
		t.Fatalf(
			"validator requirements = %+v, want %+v",
			validator.requirements,
			artifact.Validation,
		)
	}
}

func TestPipelineRunsCompositeValidator(t *testing.T) {
	baseDir := t.TempDir()
	sourcePath := filepath.Join(baseDir, "tool")
	writeELF64Fixture(
		t,
		sourcePath,
		uint16(elf.ET_EXEC),
	)

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	content := string(data)
	resolved := pipelineResolvedArtifact(content)

	artifact := pipelineArtifact()
	artifact.Validation = ValidationRequirements{
		StaticELF: true,
		Musl:      true,
	}

	stagingDir := filepath.Join(baseDir, "stage")

	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	pipeline := NewPipeline(
		NewAcquirer(
			&fakeDownloader{
				content: content,
			},
			&fakeCache{
				getErr: ErrCacheMiss,
			},
		),
		NewFilesystemStager(stagingDir),
		NewCompositeValidator(
			StaticELFValidator{},
			MuslValidator{},
		),
	)

	staged, err := pipeline.AcquireAndStage(
		context.Background(),
		artifact,
		resolved,
		nil,
	)
	if err != nil {
		t.Fatalf("AcquireAndStage() error = %v", err)
	}
	defer os.RemoveAll(staged.RootPath)

	if staged.BinaryName != "tool" {
		t.Fatalf(
			"staged BinaryName = %q, want %q",
			staged.BinaryName,
			"tool",
		)
	}

	if staged.BinaryPath == "" {
		t.Fatal("staged BinaryPath is empty")
	}
}

func TestPipelineRejectsInvalidArtifactWithCompositeValidator(t *testing.T) {
	baseDir := t.TempDir()

	artifact := pipelineArtifact()
	artifact.Validation = ValidationRequirements{
		StaticELF: true,
		Musl:      true,
	}

	content := "not an ELF"
	resolved := pipelineResolvedArtifact(content)

	stagingDir := filepath.Join(baseDir, "stage")

	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	stager := NewFilesystemStager(stagingDir)

	pipeline := NewPipeline(
		NewAcquirer(
			&fakeDownloader{
				content: content,
			},
			&fakeCache{
				getErr: ErrCacheMiss,
			},
		),
		stager,
		NewCompositeValidator(
			StaticELFValidator{},
			MuslValidator{},
		),
	)

	_, err := pipeline.AcquireAndStage(
		context.Background(),
		artifact,
		resolved,
		nil,
	)
	if err == nil {
		t.Fatal(
			"AcquireAndStage() error = nil, want validation failure",
		)
	}

	if !strings.Contains(
		err.Error(),
		"validate staged artifact",
	) {
		t.Fatalf(
			"AcquireAndStage() error = %v, want validation error",
			err,
		)
	}
}

func TestPipelineRejectsValidationFailure(t *testing.T) {
	content := "downloaded artifact"
	validationErr := errors.New("static ELF validation failure")

	downloader := &fakeDownloader{
		content: content,
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}
	stager := &pipelineStager{}
	validator := &pipelineValidator{
		err: validationErr,
	}

	pipeline := NewPipeline(
		NewAcquirer(downloader, cache),
		stager,
		validator,
	)

	artifact := pipelineArtifact()
	artifact.Validation = ValidationRequirements{
		StaticELF: true,
	}

	_, err := pipeline.AcquireAndStage(
		context.Background(),
		artifact,
		pipelineResolvedArtifact(content),
		nil,
	)
	if err == nil {
		t.Fatal("AcquireAndStage() error = nil, want validation failure")
	}

	if !errors.Is(err, validationErr) {
		t.Fatalf(
			"AcquireAndStage() error = %v, want validation failure",
			err,
		)
	}

	if validator.calls != 1 {
		t.Fatalf(
			"validator calls = %d, want 1",
			validator.calls,
		)
	}
}

func TestPipelineUsesCachedArtifact(t *testing.T) {
	content := "cached artifact"

	sum := sha256.Sum256([]byte(content))

	artifact := pipelineArtifact()
	resolved := pipelineResolvedArtifact(content)
	resolved.Checksum = hex.EncodeToString(sum[:])

	downloader := &fakeDownloader{
		content: "should not download",
	}
	cache := &fakeCache{
		content: content,
	}
	stager := &pipelineStager{}

	pipeline := NewPipeline(
		NewAcquirer(downloader, cache),
		stager,
		StaticELFValidator{},
	)

	_, err := pipeline.AcquireAndStage(
		context.Background(),
		artifact,
		resolved,
		nil,
	)
	if err != nil {
		t.Fatalf("AcquireAndStage() error = %v", err)
	}

	if downloader.calls != 0 {
		t.Fatalf(
			"downloader calls = %d, want 0",
			downloader.calls,
		)
	}

	if stager.calls != 1 {
		t.Fatalf(
			"stager calls = %d, want 1",
			stager.calls,
		)
	}

	if stager.data != content {
		t.Fatalf(
			"staged content = %q, want %q",
			stager.data,
			content,
		)
	}
}

func TestPipelineDoesNotStageAfterAcquisitionFailure(t *testing.T) {
	downloadErr := errors.New("download failure")

	downloader := &fakeDownloader{
		err: downloadErr,
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}
	stager := &pipelineStager{}

	pipeline := NewPipeline(
		NewAcquirer(downloader, cache),
		stager,
		StaticELFValidator{},
	)

	_, err := pipeline.AcquireAndStage(
		context.Background(),
		pipelineArtifact(),
		pipelineResolvedArtifact("downloaded artifact"),
		nil,
	)
	if err == nil {
		t.Fatal("AcquireAndStage() error = nil, want download failure")
	}

	if !errors.Is(err, downloadErr) {
		t.Fatalf(
			"AcquireAndStage() error = %v, want download failure",
			err,
		)
	}

	if stager.calls != 0 {
		t.Fatalf(
			"stager calls = %d, want 0",
			stager.calls,
		)
	}
}

func TestPipelinePropagatesStagingFailure(t *testing.T) {
	stageErr := errors.New("staging failure")

	downloader := &fakeDownloader{
		content: "downloaded artifact",
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}
	stager := &pipelineStager{
		err: stageErr,
	}

	pipeline := NewPipeline(
		NewAcquirer(downloader, cache),
		stager,
		StaticELFValidator{},
	)

	_, err := pipeline.AcquireAndStage(
		context.Background(),
		pipelineArtifact(),
		pipelineResolvedArtifact("downloaded artifact"),
		nil,
	)
	if err == nil {
		t.Fatal("AcquireAndStage() error = nil, want staging failure")
	}

	if !errors.Is(err, stageErr) {
		t.Fatalf(
			"AcquireAndStage() error = %v, want staging failure",
			err,
		)
	}
}

func TestPipelineRejectsNilStager(t *testing.T) {
	downloader := &fakeDownloader{
		content: "downloaded artifact",
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}

	pipeline := NewPipeline(
		NewAcquirer(downloader, cache),
		nil,
		StaticELFValidator{},
	)

	_, err := pipeline.AcquireAndStage(
		context.Background(),
		pipelineArtifact(),
		pipelineResolvedArtifact("downloaded artifact"),
		nil,
	)
	if err == nil {
		t.Fatal("AcquireAndStage() error = nil, want nil stager error")
	}

	if !strings.Contains(
		err.Error(),
		"stager is nil",
	) {
		t.Fatalf(
			"AcquireAndStage() error = %v, want stager is nil",
			err,
		)
	}

	if downloader.calls != 0 {
		t.Fatalf(
			"downloader calls = %d, want 0",
			downloader.calls,
		)
	}
}

func TestPipelineRejectsCanceledContextBeforeAcquisition(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	downloader := &fakeDownloader{
		content: "downloaded artifact",
	}
	cache := &fakeCache{
		getErr: ErrCacheMiss,
	}
	stager := &pipelineStager{}

	pipeline := NewPipeline(
		NewAcquirer(downloader, cache),
		stager,
		StaticELFValidator{},
	)

	_, err := pipeline.AcquireAndStage(
		ctx,
		pipelineArtifact(),
		pipelineResolvedArtifact("downloaded artifact"),
		nil,
	)
	if err == nil {
		t.Fatal("AcquireAndStage() error = nil, want context cancellation")
	}

	if !errors.Is(err, ErrDownloadFailed) {
		t.Fatalf(
			"AcquireAndStage() error = %v, want download failure",
			err,
		)
	}

	if stager.calls != 0 {
		t.Fatalf(
			"stager calls = %d, want 0",
			stager.calls,
		)
	}
}

func TestPipelineStagesRealFilesystemArtifact(t *testing.T) {
	content := "binary"

	sum := sha256.Sum256([]byte(content))

	artifact := pipelineArtifact()
	artifact.BinaryName = "tool"

	resolved := pipelineResolvedArtifact(content)
	resolved.Checksum = hex.EncodeToString(sum[:])

	baseDir := t.TempDir()
	stager := NewFilesystemStager(baseDir)

	pipeline := NewPipeline(
		NewAcquirer(
			&fakeDownloader{
				content: content,
			},
			&fakeCache{
				getErr: ErrCacheMiss,
			},
		),
		stager,
		StaticELFValidator{},
	)

	staged, err := pipeline.AcquireAndStage(
		context.Background(),
		artifact,
		resolved,
		nil,
	)
	if err != nil {
		t.Fatalf("AcquireAndStage() error = %v", err)
	}
	defer os.RemoveAll(staged.RootPath)

	wantPath := filepath.Join(
		staged.RootPath,
		"tool",
	)

	if staged.BinaryPath != wantPath {
		t.Fatalf(
			"BinaryPath = %q, want %q",
			staged.BinaryPath,
			wantPath,
		)
	}

	data, err := os.ReadFile(staged.BinaryPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if !bytes.Equal(data, []byte(content)) {
		t.Fatalf(
			"staged binary = %q, want %q",
			data,
			content,
		)
	}
}
