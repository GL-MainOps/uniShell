package acquisition

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func cacheKey(artifact ResolvedArtifact) (string, error) {
	if err := artifact.Validate(); err != nil {
		return "", err
	}

	identity := fmt.Sprintf(
		"%s\x00%s\x00%s\x00%s\x00%s\x00%s",
		artifact.Version,
		artifact.Platform,
		artifact.Architecture,
		artifact.URL,
		artifact.Revision,
		artifact.Checksum,
	)

	sum := sha256.Sum256([]byte(identity))

	return hex.EncodeToString(sum[:]), nil
}
