package acquisition

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
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

	if artifact.ArchiveType != "" && artifact.ArchiveType != "tar.gz" {
		return StagedArtifact{}, ErrUnsupportedArchiveType
	}

	root, err := os.MkdirTemp(s.BaseDir, "unishell-stage-")
	if err != nil {
		return StagedArtifact{}, err
	}

	var binaryPath string

	if artifact.ArchiveType == "" {
		binaryPath = filepath.Join(root, artifact.BinaryName)

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
	} else {
		if err := extractTarGz(root, reader); err != nil {
			os.RemoveAll(root)
			return StagedArtifact{}, err
		}

		binaryPath = filepath.Join(root, artifact.BinaryPath)
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

func extractTarGz(root string, reader io.Reader) error {
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		target := filepath.Join(root, filepath.FromSlash(header.Name))

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return err
			}

			file, err := os.OpenFile(
				target,
				os.O_WRONLY|os.O_CREATE|os.O_EXCL,
				os.FileMode(header.Mode),
			)
			if err != nil {
				return err
			}

			_, copyErr := io.Copy(file, tarReader)
			closeErr := file.Close()

			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}

		default:
			return fmt.Errorf("unsupported tar entry type %d", header.Typeflag)
		}
	}
}
