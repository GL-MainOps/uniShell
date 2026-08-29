package shell

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func findEnv(env []string, key string) (string, bool) {
	prefix := key + "="

	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix), true
		}
	}

	return "", false
}

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
	selected := Shell{
		Name:   "bash",
		Path:   "/bin/bash",
		Source: SourceHost,
	}

	command, err := NewCommand(
		selected,
		"/runtime/bin",
		"/runtime/session",
		Startup{},
	)
	if err != nil {
		t.Fatalf("NewCommand() returned error: %v", err)
	}

	if command.Path != selected.Path {
		t.Fatalf(
			"command path = %q, want %q",
			command.Path,
			selected.Path,
		)
	}

	if len(command.Args) != 1 ||
		command.Args[0] != selected.Path {
		t.Fatalf("unexpected command args: %#v", command.Args)
	}

	shellPath, ok := findEnv(command.Env, "SHELL")
	if !ok {
		t.Fatal("SHELL was not configured")
	}

	if shellPath != selected.Path {
		t.Fatalf(
			"SHELL = %q, want %q",
			shellPath,
			selected.Path,
		)
	}

	runtimePath, ok := findEnv(
		command.Env,
		SessionRuntimeDirEnvName,
	)
	if !ok {
		t.Fatal(
			"UNISHELL_SESSION_RUNTIME_DIR was not configured",
		)
	}

	if runtimePath != "/runtime/session" {
		t.Fatalf(
			"session runtime = %q, want %q",
			runtimePath,
			"/runtime/session",
		)
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

func TestResolveExplicitShell(t *testing.T) {
	runtimeBin := t.TempDir()
	bundled := executable(t, runtimeBin, "zsh")

	t.Setenv(ShellEnvName, "fish")
	t.Setenv("SHELL", "/bin/bash")

	got, err := Resolve("zsh", runtimeBin)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got.Name != "zsh" {
		t.Fatalf(
			"shell name = %q, want %q",
			got.Name,
			"zsh",
		)
	}

	if got.Path != bundled {
		t.Fatalf(
			"shell path = %q, want %q",
			got.Path,
			bundled,
		)
	}

	if got.Source != SourceBundled {
		t.Fatalf(
			"shell source = %q, want %q",
			got.Source,
			SourceBundled,
		)
	}
}

func TestResolveUsesUNISHELLShellBeforeSHELL(t *testing.T) {
	runtimeBin := t.TempDir()
	bundled := executable(t, runtimeBin, "fish")

	t.Setenv(ShellEnvName, "fish")
	t.Setenv("SHELL", "/bin/bash")

	got, err := Resolve("", runtimeBin)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got.Name != "fish" {
		t.Fatalf(
			"shell name = %q, want %q",
			got.Name,
			"fish",
		)
	}

	if got.Path != bundled {
		t.Fatalf(
			"shell path = %q, want %q",
			got.Path,
			bundled,
		)
	}

	if got.Source != SourceBundled {
		t.Fatalf(
			"shell source = %q, want %q",
			got.Source,
			SourceBundled,
		)
	}
}

func TestResolveUsesSHELLBasename(t *testing.T) {
	dir := t.TempDir()
	shellPath := executable(t, dir, "zsh")

	t.Setenv(ShellEnvName, "")
	t.Setenv("SHELL", shellPath)
	t.Setenv("PATH", dir)

	got, err := Resolve("", "")
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got.Name != "zsh" {
		t.Fatalf(
			"shell name = %q, want %q",
			got.Name,
			"zsh",
		)
	}

	if got.Path != shellPath {
		t.Fatalf(
			"shell path = %q, want %q",
			got.Path,
			shellPath,
		)
	}

	if got.Source != SourceHost {
		t.Fatalf(
			"shell source = %q, want %q",
			got.Source,
			SourceHost,
		)
	}
}

func TestResolveFallsBackToBashWhenNoShellConfigured(t *testing.T) {
	t.Setenv(ShellEnvName, "")
	t.Setenv("SHELL", "")

	got, err := Resolve("", "")
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got.Name != "bash" {
		t.Fatalf(
			"shell name = %q, want %q",
			got.Name,
			"bash",
		)
	}

	if got.Source != SourceHost {
		t.Fatalf(
			"shell source = %q, want %q",
			got.Source,
			SourceHost,
		)
	}
}

func TestResolvePrefersBundledNonBashShell(t *testing.T) {
	runtimeBin := t.TempDir()
	bundled := executable(t, runtimeBin, "zsh")

	hostDir := t.TempDir()
	host := executable(t, hostDir, "zsh")

	t.Setenv("PATH", hostDir)

	got, err := Resolve("zsh", runtimeBin)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got.Path != bundled {
		t.Fatalf(
			"shell path = %q, want bundled %q",
			got.Path,
			bundled,
		)
	}

	if got.Source != SourceBundled {
		t.Fatalf(
			"shell source = %q, want %q",
			got.Source,
			SourceBundled,
		)
	}

	_ = host
}

func TestResolveFallsBackToHostForNonBashShell(t *testing.T) {
	runtimeBin := t.TempDir()
	hostDir := t.TempDir()
	host := executable(t, hostDir, "fish")

	t.Setenv("PATH", hostDir)

	got, err := Resolve("fish", runtimeBin)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got.Path != host {
		t.Fatalf(
			"shell path = %q, want %q",
			got.Path,
			host,
		)
	}

	if got.Source != SourceHost {
		t.Fatalf(
			"shell source = %q, want %q",
			got.Source,
			SourceHost,
		)
	}
}

func TestResolveBashUsesHostPATH(t *testing.T) {
	runtimeBin := t.TempDir()
	hostDir := t.TempDir()
	host := executable(t, hostDir, "bash")

	t.Setenv("PATH", hostDir)

	got, err := Resolve("bash", runtimeBin)
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}

	if got.Path != host {
		t.Fatalf(
			"shell path = %q, want %q",
			got.Path,
			host,
		)
	}

	if got.Source != SourceHost {
		t.Fatalf(
			"shell source = %q, want %q",
			got.Source,
			SourceHost,
		)
	}
}

func TestResolveRejectsUnsupportedExplicitShell(t *testing.T) {
	_, err := Resolve("ksh", t.TempDir())

	if !errors.Is(err, ErrShellUnsupported) {
		t.Fatalf(
			"Resolve() error = %v, want %v",
			err,
			ErrShellUnsupported,
		)
	}
}

func TestResolveRejectsUnavailableShell(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	_, err := Resolve("fish", t.TempDir())

	if !errors.Is(err, ErrShellUnavailable) {
		t.Fatalf(
			"Resolve() error = %v, want %v",
			err,
			ErrShellUnavailable,
		)
	}
}

func TestResolveExecutableUsesAbsolutePath(t *testing.T) {
	path := executable(t, t.TempDir(), "shell")

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
	path := executable(t, dir, "shell")

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
	t.Setenv(ShellEnvName, "bash")

	env, err := NewEnvironment(
		"/runtime/bin",
		"/runtime/session",
	)
	if err != nil {
		t.Fatalf(
			"NewEnvironment() returned error: %v",
			err,
		)
	}

	path, ok := findEnv(env, "PATH")
	if !ok {
		t.Fatal("PATH was not included")
	}

	want := "/runtime/bin:/usr/bin:/bin"

	if path != want {
		t.Fatalf(
			"PATH = %q, want %q",
			path,
			want,
		)
	}
}

func TestNewEnvironmentIncludesSessionRuntimeDirectory(t *testing.T) {
	t.Setenv(ShellEnvName, "bash")

	env, err := NewEnvironment(
		"/runtime/bin",
		"/runtime/session",
	)
	if err != nil {
		t.Fatalf(
			"NewEnvironment() returned error: %v",
			err,
		)
	}

	runtimePath, ok := findEnv(
		env,
		SessionRuntimeDirEnvName,
	)
	if !ok {
		t.Fatal(
			"UNISHELL_SESSION_RUNTIME_DIR was not included",
		)
	}

	if runtimePath != "/runtime/session" {
		t.Fatalf(
			"session runtime = %q, want %q",
			runtimePath,
			"/runtime/session",
		)
	}
}

func TestNewEnvironmentRejectsEmptyRuntimeBin(t *testing.T) {
	_, err := NewEnvironment(
		"",
		"/runtime/session",
	)

	if err == nil {
		t.Fatal("NewEnvironment() returned nil error")
	}
}

func TestNewEnvironmentRejectsEmptySessionRuntime(t *testing.T) {
	_, err := NewEnvironment(
		"/runtime/bin",
		"",
	)

	if err == nil {
		t.Fatal("NewEnvironment() returned nil error")
	}
}

func TestNewEnvironmentForShellSetsResolvedShell(t *testing.T) {
	selected := Shell{
		Name:   "zsh",
		Path:   "/runtime/bin/zsh",
		Source: SourceBundled,
	}

	env, err := NewEnvironmentForShell(
		"/runtime/bin",
		"/runtime/session",
		selected,
	)
	if err != nil {
		t.Fatalf(
			"NewEnvironmentForShell() returned error: %v",
			err,
		)
	}

	shellPath, ok := findEnv(env, "SHELL")
	if !ok {
		t.Fatal("SHELL was not included")
	}

	if shellPath != selected.Path {
		t.Fatalf(
			"SHELL = %q, want %q",
			shellPath,
			selected.Path,
		)
	}
}

func TestNewEnvironmentForShellRejectsIncompleteShell(t *testing.T) {
	_, err := NewEnvironmentForShell(
		"/runtime/bin",
		"/runtime/session",
		Shell{
			Name: "zsh",
		},
	)

	if err == nil {
		t.Fatal(
			"NewEnvironmentForShell() returned nil error",
		)
	}
}

func TestNewCommandSetsShellEnvironment(t *testing.T) {
	selected := Shell{
		Name:   "bash",
		Path:   "/bin/bash",
		Source: SourceHost,
	}

	command, err := NewCommand(
		selected,
		"/runtime/bin",
		"/runtime/session",
		Startup{},
	)
	if err != nil {
		t.Fatalf(
			"NewCommand() returned error: %v",
			err,
		)
	}

	shellPath, ok := findEnv(command.Env, "SHELL")
	if !ok {
		t.Fatal("SHELL was not configured")
	}

	if shellPath != selected.Path {
		t.Fatalf(
			"SHELL = %q, want %q",
			shellPath,
			selected.Path,
		)
	}
}

func TestNewCommandRejectsEmptyShellPath(t *testing.T) {
	_, err := NewCommand(
		Shell{Name: "bash"},
		"/runtime/bin",
		"/runtime/session",
		Startup{},
	)

	if err == nil {
		t.Fatal("NewCommand() returned nil error")
	}
}
