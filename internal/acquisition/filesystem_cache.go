package acquisition

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FilesystemCache struct {
	Dir string
}

func NewFilesystemCache(dir string) FilesystemCache {
	return FilesystemCache{
		Dir: dir,
	}
}

func (c FilesystemCache) Get(
	ctx context.Context,
	artifact ResolvedArtifact,
) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCacheFailed, err)
	}

	if err := artifact.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCacheFailed, err)
	}

	path, err := c.path(artifact)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCacheFailed, err)
	}

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCacheMiss
		}

		return nil, fmt.Errorf("%w: open cached artifact: %v", ErrCacheFailed, err)
	}

	return file, nil
}

func (c FilesystemCache) Put(
	ctx context.Context,
	artifact ResolvedArtifact,
	src io.Reader,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrCacheFailed, err)
	}

	if err := artifact.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrCacheFailed, err)
	}

	if src == nil {
		return fmt.Errorf("%w: source reader is nil", ErrCacheFailed)
	}

	path, err := c.path(artifact)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrCacheFailed, err)
	}

	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return fmt.Errorf("%w: create cache directory: %v", ErrCacheFailed, err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("%w: create cached artifact: %v", ErrCacheFailed, err)
	}

	if _, err := io.Copy(file, src); err != nil {
		_ = file.Close()
		_ = os.Remove(path)

		return fmt.Errorf("%w: write cached artifact: %v", ErrCacheFailed, err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(path)

		return fmt.Errorf("%w: close cached artifact: %v", ErrCacheFailed, err)
	}

	return nil
}

func (c FilesystemCache) path(artifact ResolvedArtifact) (string, error) {
	if c.Dir == "" {
		return "", fmt.Errorf("cache directory is required")
	}

	name := filepath.Base(artifact.URL)
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "", fmt.Errorf("artifact URL does not provide a cache filename")
	}

	key, err := cacheKey(artifact)
	if err != nil {
		return "", err
	}

	return filepath.Join(c.Dir, name+"-"+key), nil
}
