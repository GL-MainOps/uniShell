package bundle

import (
	"errors"
	"io"
)

var ErrInvalidCompressedBundle = errors.New(
	"invalid compressed bundle",
)

type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
	DecompressReader(reader io.Reader) (io.ReadCloser, error)
	Matches(data []byte) bool
	Close() error
}
