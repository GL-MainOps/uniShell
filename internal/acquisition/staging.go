package acquisition

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	unishellruntime "gitlab.com/mainops/uniShell/internal/runtime"
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

	if artifact.ArchiveType != "" &&
		artifact.ArchiveType != "tar" &&
		artifact.ArchiveType != "tar.gz" &&
		artifact.ArchiveType != "tgz" &&
		artifact.ArchiveType != "zip" {
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
		if err := extractArchive(root, artifact.ArchiveType, reader); err != nil {
			os.RemoveAll(root)
			return StagedArtifact{}, err
		}

		selectedPath, err := selectStagedBinary(root, artifact.BinaryPath)
		if err != nil {
			os.RemoveAll(root)
			return StagedArtifact{}, err
		}

		binaryPath, err = canonicalizeStagedBinary(
			root,
			selectedPath,
			artifact.BinaryName,
		)
		if err != nil {
			os.RemoveAll(root)
			return StagedArtifact{}, err
		}
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

func safeArchiveTarget(root, name string) (string, error) {
	path := filepath.FromSlash(name)
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("archive entry path is absolute: %q", name)
	}

	target := filepath.Join(root, path)

	paths := unishellruntime.Paths{
		Root: root,
	}

	if !unishellruntime.IsWithinRoot(paths, target) {
		return "", fmt.Errorf("archive entry path escapes staging root: %q", name)
	}

	return target, nil
}

func selectStagedBinary(root, binaryPath string) (string, error) {
	target, err := safeArchiveTarget(root, binaryPath)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("staged binary path %q is unavailable: %w", binaryPath, err)
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("staged binary path %q is not a regular file", binaryPath)
	}

	return target, nil
}

func canonicalizeStagedBinary(
	root,
	selectedPath,
	binaryName string,
) (string, error) {
	target, err := safeArchiveTarget(root, binaryName)
	if err != nil {
		return "", fmt.Errorf("invalid binary name %q: %w", binaryName, err)
	}

	if selectedPath == target {
		return target, nil
	}

	if err := os.Rename(selectedPath, target); err != nil {
		return "", fmt.Errorf(
			"canonicalize staged binary as %q: %w",
			binaryName,
			err,
		)
	}

	return target, nil
}

func extractArchive(root, archiveType string, reader io.Reader) error {
	switch archiveType {
	case "tar":
		return extractTar(root, reader)

	case "tar.gz", "tgz":
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		defer gzipReader.Close()

		return extractTar(root, gzipReader)

	case "zip":
		return extractZip(root, reader)

	default:
		return ErrUnsupportedArchiveType
	}
}

func extractTar(root string, reader io.Reader) error {
	tarReader := tar.NewReader(reader)

	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		target, err := safeArchiveTarget(root, header.Name)
		if err != nil {
			return err
		}

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

func extractZip(root string, reader io.Reader) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	zipReader, err := zip.NewReader(
		bytes.NewReader(data),
		int64(len(data)),
	)
	if err != nil {
		return err
	}

	for _, entry := range zipReader.File {
		target, err := safeArchiveTarget(root, entry.Name)
		if err != nil {
			return err
		}

		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, os.FileMode(entry.Mode())); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return err
		}

		source, err := entry.Open()
		if err != nil {
			return err
		}

		file, err := os.OpenFile(
			target,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
			os.FileMode(entry.Mode()),
		)
		if err != nil {
			source.Close()
			return err
		}

		_, copyErr := io.Copy(file, source)
		sourceCloseErr := source.Close()
		fileCloseErr := file.Close()

		if copyErr != nil {
			return copyErr
		}
		if sourceCloseErr != nil {
			return sourceCloseErr
		}
		if fileCloseErr != nil {
			return fileCloseErr
		}
	}

	return nil
}
