package bundle

import "errors"

var ErrInvalidCompressedBundle = errors.New(
	"invalid compressed bundle",
)

type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
	Matches(data []byte) bool
	Close() error
}
