package acquisition

import "context"

type Provider interface {
	Kind() SourceKind
	Resolve(context.Context, Artifact) (ResolvedArtifact, error)
}
