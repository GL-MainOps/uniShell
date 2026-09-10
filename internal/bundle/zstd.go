package bundle

import "github.com/klauspost/compress/zstd"

type zstdCompressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

func newZstdCompressor() (*zstdCompressor, error) {
	encoder, err := zstd.NewWriter(
		nil,
		zstd.WithEncoderLevel(zstd.SpeedBestCompression),
	)
	if err != nil {
		return nil, err
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		encoder.Close()
		return nil, err
	}

	return &zstdCompressor{
		encoder: encoder,
		decoder: decoder,
	}, nil
}

func (c *zstdCompressor) Compress(data []byte) ([]byte, error) {
	return c.encoder.EncodeAll(data, nil), nil
}

func (c *zstdCompressor) Decompress(data []byte) ([]byte, error) {
	result, err := c.decoder.DecodeAll(data, nil)
	if err != nil {
		return nil, ErrInvalidCompressedBundle
	}

	return result, nil
}

func (c *zstdCompressor) Matches(data []byte) bool {
	return len(data) >= 4 &&
		data[0] == 0x28 &&
		data[1] == 0xb5 &&
		data[2] == 0x2f &&
		data[3] == 0xfd
}

func (c *zstdCompressor) Close() error {
	return c.encoder.Close()
}

var defaultCompressor = func() *zstdCompressor {
	compressor, err := newZstdCompressor()
	if err != nil {
		panic(err)
	}

	return compressor
}()
