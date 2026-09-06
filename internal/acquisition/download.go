package acquisition

import (
	"context"
	"errors"
	"io"
)

var (
	ErrDownloadFailed = errors.New("download failed")
	ErrCacheMiss      = errors.New("cache miss")
	ErrCacheFailed    = errors.New("cache operation failed")
)

type Downloader interface {
	Download(context.Context, ResolvedArtifact, io.Writer) error
}

type Cache interface {
	Get(context.Context, ResolvedArtifact) (io.ReadCloser, error)
	Put(context.Context, ResolvedArtifact, io.Reader) error
}
