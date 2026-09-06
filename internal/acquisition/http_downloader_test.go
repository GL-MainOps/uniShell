package acquisition

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPDownloaderDownloadsResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.Method != http.MethodGet {
			t.Errorf("request method = %q, want %q", request.Method, http.MethodGet)
		}

		if got, want := request.Header.Get("X-Test-Header"), "test-value"; got != want {
			t.Errorf("X-Test-Header = %q, want %q", got, want)
		}

		writer.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(writer, "downloaded")
	}))
	defer server.Close()

	var dst strings.Builder

	err := (HTTPDownloader{}).Download(
		context.Background(),
		DownloadRequest{
			Artifact: ResolvedArtifact{
				Version:      "1.0.0",
				Platform:     "linux",
				Architecture: "amd64",
				URL:          server.URL,
			},
			Headers: map[string]string{
				"X-Test-Header": "test-value",
			},
		},
		&dst,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := dst.String(), "downloaded"; got != want {
		t.Fatalf("downloaded data = %q, want %q", got, want)
	}
}

func TestHTTPDownloaderFollowsRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		if request.URL.Path == "/redirect" {
			http.Redirect(writer, request, "/download", http.StatusFound)
			return
		}

		if request.URL.Path == "/download" {
			writer.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(writer, "redirected")
			return
		}

		http.NotFound(writer, request)
	}))
	defer server.Close()

	var dst strings.Builder

	err := (HTTPDownloader{}).Download(
		context.Background(),
		DownloadRequest{
			Artifact: ResolvedArtifact{
				Version:      "1.0.0",
				Platform:     "linux",
				Architecture: "amd64",
				URL:          server.URL + "/redirect",
			},
		},
		&dst,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := dst.String(), "redirected"; got != want {
		t.Fatalf("downloaded data = %q, want %q", got, want)
	}
}

func TestHTTPDownloaderRejectsNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		_ *http.Request,
	) {
		http.Error(writer, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	var dst strings.Builder

	err := (HTTPDownloader{}).Download(
		context.Background(),
		DownloadRequest{
			Artifact: ResolvedArtifact{
				Version:      "1.0.0",
				Platform:     "linux",
				Architecture: "amd64",
				URL:          server.URL,
			},
		},
		&dst,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrDownloadFailed) {
		t.Fatalf("error = %v, want ErrDownloadFailed", err)
	}
}

func TestHTTPDownloaderHonorsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(
		writer http.ResponseWriter,
		request *http.Request,
	) {
		<-request.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var dst strings.Builder

	err := (HTTPDownloader{}).Download(
		ctx,
		DownloadRequest{
			Artifact: ResolvedArtifact{
				Version:      "1.0.0",
				Platform:     "linux",
				Architecture: "amd64",
				URL:          server.URL,
			},
		},
		&dst,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrDownloadFailed) {
		t.Fatalf("error = %v, want ErrDownloadFailed", err)
	}
}

func TestHTTPDownloaderRejectsInvalidArtifact(t *testing.T) {
	var dst strings.Builder

	err := (HTTPDownloader{}).Download(
		context.Background(),
		DownloadRequest{
			Artifact: ResolvedArtifact{
				Platform:     "linux",
				Architecture: "amd64",
			},
		},
		&dst,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrDownloadFailed) {
		t.Fatalf("error = %v, want ErrDownloadFailed", err)
	}

	if err.Error() != "download failed: invalid resolved artifact: URL is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}
