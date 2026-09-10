package bundle

import (
	"bytes"
	"testing"
)

func TestZstdCompressorRoundTrip(t *testing.T) {
	compressor, err := newZstdCompressor()
	if err != nil {
		t.Fatalf("create compressor: %v", err)
	}
	defer compressor.Close()

	source := bytes.Repeat(
		[]byte("uniShell runtime asset compression test\n"),
		10000,
	)

	compressed, err := compressor.Compress(source)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}

	if !compressor.Matches(compressed) {
		t.Fatal("compressed data was not recognized as Zstandard")
	}

	restored, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}

	if !bytes.Equal(restored, source) {
		t.Fatal("decompressed data does not match source")
	}
}

func TestZstdCompressorDoesNotMatchPlainData(t *testing.T) {
	compressor, err := newZstdCompressor()
	if err != nil {
		t.Fatalf("create compressor: %v", err)
	}
	defer compressor.Close()

	if compressor.Matches([]byte("plain tar data")) {
		t.Fatal("plain data incorrectly recognized as Zstandard")
	}
}

func TestZstdCompressorRejectsInvalidCompressedData(t *testing.T) {
	compressor, err := newZstdCompressor()
	if err != nil {
		t.Fatalf("create compressor: %v", err)
	}
	defer compressor.Close()

	invalid := []byte{
		0x28, 0xb5, 0x2f, 0xfd,
		0x00, 0x00, 0x00, 0x00,
	}

	if _, err := compressor.Decompress(invalid); err == nil {
		t.Fatal("expected invalid compressed data to fail")
	}
}
