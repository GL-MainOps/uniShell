package acquisition

import (
	"errors"
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2"
)

var (
	ErrInvalidMetadata       = errors.New("invalid acquisition metadata")
	ErrInvalidSourceMetadata = errors.New("invalid source metadata")
)

type Manifest struct {
	Tools []ToolMetadata `toml:"tools"`
}

type ToolMetadata struct {
	Name      string             `toml:"name"`
	Artifacts []ArtifactMetadata `toml:"artifacts"`
}

type ArtifactMetadata struct {
	Version      string                 `toml:"version"`
	Platform     Platform               `toml:"platform"`
	Architecture Architecture           `toml:"architecture"`
	ArchiveType  string                 `toml:"archive_type"`
	BinaryPath   string                 `toml:"binary_path"`
	Checksum     string                 `toml:"checksum"`
	Validation   ValidationRequirements `toml:"validation"`
	Source       SourceMetadata         `toml:"source"`
}

type SourceMetadata struct {
	Kind          SourceKind           `toml:"kind"`
	GitHubRelease *GitHubReleaseSource `toml:"github_release"`
	GitHubFile    *GitHubFileSource    `toml:"github_file"`
	DirectURL     *DirectURLSource     `toml:"direct_url"`
}

func LoadManifest(r io.Reader) (Manifest, error) {
	var manifest Manifest

	decoder := toml.NewDecoder(r)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("%w: %v", ErrInvalidMetadata, err)
	}

	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func (m Manifest) Validate() error {
	if len(m.Tools) == 0 {
		return fmt.Errorf("%w: no tools defined", ErrInvalidMetadata)
	}

	for _, tool := range m.Tools {
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("%w: tool %q: %v", ErrInvalidMetadata, tool.Name, err)
		}
	}

	return nil
}

func (t ToolMetadata) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("%w: tool name is required", ErrInvalidMetadata)
	}

	if len(t.Artifacts) == 0 {
		return fmt.Errorf("%w: tool %q has no artifacts", ErrInvalidMetadata, t.Name)
	}

	for _, artifact := range t.Artifacts {
		if err := artifact.Validate(); err != nil {
			return fmt.Errorf("%w: tool %q artifact: %v", ErrInvalidMetadata, t.Name, err)
		}
	}

	return nil
}

func (a ArtifactMetadata) Validate() error {
	if a.Platform == "" {
		return fmt.Errorf("%w: platform is required", ErrInvalidMetadata)
	}

	if a.Architecture == "" {
		return fmt.Errorf("%w: architecture is required", ErrInvalidMetadata)
	}

	if err := a.Source.Validate(); err != nil {
		return err
	}

	return nil
}

func (s SourceMetadata) Validate() error {
	switch s.Kind {
	case SourceKindGitHubRelease:
		if s.GitHubRelease == nil {
			return fmt.Errorf("%w: github_release configuration is required", ErrInvalidSourceMetadata)
		}

		if s.GitHubRelease.Owner == "" ||
			s.GitHubRelease.Repository == "" ||
			s.GitHubRelease.Release == "" ||
			s.GitHubRelease.Asset == "" {
			return fmt.Errorf("%w: github_release requires owner, repository, release, and asset", ErrInvalidSourceMetadata)
		}

	case SourceKindGitHubFile:
		if s.GitHubFile == nil {
			return fmt.Errorf("%w: github_file configuration is required", ErrInvalidSourceMetadata)
		}

		if s.GitHubFile.Owner == "" ||
			s.GitHubFile.Repository == "" ||
			s.GitHubFile.Path == "" ||
			s.GitHubFile.Ref == "" {
			return fmt.Errorf("%w: github_file requires owner, repository, path, and ref", ErrInvalidSourceMetadata)
		}

	case SourceKindDirectURL:
		if s.DirectURL == nil {
			return fmt.Errorf("%w: direct_url configuration is required", ErrInvalidSourceMetadata)
		}

		if s.DirectURL.URL == "" {
			return fmt.Errorf("%w: direct_url requires url", ErrInvalidSourceMetadata)
		}

	default:
		return fmt.Errorf("%w: unsupported source kind %q", ErrInvalidSourceMetadata, s.Kind)
	}

	return nil
}

func (a ArtifactMetadata) Artifact() (Artifact, error) {
	if err := a.Validate(); err != nil {
		return Artifact{}, err
	}

	var source Source

	switch a.Source.Kind {
	case SourceKindGitHubRelease:
		source = *a.Source.GitHubRelease
	case SourceKindGitHubFile:
		source = *a.Source.GitHubFile
	case SourceKindDirectURL:
		source = *a.Source.DirectURL
	default:
		return Artifact{}, fmt.Errorf("%w: unsupported source kind %q", ErrInvalidSourceMetadata, a.Source.Kind)
	}

	return Artifact{
		Version:      a.Version,
		Platform:     a.Platform,
		Architecture: a.Architecture,
		ArchiveType:  a.ArchiveType,
		BinaryPath:   a.BinaryPath,
		Checksum:     a.Checksum,
		Validation:   a.Validation,
		Source:       source,
	}, nil
}

func (t ToolMetadata) Tool() (Tool, error) {
	if err := t.Validate(); err != nil {
		return Tool{}, err
	}

	tool := Tool{
		Name: t.Name,
	}

	for _, metadata := range t.Artifacts {
		artifact, err := metadata.Artifact()
		if err != nil {
			return Tool{}, fmt.Errorf("%w: tool %q: %v", ErrInvalidMetadata, t.Name, err)
		}

		tool.Artifacts = append(tool.Artifacts, artifact)
	}

	return tool, nil
}

func (m Manifest) BuildTools() ([]Tool, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}

	tools := make([]Tool, 0, len(m.Tools))

	for _, metadata := range m.Tools {
		tool, err := metadata.Tool()
		if err != nil {
			return nil, err
		}

		tools = append(tools, tool)
	}

	return tools, nil
}
