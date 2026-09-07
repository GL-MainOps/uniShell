package acquisition

import "context"

type Provider interface {
	Resolve(context.Context, Artifact) (ResolvedArtifact, error)
	Headers(Artifact) (map[string]string, error)
}
