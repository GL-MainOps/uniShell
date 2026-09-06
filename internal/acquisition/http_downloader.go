package acquisition

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

type HTTPDownloader struct {
	Client *http.Client
}

func (d HTTPDownloader) Download(
	ctx context.Context,
	request DownloadRequest,
	dst io.Writer,
) error {
	if err := request.Artifact.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	if dst == nil {
		return fmt.Errorf("%w: destination writer is nil", ErrDownloadFailed)
	}

	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		request.Artifact.URL,
		nil,
	)
	if err != nil {
		return fmt.Errorf("%w: create request: %v", ErrDownloadFailed, err)
	}

	for name, value := range request.Headers {
		httpRequest.Header.Set(name, value)
	}

	response, err := client.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("%w: execute request: %v", ErrDownloadFailed, err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"%w: unexpected HTTP status %s",
			ErrDownloadFailed,
			response.Status,
		)
	}

	if _, err := io.Copy(dst, response.Body); err != nil {
		return fmt.Errorf("%w: copy response body: %v", ErrDownloadFailed, err)
	}

	return nil
}
