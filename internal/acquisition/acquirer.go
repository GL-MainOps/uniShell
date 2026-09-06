package acquisition

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

type Acquirer struct {
	Downloader Downloader
	Cache      Cache
}

func NewAcquirer(downloader Downloader, cache Cache) Acquirer {
	return Acquirer{
		Downloader: downloader,
		Cache:      cache,
	}
}

func (a Acquirer) Acquire(
	ctx context.Context,
	artifact ResolvedArtifact,
	headers map[string]string,
) (io.ReadCloser, error) {
	if err := artifact.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	if a.Downloader == nil {
		return nil, fmt.Errorf("%w: downloader is nil", ErrDownloadFailed)
	}

	if a.Cache == nil {
		return nil, fmt.Errorf("%w: cache is nil", ErrCacheFailed)
	}

	cached, err := a.Cache.Get(ctx, artifact)
	if err == nil {
		return cached, nil
	}

	if err != ErrCacheMiss {
		return nil, err
	}

	var downloaded bytes.Buffer

	if err := a.Downloader.Download(
		ctx,
		DownloadRequest{
			Artifact: artifact,
			Headers:  headers,
		},
		&downloaded,
	); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	if err := a.Cache.Put(
		ctx,
		artifact,
		bytes.NewReader(downloaded.Bytes()),
	); err != nil {
		return nil, err
	}

	return io.NopCloser(bytes.NewReader(downloaded.Bytes())), nil
}
