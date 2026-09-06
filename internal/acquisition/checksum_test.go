package acquisition

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func TestVerifyChecksumAcceptsMatchingSHA256(t *testing.T) {
	content := "uniShell checksum verification"

	sum := sha256.Sum256([]byte(content))
	expected := hex.EncodeToString(sum[:])

	if err := VerifyChecksum(strings.NewReader(content), expected); err != nil {
		t.Fatalf("VerifyChecksum() error = %v", err)
	}
}

func TestVerifyChecksumRejectsMismatch(t *testing.T) {
	content := "uniShell checksum verification"

	err := VerifyChecksum(
		strings.NewReader(content),
		strings.Repeat("0", sha256.Size*2),
	)
	if err == nil {
		t.Fatal("VerifyChecksum() error = nil, want checksum mismatch")
	}

	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("VerifyChecksum() error = %v, want ErrChecksumMismatch", err)
	}
}

func TestVerifyChecksumRejectsInvalidChecksumEncoding(t *testing.T) {
	err := VerifyChecksum(
		strings.NewReader("content"),
		"not-a-hex-checksum",
	)
	if err == nil {
		t.Fatal("VerifyChecksum() error = nil, want invalid checksum error")
	}

	if !errors.Is(err, ErrInvalidResolvedArtifact) {
		t.Fatalf(
			"VerifyChecksum() error = %v, want ErrInvalidResolvedArtifact",
			err,
		)
	}
}

func TestVerifyChecksumRejectsWrongChecksumLength(t *testing.T) {
	err := VerifyChecksum(
		strings.NewReader("content"),
		"00",
	)
	if err == nil {
		t.Fatal("VerifyChecksum() error = nil, want invalid checksum length error")
	}

	if !errors.Is(err, ErrInvalidResolvedArtifact) {
		t.Fatalf(
			"VerifyChecksum() error = %v, want ErrInvalidResolvedArtifact",
			err,
		)
	}
}

func TestVerifyChecksumRejectsNilReader(t *testing.T) {
	err := VerifyChecksum(
		nil,
		strings.Repeat("0", sha256.Size*2),
	)
	if err == nil {
		t.Fatal("VerifyChecksum() error = nil, want nil reader error")
	}

	if !errors.Is(err, ErrInvalidResolvedArtifact) {
		t.Fatalf(
			"VerifyChecksum() error = %v, want ErrInvalidResolvedArtifact",
			err,
		)
	}
}

func TestVerifyChecksumRejectsMissingChecksum(t *testing.T) {
	err := VerifyChecksum(
		strings.NewReader("content"),
		"",
	)
	if err == nil {
		t.Fatal("VerifyChecksum() error = nil, want missing checksum error")
	}

	if !errors.Is(err, ErrInvalidResolvedArtifact) {
		t.Fatalf(
			"VerifyChecksum() error = %v, want ErrInvalidResolvedArtifact",
			err,
		)
	}
}
