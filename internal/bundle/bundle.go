package bundle

import (
	"bytes"
	"fmt"
	"io"

	"gitlab.com/mainops/uniShell/internal/crypto"
)

// Create creates an encrypted uniShell runtime bundle from sourceDir.
func Create(sourceDir, password string) ([]byte, error) {
	archive, err := CreateArchive(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("create runtime archive: %w", err)
	}

	compressed, err := defaultCompressor.Compress(archive)
	if err != nil {
		return nil, fmt.Errorf("compress runtime archive: %w", err)
	}

	encrypted, err := crypto.Encrypt(compressed, password)
	if err != nil {
		return nil, fmt.Errorf("encrypt compressed runtime archive: %w", err)
	}

	return encrypted, nil
}

// OpenAuthenticated decrypts and authenticates an encrypted uniShell
// runtime bundle.
//
// The returned bytes contain the authenticated compressed payload. They
// must be passed to DecompressAuthenticated to materialize the tar archive.
func OpenAuthenticated(data []byte, password string) ([]byte, error) {
	compressed, err := crypto.Decrypt(data, password)
	if err != nil {
		return nil, fmt.Errorf(
			"open encrypted runtime bundle: %w",
			err,
		)
	}

	return compressed, nil
}

// DecompressAuthenticated decompresses an authenticated bundle payload.
//
// The payload must have already been authenticated by OpenAuthenticated.
func DecompressAuthenticated(data []byte) ([]byte, error) {
	if !defaultCompressor.Matches(data) {
		return data, nil
	}

	archive, err := defaultCompressor.Decompress(data)
	if err != nil {
		return nil, fmt.Errorf(
			"decompress runtime archive: %w",
			err,
		)
	}

	return archive, nil
}

// DecompressAuthenticatedReader creates a streaming reader over an
// authenticated bundle payload.
//
// The payload must have already been authenticated by OpenAuthenticated.
func DecompressAuthenticatedReader(
	data []byte,
) (io.ReadCloser, error) {
	if !defaultCompressor.Matches(data) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}

	reader, err := defaultCompressor.DecompressReader(
		bytes.NewReader(data),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"decompress runtime bundle: %w",
			err,
		)
	}

	return reader, nil
}

// Open authenticates and decrypts an encrypted uniShell runtime bundle,
// then decompresses its authenticated payload.
//
// The returned bytes contain the tar archive and must be passed to
// ExtractArchive to materialize the runtime.
func Open(data []byte, password string) ([]byte, error) {
	authenticated, err := OpenAuthenticated(data, password)
	if err != nil {
		return nil, err
	}

	return DecompressAuthenticated(authenticated)
}
