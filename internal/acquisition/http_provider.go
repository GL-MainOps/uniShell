package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

var (
	ErrSourceResolutionFailed = errors.New("source resolution failed")
	ErrSourceCredentialFailed = errors.New("source credential failed")
)

const defaultGitHubAPIBaseURL = "https://api.github.com"

type HTTPProvider struct {
	Client         *http.Client
	GitHubAPIBaseURL string
}

func NewHTTPProvider(client *http.Client) HTTPProvider {
	if client == nil {
		client = http.DefaultClient
	}

	return HTTPProvider{
		Client:           client,
		GitHubAPIBaseURL: defaultGitHubAPIBaseURL,
	}
}

func (p HTTPProvider) Resolve(
	ctx context.Context,
	artifact Artifact,
) (ResolvedArtifact, error) {
	if err := artifact.Validate(); err != nil {
		return ResolvedArtifact{}, fmt.Errorf(
			"%w: %v",
			ErrSourceResolutionFailed,
			err,
		)
	}

	switch source := artifact.Source.(type) {
	case DirectURLSource:
		return ResolvedArtifact{
			Version:       artifact.Version,
			Platform:      artifact.Platform,
			Architecture:  artifact.Architecture,
			URL:           source.URL,
			Checksum:      artifact.Checksum,
		}, nil

	case GitHubFileSource:
		rawURL := fmt.Sprintf(
			"https://raw.githubusercontent.com/%s/%s/%s/%s",
			source.Owner,
			source.Repository,
			url.PathEscape(source.Ref),
			strings.TrimPrefix(source.Path, "/"),
		)

		return ResolvedArtifact{
			Version:       artifact.Version,
			Platform:      artifact.Platform,
			Architecture:  artifact.Architecture,
			URL:           rawURL,
			Revision:      source.Ref,
			Checksum:      artifact.Checksum,
		}, nil

	case GitHubReleaseSource:
		return p.resolveGitHubRelease(ctx, artifact, source)

	default:
		return ResolvedArtifact{}, fmt.Errorf(
			"%w: unsupported source type %T",
			ErrSourceResolutionFailed,
			artifact.Source,
		)
	}
}

func (p HTTPProvider) Headers(
	artifact Artifact,
) (map[string]string, error) {
	if err := artifact.Validate(); err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrSourceResolutionFailed,
			err,
		)
	}

	var credentialEnv string

	switch source := artifact.Source.(type) {
	case DirectURLSource:
		credentialEnv = source.CredentialEnv
	case GitHubFileSource:
		credentialEnv = source.CredentialEnv
	case GitHubReleaseSource:
		credentialEnv = source.CredentialEnv
	default:
		return nil, fmt.Errorf(
			"%w: unsupported source type %T",
			ErrSourceResolutionFailed,
			artifact.Source,
		)
	}

	if credentialEnv == "" {
		return nil, nil
	}

	token := os.Getenv(credentialEnv)
	if token == "" {
		return nil, fmt.Errorf(
			"%w: environment variable %q is empty",
			ErrSourceCredentialFailed,
			credentialEnv,
		)
	}

	return map[string]string{
		"Authorization": "Bearer " + token,
	}, nil
}

type githubReleaseResponse struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func (p HTTPProvider) resolveGitHubRelease(
	ctx context.Context,
	artifact Artifact,
	source GitHubReleaseSource,
) (ResolvedArtifact, error) {
	baseURL := strings.TrimRight(p.GitHubAPIBaseURL, "/")
	releaseURL := fmt.Sprintf(
		"%s/repos/%s/%s/releases/latest",
		baseURL,
		source.Owner,
		source.Repository,
	)

	if source.Release != "latest" {
		releaseURL = fmt.Sprintf(
			"%s/repos/%s/%s/releases/tags/%s",
			baseURL,
			source.Owner,
			source.Repository,
			url.PathEscape(source.Release),
		)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		releaseURL,
		nil,
	)
	if err != nil {
		return ResolvedArtifact{}, fmt.Errorf(
			"%w: create GitHub release request: %v",
			ErrSourceResolutionFailed,
			err,
		)
	}

	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "uniShell")

	headers, err := p.Headers(artifact)
	if err != nil {
		return ResolvedArtifact{}, err
	}

	for name, value := range headers {
		request.Header.Set(name, value)
	}

	response, err := p.Client.Do(request)
	if err != nil {
		return ResolvedArtifact{}, fmt.Errorf(
			"%w: GitHub release request: %v",
			ErrSourceResolutionFailed,
			err,
		)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK ||
		response.StatusCode >= http.StatusMultipleChoices {
		return ResolvedArtifact{}, fmt.Errorf(
			"%w: GitHub release request returned %s",
			ErrSourceResolutionFailed,
			response.Status,
		)
	}

	var release githubReleaseResponse
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return ResolvedArtifact{}, fmt.Errorf(
			"%w: decode GitHub release response: %v",
			ErrSourceResolutionFailed,
			err,
		)
	}

	for _, asset := range release.Assets {
		if asset.Name != source.Asset {
			continue
		}

		return ResolvedArtifact{
			Version:       artifact.Version,
			Platform:      artifact.Platform,
			Architecture:  artifact.Architecture,
			URL:            asset.BrowserDownloadURL,
			Revision:      release.TagName,
			Checksum:      artifact.Checksum,
		}, nil
	}

	return ResolvedArtifact{}, fmt.Errorf(
		"%w: GitHub release asset %q was not found",
		ErrSourceResolutionFailed,
		source.Asset,
	)
}
