package bundle

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gitlab.com/mainops/uniShell/internal/credentials"
)

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
