package bundle

import "errors"

var ErrEmbeddedBundleUnavailable = errors.New(
	"embedded runtime bundle is unavailable",
)

var generatedBundle []byte

// Embedded returns a copy of the encrypted runtime bundle generated
// during the build process.
func Embedded() ([]byte, error) {
	if len(generatedBundle) == 0 {
		return nil, ErrEmbeddedBundleUnavailable
	}

	result := make([]byte, len(generatedBundle))
	copy(result, generatedBundle)

	return result, nil
}
