package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPATHPrependsRuntimeBin(t *testing.T) {
	got := buildPATH(
		"/runtime/bin",
		"/usr/bin:/bin",
	)

	want := "/runtime/bin" +
		string(os.PathListSeparator) +
		"/usr/bin:/bin"

	if got != want {
		t.Fatalf("buildPATH() = %q, want %q", got, want)
	}
}

func TestBuildPATHHandlesEmptyExistingPath(t *testing.T) {
	got := buildPATH("/runtime/bin", "")

	if got != "/runtime/bin" {
		t.Fatalf(
			"buildPATH() = %q, want %q",
			got,
			"/runtime/bin",
		)
	}
}

func TestSetEnvironmentReplacesExistingValue(t *testing.T) {
	env := []string{
		"PATH=/usr/bin",
		"HOME=/home/test",
	}

	got := setEnvironment(env, "PATH", "/runtime/bin")

	var paths []string

	for _, entry := range got {
		if strings.HasPrefix(entry, "PATH=") {
			paths = append(paths, entry)
		}
	}

	if len(paths) != 1 {
		t.Fatalf(
			"PATH entries = %d, want 1",
			len(paths),
		)
	}

	if paths[0] != "PATH=/runtime/bin" {
		t.Fatalf(
			"PATH = %q, want %q",
			paths[0],
			"PATH=/runtime/bin",
		)
	}
}

func TestSetEnvironmentAddsMissingValue(t *testing.T) {
	env := []string{
		"HOME=/home/test",
	}

	got := setEnvironment(env, "PATH", "/runtime/bin")

	found := false

	for _, entry := range got {
		if entry == "PATH=/runtime/bin" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("PATH was not added")
	}
}

func TestNewCommandBuildsRuntimeEnvironment(t *testing.T) {
	command, err := NewCommand(
		"/bin/sh",
		"/runtime/bin",
	)
	if err != nil {
		t.Fatalf("NewCommand() returned error: %v", err)
	}

	if command.Path != "/bin/sh" {
		t.Fatalf(
			"command path = %q, want %q",
			command.Path,
			"/bin/sh",
		)
	}

	if len(command.Args) != 1 ||
		command.Args[0] != "/bin/sh" {
		t.Fatalf("unexpected command args: %#v", command.Args)
	}

	foundPath := false

	for _, entry := range command.Env {
		if entry == "PATH=/runtime/bin:"+os.Getenv("PATH") {
			foundPath = true
			break
		}
	}

	if !foundPath {
		t.Fatal("runtime PATH was not configured")
	}
}

func TestIsExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool")

	if err := os.WriteFile(path, []byte("test"), 0700); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	if !isExecutable(path) {
		t.Fatal("expected executable file to be detected")
	}
}

func TestIsExecutableRejectsNonExecutable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool")

	if err := os.WriteFile(path, []byte("test"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if isExecutable(path) {
		t.Fatal("non-executable file was detected as executable")
	}
}

func TestIsExecutableRejectsDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tool")

	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatalf("create directory: %v", err)
	}

	if isExecutable(path) {
		t.Fatal("directory was detected as executable")
	}
}

func TestResolveUsesConfiguredShell(t *testing.T) {
	shellPath := filepath.Join(t.TempDir(), "custom-shell")

	if err := os.WriteFile(
		shellPath,
		[]byte("#!/bin/sh\n"),
		0700,
	); err != nil {
		t.Fatalf("write shell: %v", err)
	}

	t.Setenv("SHELL", shellPath)

	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	want, err := filepath.Abs(shellPath)
	if err != nil {
		t.Fatalf("resolve test path: %v", err)
	}

	if got != want {
		t.Fatalf(
			"Resolve() = %q, want %q",
			got,
			want,
		)
	}
}

func TestResolveUsesConfiguredShellFromPATH(t *testing.T) {
	dir := t.TempDir()
	shellPath := filepath.Join(dir, "custom-shell")

	if err := os.WriteFile(
		shellPath,
		[]byte("#!/bin/sh\n"),
		0700,
	); err != nil {
		t.Fatalf("write shell: %v", err)
	}

	t.Setenv("PATH", dir)
	t.Setenv("SHELL", "custom-shell")

	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got != shellPath {
		t.Fatalf(
			"Resolve() = %q, want %q",
			got,
			shellPath,
		)
	}
}

func TestResolveFallsBackToAvailableShellFromPATH(t *testing.T) {
	dir := t.TempDir()

	fallback := filepath.Join(dir, "fish")

	if err := os.WriteFile(
		fallback,
		[]byte("#!/bin/sh\n"),
		0700,
	); err != nil {
		t.Fatalf("write fallback shell: %v", err)
	}

	t.Setenv("SHELL", filepath.Join(dir, "missing-shell"))
	t.Setenv("PATH", dir)

	got, err := Resolve()
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got != fallback {
		t.Fatalf(
			"Resolve() = %q, want %q",
			got,
			fallback,
		)
	}
}

func TestResolveRejectsUnavailableConfiguredShell(
	t *testing.T,
) {
	t.Setenv(
		"SHELL",
		filepath.Join(t.TempDir(), "missing-shell"),
	)

	// Prevent the normal system fallback shells from being found.
	t.Setenv("PATH", t.TempDir())

	_, err := Resolve()

	if err != ErrShellUnavailable {
		t.Fatalf(
			"Resolve() error = %v, want %v",
			err,
			ErrShellUnavailable,
		)
	}
}

func TestResolveExecutableUsesAbsolutePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shell")

	if err := os.WriteFile(
		path,
		[]byte("#!/bin/sh\n"),
		0700,
	); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	got, ok := resolveExecutable(path)

	if !ok {
		t.Fatal("resolveExecutable() rejected executable")
	}

	want, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("resolve test path: %v", err)
	}

	if got != want {
		t.Fatalf(
			"resolveExecutable() = %q, want %q",
			got,
			want,
		)
	}
}

func TestResolveExecutableUsesPATH(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shell")

	if err := os.WriteFile(
		path,
		[]byte("#!/bin/sh\n"),
		0700,
	); err != nil {
		t.Fatalf("write executable: %v", err)
	}

	t.Setenv("PATH", dir)

	got, ok := resolveExecutable("shell")

	if !ok {
		t.Fatal("resolveExecutable() rejected PATH executable")
	}

	if got != path {
		t.Fatalf(
			"resolveExecutable() = %q, want %q",
			got,
			path,
		)
	}
}

func TestNewEnvironmentIncludesRuntimePath(t *testing.T) {
	t.Setenv("PATH", "/usr/bin:/bin")

	env, err := NewEnvironment("/runtime/bin")
	if err != nil {
		t.Fatalf(
			"NewEnvironment() returned error: %v",
			err,
		)
	}

	found := false

	for _, entry := range env {
		if entry == "PATH=/runtime/bin:/usr/bin:/bin" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal(
			"runtime PATH was not included in environment",
		)
	}
}

func TestNewEnvironmentSetsResolvedShell(t *testing.T) {
	shellPath := filepath.Join(t.TempDir(), "shell")

	if err := os.WriteFile(
		shellPath,
		[]byte("#!/bin/sh\n"),
		0700,
	); err != nil {
		t.Fatalf("write shell: %v", err)
	}

	t.Setenv("SHELL", shellPath)

	env, err := NewEnvironment("/runtime/bin")
	if err != nil {
		t.Fatalf(
			"NewEnvironment() returned error: %v",
			err,
		)
	}

	found := false

	for _, entry := range env {
		if entry == "SHELL="+shellPath {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf(
			"resolved SHELL was not included in environment",
		)
	}
}

func TestNewEnvironmentFallsBackWhenConfiguredShellIsUnavailable(
	t *testing.T,
) {
	shellPath := filepath.Join(
		t.TempDir(),
		"missing-shell",
	)

	t.Setenv("SHELL", shellPath)

	env, err := NewEnvironment("/runtime/bin")
	if err != nil {
		t.Fatalf(
			"NewEnvironment() returned error: %v",
			err,
		)
	}

	found := false

	for _, entry := range env {
		if strings.HasPrefix(entry, "SHELL=") {
			found = true

			if entry == "SHELL="+shellPath {
				t.Fatalf(
					"unavailable configured shell was retained: %q",
					entry,
				)
			}

			break
		}
	}

	if !found {
		t.Fatal("NewEnvironment() did not include SHELL")
	}
}

func TestNewCommandSetsShellEnvironment(t *testing.T) {
	command, err := NewCommand(
		"/bin/bash",
		"/runtime/bin",
	)
	if err != nil {
		t.Fatalf(
			"NewCommand() returned error: %v",
			err,
		)
	}

	found := false

	for _, entry := range command.Env {
		if entry == "SHELL=/bin/bash" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("SHELL was not configured")
	}
}
