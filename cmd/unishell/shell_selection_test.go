package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitlab.com/mainops/uniShell/internal/shell"
)

func executable(t *testing.T, dir, name string) string {
	t.Helper()

	path := filepath.Join(dir, name)

	if err := os.WriteFile(
		path,
		[]byte("#!/bin/sh\n"),
		0700,
	); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	return path
}

func TestAvailableShellsPrefersBundledShell(t *testing.T) {
	runtimeBin := t.TempDir()

	executable(t, runtimeBin, "zsh")

	t.Setenv("PATH", t.TempDir())

	available := availableShells(runtimeBin)

	var found bool

	for _, candidate := range available {
		if candidate.Name == "zsh" {
			found = true

			if candidate.Source != shell.SourceBundled {
				t.Fatalf(
					"zsh source = %q, want %q",
					candidate.Source,
					shell.SourceBundled,
				)
			}
		}
	}

	if !found {
		t.Fatal("bundled zsh was not reported as available")
	}
}

func TestSelectShellUsesAvailableRequestedShell(t *testing.T) {
	runtimeBin := t.TempDir()

	executable(t, runtimeBin, "fish")

	ctx := context.Background()

	got, err := selectShell(
		ctx,
		runtimeBin,
		"fish",
		strings.NewReader(""),
		&strings.Builder{},
	)
	if err != nil {
		t.Fatalf(
			"selectShell() returned error: %v",
			err,
		)
	}

	if got != "fish" {
		t.Fatalf(
			"selected shell = %q, want %q",
			got,
			"fish",
		)
	}
}

func TestSelectShellPromptsForUnavailableShell(t *testing.T) {
	runtimeBin := t.TempDir()

	executable(t, runtimeBin, "zsh")

	var output strings.Builder

	got, err := selectShell(
		context.Background(),
		runtimeBin,
		"fish",
		strings.NewReader("1\n"),
		&output,
	)
	if err != nil {
		t.Fatalf(
			"selectShell() returned error: %v",
			err,
		)
	}

	if got != "bash" && got != "zsh" {
		t.Fatalf(
			"selected shell = %q, want an available shell",
			got,
		)
	}

	if !strings.Contains(
		output.String(),
		"shell \"fish\" is unavailable or unsupported",
	) {
		t.Fatalf(
			"prompt output does not explain unavailable shell: %q",
			output.String(),
		)
	}
}

func TestSelectShellRejectsInvalidSelection(t *testing.T) {
	runtimeBin := t.TempDir()

	executable(t, runtimeBin, "zsh")

	var output strings.Builder

	got, err := selectShell(
		context.Background(),
		runtimeBin,
		"fish",
		strings.NewReader("invalid\n2\n"),
		&output,
	)
	if err != nil {
		t.Fatalf(
			"selectShell() returned error: %v",
			err,
		)
	}

	if got != "zsh" {
		t.Fatalf(
			"selected shell = %q, want %q",
			got,
			"zsh",
		)
	}

	if !strings.Contains(
		output.String(),
		"Invalid selection",
	) {
		t.Fatalf(
			"prompt output does not contain invalid-selection message: %q",
			output.String(),
		)
	}
}

func TestSelectShellCanExit(t *testing.T) {
	_, err := selectShell(
		context.Background(),
		t.TempDir(),
		"fish",
		strings.NewReader("q\n"),
		&strings.Builder{},
	)

	if !errors.Is(err, errShellSelectionCancelled) {
		t.Fatalf(
			"selectShell() error = %v, want %v",
			err,
			errShellSelectionCancelled,
		)
	}
}

func TestSelectShellHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := selectShell(
		ctx,
		t.TempDir(),
		"fish",
		strings.NewReader(""),
		&strings.Builder{},
	)

	if !errors.Is(err, errShellSelectionCancelled) {
		t.Fatalf(
			"selectShell() error = %v, want %v",
			err,
			errShellSelectionCancelled,
		)
	}
}
