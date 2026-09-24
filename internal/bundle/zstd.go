package bundle

import (
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

type zstdCompressor struct {
	encoderOnce sync.Once
	encoder     *zstd.Encoder
	encoderErr  error
}

func newZstdCompressor() (*zstdCompressor, error) {
	// Keep construction cheap. Launchers only need a streaming decoder, while
	// bundle creation can initialize the encoder on its first Compress call.
	return &zstdCompressor{}, nil
}

func (c *zstdCompressor) Compress(data []byte) ([]byte, error) {
	c.encoderOnce.Do(func() {
		c.encoder, c.encoderErr = zstd.NewWriter(
			nil,
			zstd.WithEncoderLevel(zstd.SpeedFastest),
		)
	})
	if c.encoderErr != nil {
		return nil, c.encoderErr
	}
	return c.encoder.EncodeAll(data, nil), nil
}

func (c *zstdCompressor) Decompress(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, ErrInvalidCompressedBundle
	}
	defer decoder.Close()

	result, err := decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, ErrInvalidCompressedBundle
	}

	return result, nil
}

func (c *zstdCompressor) DecompressReader(
	reader io.Reader,
) (io.ReadCloser, error) {
	decoder, err := zstd.NewReader(reader)
	if err != nil {
		return nil, ErrInvalidCompressedBundle
	}

	return decoder.IOReadCloser(), nil
}

func (c *zstdCompressor) Matches(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 0x28 &&
		data[1] == 0xb5 &&
		data[2] == 0x2f &&
		data[3] == 0xfd
}

func (c *zstdCompressor) Close() error {
	if c.encoder == nil {
		return nil
	}
	return c.encoder.Close()
}

var defaultCompressor = func() *zstdCompressor {
	compressor, err := newZstdCompressor()
	if err != nil {
		panic(err)
	}
	return compressor
}()
