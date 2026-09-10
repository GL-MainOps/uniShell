package bundle

import (
	"fmt"

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

// Open authenticates and decrypts an encrypted uniShell runtime bundle.
//
// The returned bytes contain the tar archive and must be passed to
// ExtractArchive to materialize the runtime.
func Open(data []byte, password string) ([]byte, error) {
	compressed, err := crypto.Decrypt(data, password)
	if err != nil {
		return nil, fmt.Errorf("open encrypted runtime bundle: %w", err)
	}

	if !defaultCompressor.Matches(compressed) {
		return compressed, nil
	}

	archive, err := defaultCompressor.Decompress(compressed)
	if err != nil {
		return nil, fmt.Errorf(
			"decompress runtime archive: %w",
			err,
		)
	}

	return archive, nil
}
