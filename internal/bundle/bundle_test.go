package bundle

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/mainops/uniShell/internal/credentials"
	"gitlab.com/mainops/uniShell/internal/crypto"
)

func TestOpenAuthenticatedReturnsCompressedPayload(t *testing.T) {
	sourceDir := t.TempDir()

	payload := bytes.Repeat([]byte("runtime payload\n"), 1024)

	if err := os.WriteFile(
		filepath.Join(sourceDir, "runtime"),
		payload,
		0o755,
	); err != nil {
		t.Fatalf("write runtime payload: %v", err)
	}

	bundleData, err := Create(sourceDir, "test-password")
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	authenticated, err := OpenAuthenticated(
		bundleData,
		"test-password",
	)
	if err != nil {
		t.Fatalf(
			"OpenAuthenticated() returned error: %v",
			err,
		)
	}

	if !defaultCompressor.Matches(authenticated) {
		t.Fatal("authenticated payload is not compressed")
	}

	archive, err := DecompressAuthenticated(authenticated)
	if err != nil {
		t.Fatalf(
			"DecompressAuthenticated() returned error: %v",
			err,
		)
	}

	opened, err := Open(bundleData, "test-password")
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}

	if !bytes.Equal(archive, opened) {
		t.Fatal(
			"OpenAuthenticated() and Open() produced different archives",
		)
	}
}

func TestCreateAndOpen(t *testing.T) {
	source := t.TempDir()

	if err := os.Mkdir(
		filepath.Join(source, "bin"),
		0700,
	); err != nil {
		t.Fatalf("create bin directory: %v", err)
	}

	payload := []byte("#!/bin/sh\necho uniShell\n")

	if err := os.WriteFile(
		filepath.Join(source, "bin", "tool"),
		payload,
		0755,
	); err != nil {
		t.Fatalf("write tool: %v", err)
	}

	password := "test-password"

	bundle, err := Create(source, password)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	if len(bundle) == 0 {
		t.Fatal("Create() returned empty bundle")
	}

	archive, err := Open(bundle, password)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}

	destination := filepath.Join(t.TempDir(), "runtime")

	if err := ExtractArchive(archive, destination); err != nil {
		t.Fatalf("ExtractArchive() returned error: %v", err)
	}

	got, err := os.ReadFile(
		filepath.Join(destination, "bin", "tool"),
	)
	if err != nil {
		t.Fatalf("read extracted tool: %v", err)
	}

	if !bytes.Equal(got, payload) {
		t.Fatal("extracted payload does not match original")
	}
}

func TestCreateCompressesRuntimeArchive(t *testing.T) {
	source := t.TempDir()

	payload := bytes.Repeat(
		[]byte("uniShell runtime compression integration test\n"),
		10000,
	)

	if err := os.WriteFile(
		filepath.Join(source, "test"),
		payload,
		0600,
	); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	password := "test-password"

	bundle, err := Create(source, password)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	decrypted, err := crypto.Decrypt(bundle, password)
	if err != nil {
		t.Fatalf("decrypt created bundle: %v", err)
	}

	if !defaultCompressor.Matches(decrypted) {
		t.Fatal("created bundle payload is not Zstandard-compressed")
	}
}

func TestOpenRejectsWrongPassword(t *testing.T) {
	source := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(source, "test"),
		[]byte("secret payload"),
		0600,
	); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	bundle, err := Create(source, "test-password")
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	_, err = Open(bundle, "wrong-password")

	if !errors.Is(err, credentials.ErrAuthenticationFailed) {
		t.Fatalf(
			"Open() error = %v, want authentication failure",
			err,
		)
	}
}

func TestOpenRejectsModifiedBundle(t *testing.T) {
	source := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(source, "test"),
		[]byte("secret payload"),
		0600,
	); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	bundle, err := Create(source, "test-password")
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}

	bundle[len(bundle)-1] ^= 0xff

	_, err = Open(bundle, "test-password")

	if !errors.Is(err, credentials.ErrAuthenticationFailed) {
		t.Fatalf(
			"Open() error = %v, want authentication failure",
			err,
		)
	}
}

func TestOpenSupportsLegacyUncompressedBundle(t *testing.T) {
	source := t.TempDir()

	payload := []byte("legacy runtime payload")

	if err := os.WriteFile(
		filepath.Join(source, "test"),
		payload,
		0600,
	); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	archive, err := CreateArchive(source)
	if err != nil {
		t.Fatalf("CreateArchive() returned error: %v", err)
	}

	password := "test-password"

	encrypted, err := crypto.Encrypt(archive, password)
	if err != nil {
		t.Fatalf("Encrypt() returned error: %v", err)
	}

	opened, err := Open(encrypted, password)
	if err != nil {
		t.Fatalf("Open() returned error: %v", err)
	}

	if !bytes.Equal(opened, archive) {
		t.Fatal("legacy uncompressed bundle was changed while opening")
	}
}

func TestCreateRejectsEmptyPassword(t *testing.T) {
	source := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(source, "test"),
		[]byte("payload"),
		0600,
	); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	_, err := Create(source, "")

	if !errors.Is(err, credentials.ErrEmptyToken) {
		t.Fatalf(
			"Create() error = %v, want empty token error",
			err,
		)
	}
}
