package acquisition

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

var ErrChecksumMismatch = errors.New("checksum mismatch")

func validateChecksum(expected string) ([]byte, error) {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil, nil
	}

	expectedBytes, err := hex.DecodeString(expected)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid checksum: %v", ErrInvalidResolvedArtifact, err)
	}

	if len(expectedBytes) != sha256.Size {
		return nil, fmt.Errorf(
			"%w: checksum must be %d bytes",
			ErrInvalidResolvedArtifact,
			sha256.Size,
		)
	}

	return expectedBytes, nil
}

func VerifyChecksum(reader io.Reader, expected string) error {
	if reader == nil {
		return fmt.Errorf("%w: reader is nil", ErrInvalidResolvedArtifact)
	}

	expectedBytes, err := validateChecksum(expected)
	if err != nil {
		return err
	}
	if len(expectedBytes) == 0 {
		return nil
	}

	hash := sha256.New()

	if _, err := io.Copy(hash, reader); err != nil {
		return fmt.Errorf("checksum calculation failed: %w", err)
	}

	actual := hash.Sum(nil)

	if !equalBytes(actual, expectedBytes) {
		return fmt.Errorf(
			"%w: expected %s, got %s",
			ErrChecksumMismatch,
			hex.EncodeToString(expectedBytes),
			hex.EncodeToString(actual),
		)
	}

	return nil
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}

	var diff byte

	for i := range left {
		diff |= left[i] ^ right[i]
	}

	return diff == 0
}
