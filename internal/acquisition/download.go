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

type DownloadRequest struct {
	Artifact ResolvedArtifact
	Headers  map[string]string
}

type Downloader interface {
	Download(context.Context, DownloadRequest, io.Writer) error
}

type Cache interface {
	Get(context.Context, ResolvedArtifact) (io.ReadCloser, error)
	Put(context.Context, ResolvedArtifact, io.Reader) error
}
