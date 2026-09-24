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

// Embedded returns a copy of the runtime bundle embedded
// during the build process.
func Embedded() ([]byte, error) {
	if len(embeddedBundle) == 0 {
		return nil, ErrEmbeddedBundleUnavailable
	}

	result := make([]byte, len(embeddedBundle))
	copy(result, embeddedBundle)

	return result, nil
}

// EmbeddedView returns the runtime bundle without copying it.
// The returned bytes refer to read-only embedded data and must not be modified.
func EmbeddedView() ([]byte, error) {
	if len(embeddedBundle) == 0 {
		return nil, ErrEmbeddedBundleUnavailable
	}

	return embeddedBundle, nil
}
