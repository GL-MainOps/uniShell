//go:build unishell_bundle

package bundle

import (
	_ "embed"
	"errors"
)

var ErrEmbeddedBundleUnavailable = errors.New(
	"embedded runtime bundle is unavailable",
)

//go:embed runtime.bundle
var embeddedBundle []byte

// Embedded returns a copy of the encrypted runtime bundle embedded
// during the build process.
func Embedded() ([]byte, error) {
	if len(embeddedBundle) == 0 {
		return nil, ErrEmbeddedBundleUnavailable
	}

	result := make([]byte, len(embeddedBundle))
	copy(result, embeddedBundle)

	return result, nil
}
