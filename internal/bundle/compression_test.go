package bundle

import (
	"bytes"
	"io"
	"testing"
)

func TestZstdCompressorStreamingRoundTrip(t *testing.T) {
	compressor, err := newZstdCompressor()
	if err != nil {
		t.Fatalf("create compressor: %v", err)
	}
	defer compressor.Close()

	source := bytes.Repeat(
		[]byte("uniShell streaming decompression test\n"),
		10000,
	)

	compressed, err := compressor.Compress(source)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}

	reader, err := compressor.DecompressReader(
		bytes.NewReader(compressed),
	)
	if err != nil {
		t.Fatalf("create decompression reader: %v", err)
	}
	defer reader.Close()

	restored, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read decompressed data: %v", err)
	}

	if !bytes.Equal(restored, source) {
		t.Fatal("streaming decompressed data does not match source")
	}
}

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

func TestZstdCompressorStreamingRejectsTruncatedData(t *testing.T) {
	compressor, err := newZstdCompressor()
	if err != nil {
		t.Fatalf("create compressor: %v", err)
	}
	defer compressor.Close()

	source := bytes.Repeat(
		[]byte("uniShell streaming corruption test\n"),
		10000,
	)

	compressed, err := compressor.Compress(source)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}

	if len(compressed) < 2 {
		t.Fatal("compressed data is unexpectedly short")
	}

	truncated := compressed[:len(compressed)-1]

	reader, err := compressor.DecompressReader(
		bytes.NewReader(truncated),
	)
	if err != nil {
		t.Fatalf(
			"create decompression reader returned unexpected error: %v",
			err,
		)
	}
	defer reader.Close()

	if _, err := io.ReadAll(reader); err == nil {
		t.Fatal("reading truncated compressed data returned nil error")
	}
}
