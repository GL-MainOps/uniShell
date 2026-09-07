package acquisition

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type StagedArtifact struct {
	RootPath   string
	BinaryPath string
	BinaryName string
}

type Stager interface {
	Stage(
		context.Context,
		Artifact,
		io.Reader,
	) (StagedArtifact, error)
}

type FilesystemStager struct {
	BaseDir string
}

func NewFilesystemStager(baseDir string) FilesystemStager {
	return FilesystemStager{
		BaseDir: baseDir,
	}
}

func (s FilesystemStager) Stage(
	ctx context.Context,
	artifact Artifact,
	reader io.Reader,
) (StagedArtifact, error) {
	if err := artifact.Validate(); err != nil {
		return StagedArtifact{}, err
	}

	if err := ctx.Err(); err != nil {
		return StagedArtifact{}, err
	}

	if artifact.ArchiveType != "" {
		return StagedArtifact{}, ErrUnsupportedArchiveType
	}

	root, err := os.MkdirTemp(s.BaseDir, "unishell-stage-")
	if err != nil {
		return StagedArtifact{}, err
	}

	binaryPath := filepath.Join(root, artifact.BinaryName)

	file, err := os.OpenFile(
		binaryPath,
		os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		0700,
	)
	if err != nil {
		os.RemoveAll(root)
		return StagedArtifact{}, err
	}

	_, copyErr := io.Copy(file, reader)
	closeErr := file.Close()

	if copyErr != nil {
		os.RemoveAll(root)
		return StagedArtifact{}, copyErr
	}

	if closeErr != nil {
		os.RemoveAll(root)
		return StagedArtifact{}, closeErr
	}

	if err := ctx.Err(); err != nil {
		os.RemoveAll(root)
		return StagedArtifact{}, err
	}

	return StagedArtifact{
		RootPath:   root,
		BinaryPath: binaryPath,
		BinaryName: artifact.BinaryName,
	}, nil
}
