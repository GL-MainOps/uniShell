package main

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"gitlab.com/mainops/uniShell/internal/credentials"
)

func TestPrintErrorAuthenticationFailed(t *testing.T) {
	originalStderr := os.Stderr

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stderr = writer

	printError(credentials.ErrAuthenticationFailed)

	_ = writer.Close()
	os.Stderr = originalStderr

	var output bytes.Buffer

	_, err = output.ReadFrom(reader)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}

	want := "Authentication Failed. Aborting...\n"

	if output.String() != want {
		t.Fatalf(
			"output = %q, want %q",
			output.String(),
			want,
		)
	}
}

func TestPrintErrorGenericError(t *testing.T) {
	originalStderr := os.Stderr

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stderr = writer

	printError(errors.New("test error"))

	_ = writer.Close()
	os.Stderr = originalStderr

	var output bytes.Buffer

	_, err = output.ReadFrom(reader)
	if err != nil {
		t.Fatalf("read captured stderr: %v", err)
	}

	want := "uniShell: test error\n"

	if output.String() != want {
		t.Fatalf(
			"output = %q, want %q",
			output.String(),
			want,
		)
	}
}
