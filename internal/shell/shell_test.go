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
