package acquisition

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidTool             = errors.New("invalid tool")
	ErrInvalidArtifact         = errors.New("invalid artifact")
	ErrInvalidSource           = errors.New("invalid source")
	ErrInvalidResolvedArtifact = errors.New("invalid resolved artifact")
)

type Platform string

type Architecture string

type SourceKind string

const (
	SourceKindGitHubRelease SourceKind = "github-release"
	SourceKindGitHubFile    SourceKind = "github-file"
	SourceKindDirectURL     SourceKind = "direct-url"
)

type Tool struct {
	Name      string
	Artifacts []Artifact
}

type Artifact struct {
	Version      string
	Platform     Platform
	Architecture Architecture
	Source       Source
	Checksum     string
}

type Source interface {
	Kind() SourceKind
}

type ResolvedArtifact struct {
	Version      string
	Platform     Platform
	Architecture Architecture
	URL          string
	Revision     string
	Checksum     string
}

func (t Tool) Validate() error {
	if t.Name == "" {
		return ErrInvalidTool
	}

	if len(t.Artifacts) == 0 {
		return fmt.Errorf("%w: no artifacts defined", ErrInvalidTool)
	}

	for _, artifact := range t.Artifacts {
		if err := artifact.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidTool, err)
		}
	}

	return nil
}

func (a Artifact) Validate() error {
	if a.Platform == "" {
		return fmt.Errorf("%w: platform is required", ErrInvalidArtifact)
	}

	if a.Architecture == "" {
		return fmt.Errorf("%w: architecture is required", ErrInvalidArtifact)
	}

	if a.Source == nil {
		return fmt.Errorf("%w: source is required", ErrInvalidArtifact)
	}

	return nil
}

func (a ResolvedArtifact) Validate() error {
	if a.URL == "" {
		return fmt.Errorf("%w: URL is required", ErrInvalidResolvedArtifact)
	}

	if a.Platform == "" {
		return fmt.Errorf("%w: platform is required", ErrInvalidResolvedArtifact)
	}

	if a.Architecture == "" {
		return fmt.Errorf("%w: architecture is required", ErrInvalidResolvedArtifact)
	}

	return nil
}
