package persistence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWritesPrivateDocumentedConfig(t *testing.T) {
	root := filepath.Join(t.TempDir(), "runtime")
	path, err := Create(root)
	if err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}
	if path != ConfigPath(root) {
		t.Fatalf("Create() path = %q, want %q", path, ConfigPath(root))
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(config) returned error: %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("config permissions = %04o, want 0600", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(config) returned error: %v", err)
	}
	for _, option := range []string{"# shell =", "# shell_profile =", "# multiplexer =", "[launch]"} {
		if !strings.Contains(string(data), option) {
			t.Errorf("config does not contain %q", option)
		}
	}
}

func TestSaveFirstLaunchPersistsOptionsOnlyOnce(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(root); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}
	first := LaunchConfig{
		Shell:        "fish",
		ShellProfile: "work",
		Multiplexer:  "tmux",
		NewSession:   true,
	}
	if err := SaveFirstLaunch(root, first); err != nil {
		t.Fatalf("SaveFirstLaunch() returned error: %v", err)
	}
	got, err := Read(root)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}
	if !got.Launch.Configured || got.Launch.Shell != first.Shell || got.Launch.ShellProfile != first.ShellProfile || got.Launch.Multiplexer != first.Multiplexer || !got.Launch.NewSession {
		t.Fatalf("saved launch config = %#v, want first launch %#v", got.Launch, first)
	}

	if err := SaveFirstLaunch(root, LaunchConfig{Shell: "bash"}); err != nil {
		t.Fatalf("second SaveFirstLaunch() returned error: %v", err)
	}
	got, err = Read(root)
	if err != nil {
		t.Fatalf("Read() after second save returned error: %v", err)
	}
	if got.Launch.Shell != first.Shell {
		t.Fatalf("second launch overwrote configured shell: got %q, want %q", got.Launch.Shell, first.Shell)
	}
}

func TestReadRejectsUnknownOptionsWithGuidance(t *testing.T) {
	root := t.TempDir()
	data := []byte("[launch]\nshel = \"bash\"\n")
	if err := os.WriteFile(ConfigPath(root), data, 0600); err != nil {
		t.Fatalf("WriteFile(config) returned error: %v", err)
	}
	_, err := Read(root)
	if err == nil || !strings.Contains(err.Error(), "setting names") {
		t.Fatalf("Read() error = %v, want unknown-setting guidance", err)
	}
}

func TestReadExplainsUnsupportedShell(t *testing.T) {
	root := t.TempDir()
	data := []byte("[launch]\nshell = \"csh\"\n")
	if err := os.WriteFile(ConfigPath(root), data, 0600); err != nil {
		t.Fatalf("WriteFile(config) returned error: %v", err)
	}
	_, err := Read(root)
	if err == nil || !strings.Contains(err.Error(), "[launch].shell") || !strings.Contains(err.Error(), "nushell") {
		t.Fatalf("Read() error = %v, want supported-shell guidance", err)
	}
}

func TestReadTracksExplicitBooleanFalse(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(root); err != nil {
		t.Fatalf("Create() returned error: %v", err)
	}
	config, err := Read(root)
	if err != nil {
		t.Fatalf("Read() returned error: %v", err)
	}
	if !config.Launch.NoSharedRCSet || config.Launch.NoSharedRC {
		t.Fatalf("no_shared_rc presence/value = %t/%t, want explicitly set false", config.Launch.NoSharedRCSet, config.Launch.NoSharedRC)
	}
}
