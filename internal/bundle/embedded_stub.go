//go:build !unishell_bundle

package bundle

import "errors"

var ErrEmbeddedBundleUnavailable = errors.New(
	"embedded runtime bundle is unavailable",
)

// Embedded returns an error because no production runtime bundle
// is embedded in non-bundle builds.
func Embedded() ([]byte, error) {
	return nil, ErrEmbeddedBundleUnavailable
}
