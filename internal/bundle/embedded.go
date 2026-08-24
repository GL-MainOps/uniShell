package bundle

import "errors"

var ErrEmbeddedBundleUnavailable = errors.New(
	"embedded runtime bundle is unavailable",
)

// Embedded returns a copy of the encrypted runtime bundle generated
// during the build process.
func Embedded() ([]byte, error) {
	data := generatedBundle()

	if len(data) == 0 {
		return nil, ErrEmbeddedBundleUnavailable
	}

	result := make([]byte, len(data))
	copy(result, data)

	return result, nil
}
